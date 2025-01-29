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

// GetEpochsKeeper returns the module's EpochsKeeper.
func (k Keeper) GetEpochsKeeper() types.EpochsKeeper {
	return k.epochsKeeper
}

// GetDistributionKeeper returns the module's DistributionKeeper.
func (k Keeper) GetDistributionKeeper() types.DistributionKeeper {
	return k.distributionKeeper
}

// Logger returns a module-specific logger.
func (k Keeper) Logger() log.Logger {
	return k.logger.With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// CompleteUnlocking completes the unlocking process for all lockups that have reached their unlock time.
// It is called at the end of each Epoch.
func (k Keeper) CompleteUnlocking(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	cb := func(delAddr sdk.AccAddress, valAddr sdk.ValAddress, creationHeight int64, unlockingLockup types.UnlockingLockup) error {
		// Check both CompletionTime and UnlockTime so that CompleteUnlocking works correctly regardless of the
		// staking module Undelegate/Redelegate completion time and tier module EpochDuration/UnlockingEpochs params
		if sdkCtx.BlockTime().Before(unlockingLockup.CompletionTime) || sdkCtx.BlockTime().Before(unlockingLockup.UnlockTime) {
			fmt.Printf("Unlock time not reached for %s/%s\n", delAddr, valAddr)
			return nil
		}

		moduleBalance := k.bankKeeper.GetBalance(ctx, authtypes.NewModuleAddress(types.ModuleName), appparams.DefaultBondDenom)
		if moduleBalance.Amount.LT(unlockingLockup.Amount) {
			fmt.Printf("Module account balance is less than required amount. delAddr: %s, valAddr: %s\n", delAddr, valAddr)
			return nil
		}

		// Redeem the unlocked lockup for stake.
		coins := sdk.NewCoins(sdk.NewCoin(appparams.DefaultBondDenom, unlockingLockup.Amount))
		err := k.bankKeeper.UndelegateCoinsFromModuleToAccount(ctx, types.ModuleName, delAddr, coins)
		if err != nil {
			return errorsmod.Wrapf(err, "undelegate coins to %s for amount %s", delAddr, coins)
		}

		k.removeUnlockingLockup(ctx, delAddr, valAddr, creationHeight)

		sdkCtx.EventManager().EmitEvent(
			sdk.NewEvent(
				types.EventTypeCompleteUnlocking,
				sdk.NewAttribute(stakingtypes.AttributeKeyDelegator, delAddr.String()),
				sdk.NewAttribute(stakingtypes.AttributeKeyValidator, valAddr.String()),
				sdk.NewAttribute(types.AttributeKeyCreationHeight, fmt.Sprintf("%d", creationHeight)),
				sdk.NewAttribute(sdk.AttributeKeyAmount, unlockingLockup.Amount.String()),
			),
		)

		return nil
	}

	err := k.IterateUnlockingLockups(ctx, cb)
	if err != nil {
		return errorsmod.Wrap(err, "iterate unlocking lockups")
	}

	return nil
}

// Lock locks the stake of a delegator to a validator.
func (k Keeper) Lock(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, amt math.Int) error {
	// Specified amt must be a positive integer
	if !amt.IsPositive() {
		return types.ErrInvalidAmount.Wrap("lock non-positive amount")
	}

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
	modAddr := authtypes.NewModuleAddress(types.ModuleName)
	_, err = k.stakingKeeper.Delegate(ctx, modAddr, stake.Amount, stakingtypes.Unbonded, validator, true)
	if err != nil {
		return errorsmod.Wrapf(err, "delegate %s", stake)
	}

	// Record the lockup
	k.AddLockup(ctx, delAddr, valAddr, stake.Amount)

	// Mint credits
	creditAmt := k.proratedCredit(ctx, delAddr, amt)
	err = k.mintCredit(ctx, delAddr, creditAmt)
	if err != nil {
		return errorsmod.Wrap(err, "mint credit")
	}

	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeLock,
			sdk.NewAttribute(stakingtypes.AttributeKeyDelegator, delAddr.String()),
			sdk.NewAttribute(stakingtypes.AttributeKeyValidator, valAddr.String()),
			sdk.NewAttribute(sdk.AttributeKeyAmount, amt.String()),
		),
	)

	return nil
}

// Unlock initiates the unlocking of the specified lockup amount of a delegator from a validator.
// The specified lockup amount will be unlocked in CompleteUnlocking after the unlocking period has passed.
func (k Keeper) Unlock(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress,
	amt math.Int) (creationHeight int64, completionTime, unlockTime time.Time, err error) {
	// Specified amt must be a positive integer
	if !amt.IsPositive() {
		return 0, time.Time{}, time.Time{}, types.ErrInvalidAmount.Wrap("unlock non-positive amount")
	}

	err = k.SubtractLockup(ctx, delAddr, valAddr, amt)
	if err != nil {
		return 0, time.Time{}, time.Time{}, errorsmod.Wrap(err, "subtract lockup")
	}

	modAddr := authtypes.NewModuleAddress(types.ModuleName)
	shares, err := k.stakingKeeper.ValidateUnbondAmount(ctx, modAddr, valAddr, amt)
	if err != nil {
		return 0, time.Time{}, time.Time{}, errorsmod.Wrap(err, "validate unbond amount")
	}
	if !shares.IsPositive() {
		return 0, time.Time{}, time.Time{}, errorsmod.Wrap(stakingtypes.ErrInsufficientShares, "shares are not positive")
	}

	completionTime, returnAmount, err := k.stakingKeeper.Undelegate(ctx, modAddr, valAddr, shares)
	if err != nil {
		return 0, time.Time{}, time.Time{}, errorsmod.Wrap(err, "undelegate")
	}

	// Adjust token amount to match the actual undelegated tokens
	if returnAmount.LT(amt) {
		amt = returnAmount
	}

	params := k.GetParams(ctx)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	creationHeight = sdkCtx.BlockHeight()
	unlockTime = sdkCtx.BlockTime().Add(*params.EpochDuration * time.Duration(params.UnlockingEpochs))

	// Create unlocking lockup record at the current block height
	k.SetUnlockingLockup(ctx, delAddr, valAddr, creationHeight, amt, completionTime, unlockTime)

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeUnlock,
			sdk.NewAttribute(stakingtypes.AttributeKeyDelegator, delAddr.String()),
			sdk.NewAttribute(stakingtypes.AttributeKeyValidator, valAddr.String()),
			sdk.NewAttribute(types.AttributeKeyCreationHeight, fmt.Sprintf("%d", creationHeight)),
			sdk.NewAttribute(sdk.AttributeKeyAmount, amt.String()),
			sdk.NewAttribute(types.AttributeKeyCompletionTime, completionTime.String()),
			sdk.NewAttribute(types.AttributeKeyUnlockTime, unlockTime.String()),
		),
	)

	return creationHeight, completionTime, unlockTime, nil
}

