package keeper

import (
	"context"
	"fmt"
	"time"

	"cosmossdk.io/core/store"
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	appparams "github.com/sourcenetwork/sourcehub/app/params"

	"github.com/sourcenetwork/sourcehub/x/tier/types"
)

type (
	Keeper struct {
		cdc codec.BinaryCodec

		storeService store.KVStoreService
		logger       log.Logger

		// the address capable of executing a MsgUpdateParams message.
		// Typically, this should be the x/gov module account.
		authority string

		bankKeeper         types.BankKeeper
		stakingKeeper      types.StakingKeeper
		epochsKeeper       types.EpochsKeeper
		distributionKeeper types.DistributionKeeper
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,

	storeService store.KVStoreService,
	logger log.Logger,
	authority string,

	bankKeeper types.BankKeeper,
	stakingKeeper types.StakingKeeper,
	epochsKeeper types.EpochsKeeper,
	distributionKeeper types.DistributionKeeper,
) Keeper {
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address: %s", authority))
	}

	return Keeper{
		cdc:          cdc,
		storeService: storeService,
		authority:    authority,
		logger:       logger,

		bankKeeper:         bankKeeper,
		stakingKeeper:      stakingKeeper,
		epochsKeeper:       epochsKeeper,
		distributionKeeper: distributionKeeper,
	}
}

// GetAuthority returns the module's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// GetStakingKeeper returns the module's StakingKeeper.
func (k Keeper) GetStakingKeeper() types.StakingKeeper {
	return k.stakingKeeper
}

// GetBankKeeper returns the module's BankKeeper.
func (k Keeper) GetBankKeeper() types.BankKeeper {
	return k.bankKeeper
}

// Logger returns a module-specific logger.
func (k Keeper) Logger() log.Logger {
	return k.logger.With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// CompleteUnlocking completes the unlocking process for all lockups that have reached their unlock time.
// It is called at the end of each Epoch.
func (k Keeper) CompleteUnlocking(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	cb := func(delAddr sdk.AccAddress, valAddr sdk.ValAddress, creationHeight int64, lockup types.Lockup) error {
		if sdk.UnwrapSDKContext(ctx).BlockTime().Before(*lockup.UnlockTime) {
			fmt.Printf("Unlock time not reached for %s/%s\n", delAddr, valAddr)
			return nil
		}

		// Redeem the unlocked lockup for stake.
		stake := sdk.NewCoin(appparams.DefaultBondDenom, lockup.Amount)
		coins := sdk.NewCoins(stake)

		moduleBalance := k.bankKeeper.GetBalance(ctx, authtypes.NewModuleAddress(types.ModuleName), appparams.DefaultBondDenom)
		if moduleBalance.Amount.LT(lockup.Amount) {
			fmt.Printf("Module account balance %s is smaller than required amount %s\n", delAddr, valAddr)
			return nil
		}

		err := k.bankKeeper.UndelegateCoinsFromModuleToAccount(ctx, types.ModuleName, delAddr, coins)
		if err != nil {
			return errorsmod.Wrapf(err, "undelegate coins to %s for amount %s", delAddr, stake)
		}

		k.removeUnlockingLockup(ctx, delAddr, valAddr, creationHeight)

		sdkCtx.EventManager().EmitEvent(
			sdk.NewEvent(
				types.EventTypeCompleteUnlocking,
				sdk.NewAttribute(stakingtypes.AttributeKeyDelegator, delAddr.String()),
				sdk.NewAttribute(stakingtypes.AttributeKeyValidator, valAddr.String()),
				sdk.NewAttribute(sdk.AttributeKeyAmount, lockup.Amount.String()),
				sdk.NewAttribute(types.AttributeKeyCreationHeight, fmt.Sprintf("%d", creationHeight)),
			),
		)

		return nil
	}

	err := k.IterateLockups(ctx, true, cb)
	if err != nil {
		return errorsmod.Wrap(err, "iterate unlocking lockups")
	}

	return nil
}

// Lock locks the stake of a delegator to a validator.
func (k Keeper) Lock(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, amt math.Int) error {
	// specified amt must be a positive integer
	if !amt.IsPositive() {
		return types.ErrInvalidAmount.Wrap("invalid amount")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	modAddr := authtypes.NewModuleAddress(types.ModuleName)

	validator, err := k.stakingKeeper.GetValidator(ctx, valAddr)
	if err != nil {
		return types.ErrInvalidAddress.Wrapf("validator address %s: %s", valAddr, err)
	}

	// Move the stake from delegator to the module
	stake := sdk.NewCoin(appparams.DefaultBondDenom, amt)
	coins := sdk.NewCoins(stake)
	err = k.bankKeeper.DelegateCoinsFromAccountToModule(ctx, delAddr, types.ModuleName, coins)
	if err != nil {
		return errorsmod.Wrapf(err, "delegate %s from account to module", stake)
	}

	// Delegate the stake to the validator.
	_, err = k.stakingKeeper.Delegate(ctx, modAddr, stake.Amount, stakingtypes.Unbonded, validator, true)
	if err != nil {
		return errorsmod.Wrapf(err, "delegate %s", stake)
	}

	// Record the lockup
	k.AddLockup(ctx, delAddr, valAddr, stake.Amount)

	// Mint credits
	creditAmt := k.proratedCredit(ctx, delAddr, amt)
	err = k.MintCredit(ctx, delAddr, creditAmt)
	if err != nil {
		return errorsmod.Wrap(err, "mint credit")
	}

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeLock,
			sdk.NewAttribute(stakingtypes.AttributeKeyDelegator, delAddr.String()),
			sdk.NewAttribute(stakingtypes.AttributeKeyValidator, valAddr.String()),
			sdk.NewAttribute(sdk.AttributeKeyAmount, amt.String()),
		),
	)

	return nil
}

