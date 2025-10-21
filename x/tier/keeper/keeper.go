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
	"github.com/sourcenetwork/sourcehub/app/metrics"
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
		feegrantKeeper     types.FeegrantKeeper
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
	feegrantKeeper types.FeegrantKeeper,
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
		feegrantKeeper:     feegrantKeeper,
	}
}

// GetAuthority returns the module's authority.
func (k *Keeper) GetAuthority() string {
	return k.authority
}

// GetStakingKeeper returns the module's StakingKeeper.
func (k *Keeper) GetStakingKeeper() types.StakingKeeper {
	return k.stakingKeeper
}

// GetBankKeeper returns the module's BankKeeper.
func (k *Keeper) GetBankKeeper() types.BankKeeper {
	return k.bankKeeper
}

// GetEpochsKeeper returns the module's EpochsKeeper.
func (k *Keeper) GetEpochsKeeper() types.EpochsKeeper {
	return k.epochsKeeper
}

// GetDistributionKeeper returns the module's DistributionKeeper.
func (k *Keeper) GetDistributionKeeper() types.DistributionKeeper {
	return k.distributionKeeper
}

// GetFeegrantKeeper returns the module's FeegrantKeeper.
func (k *Keeper) GetFeegrantKeeper() types.FeegrantKeeper {
	return k.feegrantKeeper
}

// Logger returns a module-specific logger.
func (k *Keeper) Logger() log.Logger {
	return k.logger.With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// CompleteUnlocking completes the unlocking process for all lockups that have reached their unlock time.
// It is called at the end of each Epoch.
func (k *Keeper) CompleteUnlocking(ctx context.Context) (err error) {
	start := time.Now()

	defer func() {
		metrics.ModuleMeasureSinceWithCounter(
			types.ModuleName,
			metrics.CompleteUnlocking,
			start,
			err,
			nil,
		)
	}()

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

	err = k.iterateUnlockingLockups(ctx, cb)
	if err != nil {
		return errorsmod.Wrap(err, "iterate unlocking lockups")
	}

	return nil
}

// Lock locks the stake of a delegator to a validator.
func (k *Keeper) Lock(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, amt math.Int) (err error) {
	start := time.Now()

	defer func() {
		metrics.ModuleMeasureSinceWithCounter(
			types.ModuleName,
			metrics.Lock,
			start,
			err,
			[]metrics.Label{
				metrics.NewLabel(metrics.Amount, amt.String()),
				metrics.NewLabel(metrics.Delegator, delAddr.String()),
				metrics.NewLabel(metrics.Validator, valAddr.String()),
			},
		)
	}()

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

	// Delegate the stake to the validator
	modAddr := authtypes.NewModuleAddress(types.ModuleName)
	_, err = k.stakingKeeper.Delegate(ctx, modAddr, stake.Amount, stakingtypes.Unbonded, validator, true)
	if err != nil {
		return errorsmod.Wrapf(err, "delegate %s", stake)
	}

	// Mint credits
	creditAmt := k.proratedCredit(ctx, delAddr, amt)
	err = k.mintCredit(ctx, delAddr, creditAmt)
	if err != nil {
		return errorsmod.Wrap(err, "mint credit")
	}

	// Record the lockup after minting credits
	err = k.AddLockup(ctx, delAddr, valAddr, stake.Amount)
	if err != nil {
		return errorsmod.Wrap(err, "add lockup")
	}

	// Update total credit amount
	totalCredits := k.getTotalCreditAmount(ctx)
	totalCredits = totalCredits.Add(creditAmt)
	err = k.setTotalCreditAmount(ctx, totalCredits)
	if err != nil {
		return errorsmod.Wrap(err, "set total credit amount")
	}

	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeLock,
			sdk.NewAttribute(stakingtypes.AttributeKeyDelegator, delAddr.String()),
			sdk.NewAttribute(stakingtypes.AttributeKeyValidator, valAddr.String()),
			sdk.NewAttribute(sdk.AttributeKeyAmount, amt.String()),
			sdk.NewAttribute(types.AttributeCreditAmount, creditAmt.String()),
		),
	)

	return nil
}

