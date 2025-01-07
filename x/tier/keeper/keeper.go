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
	// Specified amt must be a positive integer
	if !amt.IsPositive() {
		return types.ErrInvalidAmount.Wrap("lock negative amount")
	}

	// Move the stake from delegator to the module
	stake := sdk.NewCoin(appparams.DefaultBondDenom, amt)
	coins := sdk.NewCoins(stake)
	err := k.bankKeeper.DelegateCoinsFromAccountToModule(ctx, delAddr, types.ModuleName, coins)
	if err != nil {
		return errorsmod.Wrapf(err, "delegate %s from account to module", stake)
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
func (k Keeper) Unlock(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, amt math.Int) (*time.Time, int64, error) {
	// Specified amt must be a positive integer
	if !amt.IsPositive() {
		return nil, 0, types.ErrInvalidAmount.Wrap("unlock negative amount")
	}

	// Subtract the lockup from the validator
	err := k.SubtractLockup(ctx, delAddr, valAddr, amt)
	if err != nil {
		return nil, 0, errorsmod.Wrap(err, "subtract lockup")
	}

	// Create unlocking lockup record at the current block height and set unlockTime
	creationHeight, unlockTime := k.SetLockup(ctx, true, delAddr, valAddr, amt)

	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeUnlock,
			sdk.NewAttribute(stakingtypes.AttributeKeyDelegator, delAddr.String()),
			sdk.NewAttribute(stakingtypes.AttributeKeyValidator, valAddr.String()),
			sdk.NewAttribute(sdk.AttributeKeyAmount, amt.String()),
			sdk.NewAttribute(types.AttributeKeyUnlockTime, unlockTime.String()),
			sdk.NewAttribute(types.AttributeKeyCreationHeight, fmt.Sprintf("%d", creationHeight)),
		),
	)

	return unlockTime, creationHeight, nil
}

// Redelegate moves the stake of a delegator from a source validator to a destination validator.
// The redelegation will be completed immediately because we keep the stake within the tier module.
func (k Keeper) Redelegate(ctx context.Context, delAddr sdk.AccAddress, srcValAddr, dstValAddr sdk.ValAddress, amt math.Int) (*time.Time, error) {
	// Specified amt must be a positive integer
	if !amt.IsPositive() {
		return nil, types.ErrInvalidAmount.Wrap("redelegate negative amount")
	}

	// Subtract the lockup from the source validator
	err := k.SubtractLockup(ctx, delAddr, srcValAddr, amt)
	if err != nil {
		return nil, errorsmod.Wrap(err, "subtract lockup from source validator")
	}

	// Add the lockup to the destination validator
	k.AddLockup(ctx, delAddr, dstValAddr, amt)

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	completionTime := sdkCtx.BlockTime()

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

	return &completionTime, nil
}

// CancelUnlocking effectively cancels the pending unlocking lockup partially or in full.
// Reverts the specified amt if a valid value is provided (e.g. 0 < amt < unlocking lockup amount).
func (k Keeper) CancelUnlocking(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, creationHeight int64, amt math.Int) error {
	// Specified amt must be a positive integer
	if !amt.IsPositive() {
		return types.ErrInvalidAmount.Wrap("cancel unlocking negative amount")
	}

	// Subtract the specified unlocking lockup amt
	err := k.SubtractUnlockingLockup(ctx, delAddr, valAddr, creationHeight, amt)
	if err != nil {
		return errorsmod.Wrap(err, "subtract unlocking lockup")
	}

	// Add the specified amt back to existing lockup (without modifying the unlock/unbond times)
	k.AddLockup(ctx, delAddr, valAddr, amt)

	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeCancelUnlocking,
			sdk.NewAttribute(stakingtypes.AttributeKeyDelegator, delAddr.String()),
			sdk.NewAttribute(stakingtypes.AttributeKeyValidator, valAddr.String()),
			sdk.NewAttribute(sdk.AttributeKeyAmount, amt.String()),
			sdk.NewAttribute(types.AttributeKeyCreationHeight, fmt.Sprintf("%d", creationHeight)),
		),
	)

	return nil
}