// Redelegate redelegates the stake of a delegator from a source validator to a destination validator.
// The redelegation will be completed after the unbonding period has passed (e.g. at completionTime).
func (k Keeper) Redelegate(ctx context.Context, delAddr sdk.AccAddress, srcValAddr, dstValAddr sdk.ValAddress,
	amt math.Int) (time.Time, error) {
	// Specified amt must be a positive integer
	if !amt.IsPositive() {
		return time.Time{}, types.ErrInvalidAmount.Wrap("redelegate non-positive amount")
	}

	// Subtract the lockup from the source validator
	err := k.SubtractLockup(ctx, delAddr, srcValAddr, amt)
	if err != nil {
		return time.Time{}, errorsmod.Wrap(err, "subtract lockup from source validator")
	}

	// Add the lockup to the destination validator
	k.AddLockup(ctx, delAddr, dstValAddr, amt)

	modAddr := authtypes.NewModuleAddress(types.ModuleName)
	shares, err := k.stakingKeeper.ValidateUnbondAmount(ctx, modAddr, srcValAddr, amt)
	if err != nil {
		return time.Time{}, errorsmod.Wrap(err, "validate unbond amount")
	}

	completionTime, err := k.stakingKeeper.BeginRedelegation(ctx, modAddr, srcValAddr, dstValAddr, shares)
	if err != nil {
		return time.Time{}, errorsmod.Wrap(err, "begin redelegation")
	}

	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(
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
// Reverts the specified amt if a valid value is provided (e.g. 0 < amt < unlocking lockup amount).
func (k Keeper) CancelUnlocking(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress,
	creationHeight int64, amt math.Int) error {
	// Specified amt must be a positive integer
	if !amt.IsPositive() {
		return types.ErrInvalidAmount.Wrap("cancel unlocking non-positive amount")
	}

	validator, err := k.stakingKeeper.GetValidator(ctx, valAddr)
	if err != nil {
		return types.ErrInvalidAddress.Wrapf("validator address %s: %s", valAddr, err)
	}

	modAddr := authtypes.NewModuleAddress(types.ModuleName)
	ubd, err := k.stakingKeeper.GetUnbondingDelegation(ctx, modAddr, valAddr)
	if err != nil {
		return errorsmod.Wrapf(err, "unbonding delegation not found for delegator %s and validator %s", modAddr, valAddr)
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Find unbonding delegation entry by CreationHeight
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
		return errorsmod.Wrapf(stakingtypes.ErrNoUnbondingDelegation, "no valid entry for height %d", creationHeight)
	}

	if amt.GT(unbondEntry.Balance) {
		return types.ErrInvalidAmount.Wrap("cancel unlocking amount exceeds unbonding entry balance")
	}

	_, err = k.stakingKeeper.Delegate(ctx, modAddr, amt, stakingtypes.Unbonding, validator, false)
	if err != nil {
		return errorsmod.Wrap(err, "failed to delegate tokens back to validator")
	}

	// Update or remove the unbonding delegation entry
	remainingBalance := unbondEntry.Balance.Sub(amt)
	if remainingBalance.IsZero() {
		ubd.RemoveEntry(unbondEntryIndex)
	} else {
		unbondEntry.Balance = remainingBalance
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

	// Subtract the specified unlocking lockup amt
	err = k.SubtractUnlockingLockup(ctx, delAddr, valAddr, creationHeight, amt)
	if err != nil {
		return errorsmod.Wrap(err, "subtract unlocking lockup")
	}

	// Add the specified amt back to existing lockup
	k.AddLockup(ctx, delAddr, valAddr, amt)

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeCancelUnlocking,
			sdk.NewAttribute(stakingtypes.AttributeKeyDelegator, delAddr.String()),
			sdk.NewAttribute(stakingtypes.AttributeKeyValidator, valAddr.String()),
			sdk.NewAttribute(types.AttributeKeyCreationHeight, fmt.Sprintf("%d", creationHeight)),
			sdk.NewAttribute(sdk.AttributeKeyAmount, amt.String()),
		),
	)

	return nil
}

// GetDeveloperStake calculates and returns the total amount of all active lockups.
func (k Keeper) GetDeveloperStake(ctx sdk.Context) math.Int {
	totalDeveloperStake := math.ZeroInt()

	lockupsCallback := func(delAddr sdk.AccAddress, valAddr sdk.ValAddress, lockup types.Lockup) {
		totalDeveloperStake = totalDeveloperStake.Add(lockup.Amount)
	}

	k.MustIterateLockups(ctx, lockupsCallback)

	return totalDeveloperStake
}