// Unlock initiates the unlocking of the specified lockup amount of a delegator from a validator.
// The specified lockup amount will be unlocked in CompleteUnlocking after the unlocking period has passed.
func (k *Keeper) Unlock(
	ctx context.Context,
	delAddr sdk.AccAddress,
	valAddr sdk.ValAddress,
	amt math.Int,
) (creationHeight int64, completionTime, unlockTime time.Time, err error) {
	start := time.Now()

	defer func() {
		metrics.ModuleMeasureSinceWithCounter(
			types.ModuleName,
			metrics.Unlock,
			start,
			err,
			[]metrics.Label{
				metrics.NewLabel(metrics.Amount, amt.String()),
				metrics.NewLabel(metrics.Delegator, delAddr.String()),
				metrics.NewLabel(metrics.Validator, valAddr.String()),
			},
		)
	}()

	// Specified amt must be a positive integer
	if !amt.IsPositive() {
		return 0, time.Time{}, time.Time{}, types.ErrInvalidAmount.Wrap("unlock non-positive amount")
	}

	err = k.subtractLockup(ctx, delAddr, valAddr, amt)
	if err != nil {
		return 0, time.Time{}, time.Time{}, errorsmod.Wrap(err, "subtract lockup")
	}

	// Remove the associated insurance lockup record if the lockup was fully removed
	if !k.hasLockup(ctx, delAddr, valAddr) {
		k.removeInsuranceLockup(ctx, delAddr, valAddr)
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
			sdk.NewAttribute(types.AttributeKeyCompletionTime, completionTime.Format(time.RFC3339)),
			sdk.NewAttribute(types.AttributeKeyUnlockTime, unlockTime.Format(time.RFC3339)),
		),
	)

	return creationHeight, completionTime, unlockTime, nil
}

// Redelegate redelegates the stake of a delegator from a source validator to a destination validator.
// The redelegation will be completed after the unbonding period has passed (e.g. at completionTime).
func (k *Keeper) Redelegate(ctx context.Context, delAddr sdk.AccAddress, srcValAddr, dstValAddr sdk.ValAddress,
	amt math.Int) (completionTime time.Time, err error) {

	start := time.Now()

	defer func() {
		metrics.ModuleMeasureSinceWithCounter(
			types.ModuleName,
			metrics.Redelegate,
			start,
			err,
			[]metrics.Label{
				metrics.NewLabel(metrics.Amount, amt.String()),
				metrics.NewLabel(metrics.Delegator, delAddr.String()),
				metrics.NewLabel(metrics.SrcValidator, srcValAddr.String()),
				metrics.NewLabel(metrics.DstValidator, dstValAddr.String()),
			},
		)
	}()

	// Specified amt must be a positive integer
	if !amt.IsPositive() {
		return time.Time{}, types.ErrInvalidAmount.Wrap("redelegate non-positive amount")
	}

	// Subtract the lockup from the source validator
	err = k.subtractLockup(ctx, delAddr, srcValAddr, amt)
	if err != nil {
		return time.Time{}, errorsmod.Wrap(err, "subtract lockup from source validator")
	}

	// Add the lockup to the destination validator
	err = k.AddLockup(ctx, delAddr, dstValAddr, amt)
	if err != nil {
		return time.Time{}, errorsmod.Wrap(err, "add lockup to destination validator")
	}

	// Redelegate the  associated insurance lockup record if the lockup was fully redelegated
	if !k.hasLockup(ctx, delAddr, srcValAddr) {
		insuranceLockupAmount := k.getInsuranceLockupAmount(ctx, delAddr, srcValAddr)
		if insuranceLockupAmount.IsPositive() {
			k.removeInsuranceLockup(ctx, delAddr, srcValAddr)
			k.AddInsuranceLockup(ctx, delAddr, dstValAddr, insuranceLockupAmount)
		}
	}

	modAddr := authtypes.NewModuleAddress(types.ModuleName)
	shares, err := k.stakingKeeper.ValidateUnbondAmount(ctx, modAddr, srcValAddr, amt)
	if err != nil {
		return time.Time{}, errorsmod.Wrap(err, "validate unbond amount")
	}

	completionTime, err = k.stakingKeeper.BeginRedelegation(ctx, modAddr, srcValAddr, dstValAddr, shares)
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
			sdk.NewAttribute(types.AttributeKeyCompletionTime, completionTime.Format(time.RFC3339)),
		),
	)

	return completionTime, nil
}