// Unlock initiates the unlocking of stake of a delegator from a validator.
// The stake will be unlocked after the unlocking period has passed.
func (k Keeper) Unlock(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, amt math.Int) (
	unbondTime time.Time, unlockTime time.Time, creationHeight int64, err error) {

	// specified amt must be a positive integer
	if !amt.IsPositive() {
		return time.Time{}, time.Time{}, 0, types.ErrInvalidAmount.Wrap("invalid amount")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	modAddr := authtypes.NewModuleAddress(types.ModuleName)

	validator, err := k.stakingKeeper.GetValidator(ctx, valAddr)
	if err != nil {
		return time.Time{}, time.Time{}, 0, types.ErrInvalidAddress.Wrapf("validator address %s: %s", valAddr, err)
	}

	err = k.SubtractLockup(ctx, delAddr, valAddr, amt)
	if err != nil {
		return time.Time{}, time.Time{}, 0, errorsmod.Wrap(err, "subtract lockup")
	}

	shares, err := k.stakingKeeper.ValidateUnbondAmount(ctx, modAddr, valAddr, amt)
	if err != nil {
		return time.Time{}, time.Time{}, 0, errorsmod.Wrap(err, "validate unbond amount")
	}

	if shares.IsZero() {
		return time.Time{}, time.Time{}, 0, errorsmod.Wrap(stakingtypes.ErrInsufficientShares, "calculated shares are zero")
	}

	// adjust token amount to match the actual undelegated tokens
	tokenAmount := validator.TokensFromSharesTruncated(shares).TruncateInt()
	if tokenAmount.LT(amt) {
		amt = tokenAmount
	}

	unbondTime, _, err = k.stakingKeeper.Undelegate(ctx, modAddr, valAddr, shares)
	if err != nil {
		return time.Time{}, time.Time{}, 0, errorsmod.Wrap(err, "undelegate")
	}

	creationHeight, _, unlockTime = k.SetLockup(ctx, true, delAddr, valAddr, amt, &unbondTime)

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeUnlock,
			sdk.NewAttribute(stakingtypes.AttributeKeyDelegator, delAddr.String()),
			sdk.NewAttribute(stakingtypes.AttributeKeyValidator, valAddr.String()),
			sdk.NewAttribute(sdk.AttributeKeyAmount, amt.String()),
			sdk.NewAttribute(types.AttributeKeyUnbondTime, unbondTime.String()),
			sdk.NewAttribute(types.AttributeKeyUnlockTime, unlockTime.String()),
			sdk.NewAttribute(types.AttributeKeyCreationHeight, fmt.Sprintf("%d", creationHeight)),
		),
	)

	return unbondTime, unlockTime, creationHeight, nil
}

// Redelegate redelegates the stake of a delegator from a source validator to a destination validator.
// The redelegation will be completed after the unbonding period has passed.
func (k Keeper) Redelegate(ctx context.Context, delAddr sdk.AccAddress, srcValAddr, dstValAddr sdk.ValAddress, amt math.Int) (
	completionTime time.Time, err error) {

	// specified amt must be a positive integer
	if !amt.IsPositive() {
		return time.Time{}, types.ErrInvalidAmount.Wrap("invalid amount")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	modAddr := authtypes.NewModuleAddress(types.ModuleName)

	err = k.SubtractLockup(ctx, delAddr, srcValAddr, amt)
	if err != nil {
		return time.Time{}, errorsmod.Wrap(err, "subtract locked stake from source validator")
	}

	k.AddLockup(ctx, delAddr, dstValAddr, amt)

	shares, err := k.stakingKeeper.ValidateUnbondAmount(ctx, modAddr, srcValAddr, amt)
	if err != nil {
		return time.Time{}, errorsmod.Wrap(err, "validate unbond amount")
	}

	completionTime, err = k.stakingKeeper.BeginRedelegation(ctx, modAddr, srcValAddr, dstValAddr, shares)
	if err != nil {
		return time.Time{}, errorsmod.Wrap(err, "begin redelegation")
	}

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeRedelegate,
			sdk.NewAttribute(stakingtypes.AttributeKeyDelegator, delAddr.String()),
			sdk.NewAttribute(types.AttributeKeySourceValidator, srcValAddr.String()),
			sdk.NewAttribute(types.AttributeKeyDestinationValidator, dstValAddr.String()),
			sdk.NewAttribute(sdk.AttributeKeyAmount, amt.String()),
			sdk.NewAttribute(types.AttributeKeyCompletionTime, completionTime.String()),
		),
	)

	return completionTime, nil
}

