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

// GetAuthority returns the module's StakingKeeper.
func (k Keeper) GetStakingKeeper() types.StakingKeeper {
	return k.stakingKeeper
}

// GetAuthority returns the module's BankKeeper.
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
	cb := func(delAddr sdk.AccAddress, valAddr sdk.ValAddress, creationHeight int64, lockup types.Lockup) error {
		if sdk.UnwrapSDKContext(ctx).BlockTime().Before(*lockup.UnlockTime) {
			fmt.Printf("Unlock time not reached for %s/%s\n", delAddr, valAddr)
			return nil
		}

		// Redeem the unlocked lockup for stake.
		// Adjust calculation forthe given tolerance, because the undelegated amount
		// could be less than the expected amount due to validator share calculations.
		// TODO: handle tolerance more gracefully.
		tolerance := math.NewInt(1)
		stake := sdk.NewCoin(appparams.DefaultBondDenom, lockup.Amount.Sub(tolerance))
		coins := sdk.NewCoins(stake)

		moduleBalance := k.bankKeeper.GetBalance(ctx, authtypes.NewModuleAddress(types.ModuleName), appparams.DefaultBondDenom)
		if moduleBalance.Amount.Add(tolerance).LT(lockup.Amount) {
			fmt.Printf("Module account balance %s is smaller than required amount %s\n", delAddr, valAddr)
			return nil
		}

		err := k.bankKeeper.UndelegateCoinsFromModuleToAccount(ctx, types.ModuleName, delAddr, coins)
		if err != nil {
			return errorsmod.Wrapf(err, "undelegate coins to %s for amount %s", delAddr, stake)
		}

		k.removeUnlockingLockup(ctx, delAddr, valAddr)

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

	return nil
}

// Unlock initiates the unlocking of stake of a delegator from a validator.
// The stake will be unlocked after the unlocking period has passed.
func (k Keeper) Unlock(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, amt math.Int) (
	unbondTime time.Time, unlockTime time.Time, creationHeight int64, err error) {

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	creationHeight = sdkCtx.BlockHeight()

	err = k.SubtractLockup(ctx, delAddr, valAddr, amt)
	if err != nil {
		return time.Time{}, time.Time{}, 0, errorsmod.Wrap(err, "subtract lockup")
	}

	params := k.GetParams(ctx)
	epochDuration := params.EpochDuration
	unlockingDuration := time.Duration(params.UnlockingEpochs) * *epochDuration
	unlockTime = sdkCtx.BlockTime().Add(unlockingDuration)
	modAddr := authtypes.NewModuleAddress(types.ModuleName)

	shares, err := k.stakingKeeper.ValidateUnbondAmount(ctx, modAddr, valAddr, amt)
	if err != nil {
		return time.Time{}, time.Time{}, 0, errorsmod.Wrap(err, "validate unbond amount")
	}

	unbondTime, _, err = k.stakingKeeper.Undelegate(ctx, modAddr, valAddr, shares)
	if err != nil {
		return time.Time{}, time.Time{}, 0, errorsmod.Wrap(err, "undelegate")
	}

	k.removeLockup(ctx, delAddr, valAddr)

	k.SetLockup(ctx, true, delAddr, valAddr, amt, creationHeight, &unbondTime, &unlockTime)

	return unbondTime, unlockTime, creationHeight, nil
}

// Redelegate redelegates the stake of a delegator from a source validator to a destination validator.
// The redelegation will be completed after the unbonding period has passed.
func (k Keeper) Redelegate(ctx context.Context, delAddr sdk.AccAddress, srcValAddr, dstValAddr sdk.ValAddress, amt math.Int) (
	completionTime time.Time, err error) {

	err = k.SubtractLockup(ctx, delAddr, srcValAddr, amt)
	if err != nil {
		return time.Time{}, errorsmod.Wrap(err, "subtract locked stake from source validator")
	}

	k.AddLockup(ctx, delAddr, dstValAddr, amt)

	modAddr := authtypes.NewModuleAddress(types.ModuleName)

	shares, err := k.stakingKeeper.ValidateUnbondAmount(ctx, modAddr, srcValAddr, amt)
	if err != nil {
		return time.Time{}, errorsmod.Wrap(err, "validate unbond amount")
	}

	completionTime, err = k.stakingKeeper.BeginRedelegation(ctx, modAddr, srcValAddr, dstValAddr, shares)
	if err != nil {
		return time.Time{}, errorsmod.Wrap(err, "begin redelegation")
	}

	return completionTime, nil
}