// CancelUnlocking effectively cancels the pending unlocking lockup partially or in full.
// Reverts the specified amt if a valid value is provided (e.g. 0 < amt < unlocking lockup amount).
func (k *Keeper) CancelUnlocking(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress,
	creationHeight int64, amt math.Int) (err error) {

	start := time.Now()

	defer func() {
		metrics.ModuleMeasureSinceWithCounter(
			types.ModuleName,
			metrics.CancelUnlocking,
			start,
			err,
			[]metrics.Label{
				metrics.NewLabel(metrics.Amount, amt.String()),
				metrics.NewLabel(metrics.CreationHeight, fmt.Sprintf("%d", creationHeight)),
				metrics.NewLabel(metrics.Delegator, delAddr.String()),
				metrics.NewLabel(metrics.Validator, valAddr.String()),
			},
		)
	}()

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
	err = k.subtractUnlockingLockup(ctx, delAddr, valAddr, creationHeight, amt)
	if err != nil {
		return errorsmod.Wrap(err, "subtract unlocking lockup")
	}

	// Add the specified amt back to existing lockup
	err = k.AddLockup(ctx, delAddr, valAddr, amt)
	if err != nil {
		return errorsmod.Wrap(err, "add lockup")
	}

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

// CreateDeveloper creates a new developer record.
func (k *Keeper) CreateDeveloper(ctx context.Context, developerAddr sdk.AccAddress, autoLockEnabled bool) (err error) {
	start := time.Now()

	defer func() {
		metrics.ModuleMeasureSinceWithCounter(
			types.ModuleName,
			metrics.CreateDeveloper,
			start,
			err,
			[]metrics.Label{
				metrics.NewLabel(metrics.Developer, developerAddr.String()),
			},
		)
	}()

	existingDev := k.GetDeveloper(ctx, developerAddr)
	if existingDev != nil {
		return types.ErrInvalidAddress.Wrapf("developer %s already exists", developerAddr.String())
	}

	developer := &types.Developer{
		Address:         developerAddr.String(),
		AutoLockEnabled: autoLockEnabled,
	}

	k.SetDeveloper(ctx, developerAddr, developer)

	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeDeveloperCreated,
			sdk.NewAttribute(types.AttributeKeyDeveloper, developerAddr.String()),
			sdk.NewAttribute(types.AttributeKeyAutoLockEnabled, fmt.Sprintf("%t", autoLockEnabled)),
		),
	)

	return nil
}

// UpdateDeveloper updates the developer record.
func (k *Keeper) UpdateDeveloper(ctx context.Context, developerAddr sdk.AccAddress, autoLockEnabled bool) (err error) {
	start := time.Now()

	defer func() {
		metrics.ModuleMeasureSinceWithCounter(
			types.ModuleName,
			metrics.UpdateDeveloper,
			start,
			err,
			[]metrics.Label{
				metrics.NewLabel(metrics.Developer, developerAddr.String()),
			},
		)
	}()

	developer := k.GetDeveloper(ctx, developerAddr)
	if developer == nil {
		return k.CreateDeveloper(ctx, developerAddr, autoLockEnabled)
	}

	developer.AutoLockEnabled = autoLockEnabled

	k.SetDeveloper(ctx, developerAddr, developer)

	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeDeveloperUpdated,
			sdk.NewAttribute(types.AttributeKeyDeveloper, developerAddr.String()),
			sdk.NewAttribute(types.AttributeKeyAutoLockEnabled, fmt.Sprintf("%t", autoLockEnabled)),
		),
	)

	return nil
}