// CancelUnlocking effectively cancels the pending unlocking lockup partially or in full.
// Reverts the specified amt if a valid value is provided (e.g. amt != nil && 0 < amt < unbondEntry.Balance).
// Otherwise, cancels unlocking lockup record in full (e.g. unbondEntry.Balance).
func (k Keeper) CancelUnlocking(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, creationHeight int64, amt *math.Int) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	modAddr := authtypes.NewModuleAddress(types.ModuleName)

	validator, err := k.stakingKeeper.GetValidator(ctx, valAddr)
	if err != nil {
		return types.ErrInvalidAddress.Wrapf("validator address %s: %s", valAddr, err)
	}

	ubd, err := k.stakingKeeper.GetUnbondingDelegation(ctx, modAddr, valAddr)
	if err != nil {
		return errorsmod.Wrapf(err, "unbonding delegation not found for delegator %s and validator %s", modAddr, valAddr)
	}

	// find unbonding delegation entry by CreationHeight
	// TODO: handle edge case with 2+ messages at the same height
	var (
		unbondEntryIndex int64 = -1
		unbondEntry      stakingtypes.UnbondingDelegationEntry
	)

	for i, entry := range ubd.Entries {
		if entry.CreationHeight == creationHeight && entry.CompletionTime.After(sdkCtx.BlockTime()) {
			unbondEntryIndex = int64(i)
			unbondEntry = entry
			break
		}
	}

	if unbondEntryIndex == -1 {
		return errorsmod.Wrapf(
			stakingtypes.ErrNoUnbondingDelegation,
			"no valid unbonding entry found for creation height %d",
			creationHeight,
		)
	}

	// revert the specified amt if set and is positive, otherwise revert the entire UnbondingDelegationEntry
	restoreAmount := unbondEntry.Balance
	if amt != nil && amt.IsPositive() && amt.LT(unbondEntry.Balance) {
		restoreAmount = *amt
	}

	_, err = k.stakingKeeper.Delegate(ctx, modAddr, restoreAmount, stakingtypes.Unbonding, validator, false)
	if err != nil {
		return errorsmod.Wrap(err, "failed to delegate tokens back to validator")
	}

	// update or remove the unbonding delegation entry
	remainingBalance := unbondEntry.Balance.Sub(restoreAmount)
	if remainingBalance.IsZero() {
		ubd.RemoveEntry(unbondEntryIndex)
	} else {
		unbondEntry.Balance = remainingBalance
		unbondEntry.InitialBalance = unbondEntry.InitialBalance.Sub(restoreAmount)
		ubd.Entries[unbondEntryIndex] = unbondEntry
	}

	// update or remove the unbonding delegation in the store
	if len(ubd.Entries) == 0 {
		err = k.stakingKeeper.RemoveUnbondingDelegation(ctx, ubd)
	} else {
		err = k.stakingKeeper.SetUnbondingDelegation(ctx, ubd)
	}
	if err != nil {
		return errorsmod.Wrap(err, "failed to update unbonding delegation")
	}

	// remove unlocking lockup if no amt was specified (e.g. no partial unlocking lockup cancelation)
	k.SubtractUnlockingLockup(ctx, delAddr, valAddr, creationHeight, restoreAmount)

	// add restoreAmount back to the lockup (without modifying the unlock/unbond times)
	k.AddLockup(ctx, delAddr, valAddr, restoreAmount)

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeCancelUnlocking,
			sdk.NewAttribute(stakingtypes.AttributeKeyDelegator, delAddr.String()),
			sdk.NewAttribute(stakingtypes.AttributeKeyValidator, valAddr.String()),
			sdk.NewAttribute(sdk.AttributeKeyAmount, restoreAmount.String()),
			sdk.NewAttribute(types.AttributeKeyCreationHeight, fmt.Sprintf("%d", creationHeight)),
		),
	)

	return nil
}