// RemoveDeveloper removes the developer record and all associated user subscriptions.
func (k *Keeper) RemoveDeveloper(ctx context.Context, developerAddr sdk.AccAddress) (err error) {
	start := time.Now()

	defer func() {
		metrics.ModuleMeasureSinceWithCounter(
			types.ModuleName,
			metrics.RemoveDeveloper,
			start,
			err,
			[]metrics.Label{
				metrics.NewLabel(metrics.Developer, developerAddr.String()),
			},
		)
	}()

	existingDev := k.GetDeveloper(ctx, developerAddr)
	if existingDev == nil {
		return types.ErrInvalidAddress.Wrapf("developer %s does not exist", developerAddr.String())
	}

	var subscriptionsToRemove []struct {
		developerAddr sdk.AccAddress
		userDid       string
	}

	k.mustIterateUserSubscriptionsForDeveloper(ctx, developerAddr,
		func(devAddr sdk.AccAddress, userDid string, userSubscription types.UserSubscription) {
			subscriptionsToRemove = append(subscriptionsToRemove, struct {
				developerAddr sdk.AccAddress
				userDid       string
			}{devAddr, userDid})
		})

	for _, sub := range subscriptionsToRemove {
		err := k.RemoveUserSubscription(ctx, sub.developerAddr, sub.userDid)
		if err != nil {
			continue
		}
	}

	k.removeDeveloper(ctx, developerAddr)

	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeDeveloperRemoved,
			sdk.NewAttribute(types.AttributeKeyDeveloper, developerAddr.String()),
		),
	)

	return nil
}

// AddUserSubscription registers a user to a developer for receiving allowances.
func (k *Keeper) AddUserSubscription(
	ctx context.Context,
	developerAddr sdk.AccAddress,
	userDid string,
	amount *sdk.Coin,
	period uint64,
) (err error) {
	start := time.Now()

	subscriptionAmount := math.ZeroInt()
	if amount != nil {
		subscriptionAmount = amount.Amount
	}

	defer func() {
		metrics.ModuleMeasureSinceWithCounter(
			types.ModuleName,
			metrics.AddUserSubscription,
			start,
			err,
			[]metrics.Label{
				metrics.NewLabel(metrics.Developer, developerAddr.String()),
				metrics.NewLabel(metrics.UserDid, userDid),
				metrics.NewLabel(metrics.SubscriptionAmount, subscriptionAmount.String()),
				metrics.NewLabel(metrics.SubscriptionPeriod, fmt.Sprintf("%d", period)),
			},
		)
	}()

	if developerAddr.Empty() {
		return types.ErrInvalidAddress.Wrap("developer address cannot be empty")
	}
	if userDid == "" {
		return types.ErrInvalidAddress.Wrap("user did cannot be empty")
	}

	existingSub := k.GetUserSubscription(ctx, developerAddr, userDid)
	if existingSub != nil {
		return types.ErrInvalidAddress.Wrapf("user %s is already subscribed to developer %s", userDid, developerAddr.String())
	}

	if amount != nil {
		if amount.Denom != appparams.MicroCreditDenom {
			return types.ErrInvalidAmount.Wrapf("invalid amount denomination: expected %s, got %s", appparams.MicroCreditDenom, amount.Denom)
		}
		if !subscriptionAmount.IsPositive() {
			return types.ErrInvalidAmount.Wrap("amount must be positive")
		}

		developer := k.GetDeveloper(ctx, developerAddr)
		if developer == nil {
			return types.ErrInvalidAddress.Wrapf("developer %s not found", developerAddr.String())
		}

		// Check if developer has enough available credits
		err := k.validateDeveloperCredits(ctx, developerAddr, subscriptionAmount)
		if err != nil {
			if !developer.AutoLockEnabled {
				return types.ErrInvalidAmount.Wrapf("insufficient credits and auto-lock disabled: %v", err)
			}
			// Convert required credits to uopen to lock using highest applicable rate
			params := k.GetParams(ctx)
			effectiveRate := int64(0)
			for _, r := range params.RewardRates {
				if subscriptionAmount.GTE(r.Amount) {
					effectiveRate = r.Rate
					break
				}
			}
			if effectiveRate <= 0 {
				last := params.RewardRates[len(params.RewardRates)-1]
				effectiveRate = last.Rate
			}
			lockNumerator := subscriptionAmount.MulRaw(100).AddRaw(effectiveRate - 1)
			lockAmt := lockNumerator.QuoRaw(effectiveRate)
			if !lockAmt.IsPositive() {
				lockAmt = math.OneInt()
			}

			// Try to add more lockup to cover the required amount
			err = k.addLockupForRegistration(ctx, developerAddr, lockAmt)
			if err != nil {
				return types.ErrInvalidAmount.Wrapf("auto-lock failed: %v", err)
			}
		}
	}

	now := sdk.UnwrapSDKContext(ctx).BlockTime()
	userSubscription := &types.UserSubscription{
		Developer:    developerAddr.String(),
		UserDid:      userDid,
		CreditAmount: sdk.NewCoin(appparams.MicroCreditDenom, subscriptionAmount),
		StartDate:    now,
		Period:       period,
		LastRenewed:  now,
	}

	k.SetUserSubscription(ctx, developerAddr, userDid, userSubscription)

	// Grant or update periodic DID allowance via feegrant to the user DID from the developer
	if amount != nil && subscriptionAmount.IsPositive() {
		spendLimit := sdk.NewCoins(sdk.NewCoin(appparams.MicroCreditDenom, subscriptionAmount))
		grantPeriod := time.Duration(period) * time.Second
		if grantPeriod == 0 {
			params := k.GetParams(ctx)
			grantPeriod = *params.EpochDuration
		}
		if err := k.grantPeriodicDIDAllowance(ctx, developerAddr, userDid, spendLimit, grantPeriod); err != nil {
			return errorsmod.Wrap(err, "grant periodic DID allowance")
		}
	}

	err = k.updateDeveloperTotalGranted(ctx, developerAddr, userSubscription.CreditAmount.Amount, true)
	if err != nil {
		return types.ErrInvalidAmount.Wrapf("failed to update total granted amount: %v", err)
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	event := sdk.NewEvent(
		types.EventTypeUserSubscriptionCreated,
		sdk.NewAttribute(types.AttributeKeyDeveloper, developerAddr.String()),
		sdk.NewAttribute(types.AttributeKeyUserDid, userDid),
	)
	if amount != nil {
		event.AppendAttributes(sdk.NewAttribute(types.AttributeKeySubscriptionAmount, subscriptionAmount.String()))
	}
	if period > 0 {
		event.AppendAttributes(sdk.NewAttribute(types.AttributeKeySubscriptionPeriod, fmt.Sprintf("%d", period)))
	}
	sdkCtx.EventManager().EmitEvent(event)

	return nil
}

// UpdateUserSubscription updates the user subscription.
func (k *Keeper) UpdateUserSubscription(
	ctx context.Context,
	developerAddr sdk.AccAddress,
	userDid string,
	amount *sdk.Coin,
	period uint64,
) (err error) {
	start := time.Now()

	newAmount := math.ZeroInt()
	if amount != nil {
		newAmount = amount.Amount
	}

	defer func() {
		metrics.ModuleMeasureSinceWithCounter(
			types.ModuleName,
			metrics.UpdateUserSubscription,
			start,
			err,
			[]metrics.Label{
				metrics.NewLabel(metrics.Developer, developerAddr.String()),
				metrics.NewLabel(metrics.UserDid, userDid),
				metrics.NewLabel(metrics.SubscriptionAmount, newAmount.String()),
				metrics.NewLabel(metrics.SubscriptionPeriod, fmt.Sprintf("%d", period)),
			},
		)
	}()

	userSubscription := k.GetUserSubscription(ctx, developerAddr, userDid)
	if userSubscription == nil {
		return types.ErrInvalidAddress.Wrapf("user %s is not subscribed to developer %s", userDid, developerAddr.String())
	}

	oldAmount := userSubscription.CreditAmount.Amount
	if newAmount.GT(oldAmount) {
		delta := newAmount.Sub(oldAmount)
		if err := k.updateDeveloperTotalGranted(ctx, developerAddr, delta, true); err != nil {
			return err
		}
	} else if oldAmount.GT(newAmount) {
		delta := oldAmount.Sub(newAmount)
		if err := k.updateDeveloperTotalGranted(ctx, developerAddr, delta, false); err != nil {
			return err
		}
	}

	userSubscription.CreditAmount = sdk.NewCoin(appparams.MicroCreditDenom, newAmount)
	userSubscription.Period = period
	userSubscription.LastRenewed = sdk.UnwrapSDKContext(ctx).BlockTime()
	k.SetUserSubscription(ctx, developerAddr, userDid, userSubscription)

	if amount != nil && newAmount.IsPositive() {
		spendLimit := sdk.NewCoins(sdk.NewCoin(appparams.MicroCreditDenom, newAmount))
		updatePeriod := time.Duration(period) * time.Second
		if updatePeriod == 0 {
			params := k.GetParams(ctx)
			updatePeriod = *params.EpochDuration
		}
		if err := k.grantPeriodicDIDAllowance(ctx, developerAddr, userDid, spendLimit, updatePeriod); err != nil {
			return errorsmod.Wrap(err, "grant periodic DID allowance")
		}
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	event := sdk.NewEvent(
		types.EventTypeUserSubscriptionUpdated,
		sdk.NewAttribute(types.AttributeKeyDeveloper, developerAddr.String()),
		sdk.NewAttribute(types.AttributeKeyUserDid, userDid),
	)
	if amount != nil {
		event.AppendAttributes(sdk.NewAttribute(types.AttributeKeySubscriptionAmount, newAmount.String()))
	}
	if period > 0 {
		event.AppendAttributes(sdk.NewAttribute(types.AttributeKeySubscriptionPeriod, fmt.Sprintf("%d", period)))
	}
	sdkCtx.EventManager().EmitEvent(event)

	return nil
}

// RemoveUserSubscription removes the user subscription.
func (k *Keeper) RemoveUserSubscription(
	ctx context.Context,
	developerAddr sdk.AccAddress,
	userDid string,
) (err error) {
	start := time.Now()

	defer func() {
		metrics.ModuleMeasureSinceWithCounter(
			types.ModuleName,
			metrics.RemoveUserSubscription,
			start,
			err,
			[]metrics.Label{
				metrics.NewLabel(metrics.Developer, developerAddr.String()),
				metrics.NewLabel(metrics.UserDid, userDid),
			},
		)
	}()

	existingSub := k.GetUserSubscription(ctx, developerAddr, userDid)
	if existingSub == nil {
		return types.ErrInvalidAddress.Wrapf("user %s is not subscribed to developer %s", userDid, developerAddr.String())
	}

	if err := k.updateDeveloperTotalGranted(ctx, developerAddr, existingSub.CreditAmount.Amount, false); err != nil {
		return err
	}

	k.removeUserSubscription(ctx, developerAddr, userDid)

	// Expire the DID allowance by setting the expiration to the current period reset
	// Ignore error if no allowance exists
	_ = k.feegrantKeeper.ExpireDIDAllowance(ctx, developerAddr, userDid)

	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeUserSubscriptionRemoved,
			sdk.NewAttribute(types.AttributeKeyDeveloper, developerAddr.String()),
			sdk.NewAttribute(types.AttributeKeyUserDid, userDid),
		),
	)

	return nil
}
