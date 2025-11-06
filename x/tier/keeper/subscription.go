package keeper

import (
	"context"
	"fmt"
	"time"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sourcenetwork/sourcehub/app/metrics"
	appparams "github.com/sourcenetwork/sourcehub/app/params"
	"github.com/sourcenetwork/sourcehub/x/feegrant"
	"github.com/sourcenetwork/sourcehub/x/tier/types"
)

// developerStore returns a prefix store for developer configurations.
func (k *Keeper) developerStore(ctx context.Context) prefix.Store {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	return prefix.NewStore(storeAdapter, []byte(types.DeveloperKeyPrefix))
}

// GetAllDevelopers returns all developers in the store.
func (k *Keeper) GetAllDevelopers(ctx context.Context) []types.Developer {
	var developers []types.Developer

	developersCallback := func(developerAddr sdk.AccAddress, developer types.Developer) {
		developers = append(developers, developer)
	}

	k.mustIterateDevelopers(ctx, developersCallback)

	return developers
}

// GetDeveloper returns a pointer to existing developer configuration, or nil if not found.
func (k *Keeper) GetDeveloper(ctx context.Context, developerAddr sdk.AccAddress) *types.Developer {
	key := types.DeveloperKey(developerAddr)
	store := k.developerStore(ctx)
	b := store.Get(key)
	if b == nil {
		return nil
	}

	var developer types.Developer
	k.cdc.MustUnmarshal(b, &developer)

	return &developer
}

// SetDeveloper sets a developer configuration in the store.
func (k *Keeper) SetDeveloper(ctx context.Context, developerAddr sdk.AccAddress, developer *types.Developer) {
	key := types.DeveloperKey(developerAddr)
	b := k.cdc.MustMarshal(developer)
	store := k.developerStore(ctx)
	store.Set(key, b)
}

// removeDeveloper removes existing Developer (developerAddr/).
func (k *Keeper) removeDeveloper(ctx context.Context, developerAddr sdk.AccAddress) {
	key := types.DeveloperKey(developerAddr)
	store := k.developerStore(ctx)
	store.Delete(key)
}

// mustIterateDevelopers iterates through all developers and calls the callback function.
func (k *Keeper) mustIterateDevelopers(ctx context.Context,
	cb func(developerAddr sdk.AccAddress, developer types.Developer)) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, []byte(types.DeveloperKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var developer types.Developer
		k.cdc.MustUnmarshal(iterator.Value(), &developer)
		developerAddr := types.DeveloperKeyToAddress(iterator.Key())
		cb(developerAddr, developer)
	}
}

// userSubscriptionStore returns a prefix store for user subscriptions.
func (k *Keeper) userSubscriptionStore(ctx context.Context) prefix.Store {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	return prefix.NewStore(storeAdapter, []byte(types.UserSubscriptionKeyPrefix))
}

// GetAllUserSubscriptions returns all user subscriptions in the store.
func (k *Keeper) GetAllUserSubscriptions(ctx context.Context) []types.UserSubscription {
	var userSubscriptions []types.UserSubscription

	userSubscriptionsCallback := func(developerAddr sdk.AccAddress, userDid string, userSubscription types.UserSubscription) {
		userSubscriptions = append(userSubscriptions, userSubscription)
	}

	k.mustIterateUserSubscriptions(ctx, userSubscriptionsCallback)

	return userSubscriptions
}

// GetUserSubscription returns a pointer to existing user subscription, or nil if not found.
func (k *Keeper) GetUserSubscription(ctx context.Context, developerAddr sdk.AccAddress, userDid string) *types.UserSubscription {
	key := types.UserSubscriptionKey(developerAddr, userDid)
	store := k.userSubscriptionStore(ctx)
	b := store.Get(key)
	if b == nil {
		return nil
	}

	var userSubscription types.UserSubscription
	k.cdc.MustUnmarshal(b, &userSubscription)

	return &userSubscription
}

// SetUserSubscription sets a user subscription in the store based on the UserSubscriptionKey.
func (k *Keeper) SetUserSubscription(ctx context.Context, developerAddr sdk.AccAddress, userDid string, userSubscription *types.UserSubscription) {
	key := types.UserSubscriptionKey(developerAddr, userDid)
	b := k.cdc.MustMarshal(userSubscription)
	store := k.userSubscriptionStore(ctx)
	store.Set(key, b)
}

// removeUserSubscription removes existing UserSubscription (developerAddr/userDid/).
func (k *Keeper) removeUserSubscription(ctx context.Context, developerAddr sdk.AccAddress, userDid string) {
	key := types.UserSubscriptionKey(developerAddr, userDid)
	store := k.userSubscriptionStore(ctx)
	store.Delete(key)
}

// mustIterateUserSubscriptions iterates through all user subscriptions and calls the callback function.
func (k *Keeper) mustIterateUserSubscriptions(ctx context.Context,
	cb func(developerAddr sdk.AccAddress, userDid string, userSubscription types.UserSubscription)) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, []byte(types.UserSubscriptionKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var userSubscription types.UserSubscription
		k.cdc.MustUnmarshal(iterator.Value(), &userSubscription)
		developerAddr, userDid := types.UserSubscriptionKeyToAddresses(iterator.Key())
		cb(developerAddr, userDid, userSubscription)
	}
}

// mustIterateUserSubscriptionsForDeveloper iterates through user subscriptions for a specific developer and calls the callback function.
func (k *Keeper) mustIterateUserSubscriptionsForDeveloper(ctx context.Context, developerAddr sdk.AccAddress,
	cb func(developerAddr sdk.AccAddress, userDid string, userSubscription types.UserSubscription)) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, []byte(types.UserSubscriptionKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, developerAddr.Bytes())

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var userSubscription types.UserSubscription
		k.cdc.MustUnmarshal(iterator.Value(), &userSubscription)
		developerAddr, userDid := types.UserSubscriptionKeyToAddresses(iterator.Key())
		cb(developerAddr, userDid, userSubscription)
	}
}

// checkDeveloperCredits checks if all developers have enough credits compared to total dev granted.
// If developer does not have enough credits, we check the auto-lock setting.
// If auto-lock is on and developer has enough "uopen", we perform auto-lock.
// If auto-lock is off or developer does not have enough "uopen", we log corresponding event.
func (k *Keeper) checkDeveloperCredits(ctx context.Context, epochNumber int64) (err error) {
	start := time.Now()

	defer func() {
		metrics.ModuleMeasureSinceWithCounter(
			types.ModuleName,
			metrics.CheckDeveloperCredits,
			start,
			err,
			[]metrics.Label{
				metrics.NewLabel(metrics.Epoch, fmt.Sprintf("%d", epochNumber)),
			},
		)
	}()

	// Track unique developers to avoid processing the same developer multiple times
	processedDevelopers := make(map[string]bool)
	k.mustIterateUserSubscriptions(ctx, func(developerAddr sdk.AccAddress, userDid string, userSubscription types.UserSubscription) {
		if processedDevelopers[developerAddr.String()] {
			return
		}
		processedDevelopers[developerAddr.String()] = true
		totalGranted, err := k.getTotalDevGranted(ctx, developerAddr)
		if err != nil {
			k.logger.Error("failed to get total granted amount",
				types.AttributeKeyDeveloper, developerAddr.String(),
				types.AttributeKeyError, err.Error())
			return
		}

		// Developer has enough credits, no action needed
		creditBalance := k.bankKeeper.GetBalance(ctx, developerAddr, appparams.MicroCreditDenom)
		if creditBalance.Amount.GTE(totalGranted) {
			return
		}

		developer := k.GetDeveloper(ctx, developerAddr)
		sdkCtx := sdk.UnwrapSDKContext(ctx)

		// Auto-lock is off, log event that developer needs to lock
		if developer == nil || !developer.AutoLockEnabled {
			sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
				types.EventTypeDeveloperInsufficientCredits,
				sdk.NewAttribute(types.AttributeKeyDeveloper, developerAddr.String()),
				sdk.NewAttribute(types.AttributeKeyCurrentBalance, creditBalance.Amount.String()),
				sdk.NewAttribute(types.AttributeKeyTotalGranted, totalGranted.String()),
				sdk.NewAttribute(types.AttributeKeyAutoLockEnabled, "false"),
			))
			return
		}

		// Auto-lock is on, check uopen balance for the actual lock amount
		lockAmount := totalGranted.Sub(creditBalance.Amount)
		uopenBalance := k.bankKeeper.GetBalance(ctx, developerAddr, appparams.DefaultBondDenom)
		if uopenBalance.Amount.LT(lockAmount) {
			sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
				types.EventTypeDeveloperInsufficientOpenForAutoLock,
				sdk.NewAttribute(types.AttributeKeyDeveloper, developerAddr.String()),
				sdk.NewAttribute(types.AttributeKeyCurrentBalance, creditBalance.Amount.String()),
				sdk.NewAttribute(types.AttributeKeyTotalGranted, totalGranted.String()),
				sdk.NewAttribute(types.AttributeKeyUopenBalance, uopenBalance.Amount.String()),
				sdk.NewAttribute(types.AttributeKeyRequiredLockAmount, lockAmount.String()),
			))
			return
		}

		// Enough uopen balance, perform auto-lock
		err = k.autoLockDeveloperCredits(ctx, developerAddr, developer, lockAmount)
		if err != nil {
			sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
				types.EventTypeDeveloperAutoLockFailed,
				sdk.NewAttribute(types.AttributeKeyDeveloper, developerAddr.String()),
				sdk.NewAttribute(types.AttributeKeyError, err.Error()),
			))
		} else {
			sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
				types.EventTypeDeveloperAutoLockSuccess,
				sdk.NewAttribute(types.AttributeKeyDeveloper, developerAddr.String()),
				sdk.NewAttribute(types.AttributeKeyLockAmount, lockAmount.String()),
			))
		}
	})

	return nil
}

// autoLockDeveloperCredits automatically adds lockups to cover missing credits for a developer.
func (k *Keeper) autoLockDeveloperCredits(
	ctx context.Context,
	developerAddr sdk.AccAddress,
	developer *types.Developer,
	lockAmount math.Int,
) error {
	if lockAmount.LTE(math.ZeroInt()) {
		return types.ErrInvalidAmount.Wrap("auto-lock amount must be positive")
	}

	// Try to reuse an existing validator lockup for this developer; prefer the largest lockup
	var (
		chosenValAddr sdk.ValAddress
		found         bool
		maxLockupAmt  = math.ZeroInt()
	)

	k.mustIterateLockups(ctx, func(delAddr sdk.AccAddress, valAddr sdk.ValAddress, lockup types.Lockup) {
		if !delAddr.Equals(developerAddr) {
			return
		}
		if !found || lockup.Amount.GT(maxLockupAmt) {
			found = true
			chosenValAddr = valAddr
			maxLockupAmt = lockup.Amount
		}
	})

	if !found {
		validators, err := k.stakingKeeper.GetAllValidators(ctx)
		if err != nil {
			return types.ErrInvalidAmount.Wrapf("failed to get validators: %v", err)
		}
		if len(validators) == 0 {
			return types.ErrInvalidAmount.Wrap("no validators available for lockup")
		}

		validator := validators[0]
		valAddr, err := sdk.ValAddressFromBech32(validator.GetOperator())
		if err != nil {
			return types.ErrInvalidAmount.Wrapf("invalid validator address: %v", err)
		}
		chosenValAddr = valAddr
	}

	if err := k.Lock(ctx, developerAddr, chosenValAddr, lockAmount); err != nil {
		return types.ErrInvalidAmount.Wrapf("failed to add lockup: %v", err)
	}

	k.logger.Info("auto-locked developer credits",
		types.AttributeKeyDeveloper, developerAddr.String(),
		types.AttributeKeyLockAmount, lockAmount.String(),
		types.AttributeKeyDestinationValidator, chosenValAddr.String())

	return nil
}

// grantPeriodicDIDAllowance grants a periodic allowance from the developer to the user DID.
func (k *Keeper) grantPeriodicDIDAllowance(
	ctx context.Context,
	granter sdk.AccAddress,
	granteeDID string,
	spendLimit sdk.Coins,
	period time.Duration,
) error {
	now := sdk.UnwrapSDKContext(ctx).BlockTime()

	basicAllowance := feegrant.BasicAllowance{
		SpendLimit: spendLimit,
	}

	periodicAllowance := &feegrant.PeriodicAllowance{
		Basic:            basicAllowance,
		Period:           period,
		PeriodSpendLimit: spendLimit,
		PeriodCanSpend:   spendLimit,
		PeriodReset:      now.Add(period),
	}

	// Try update first; if no existing grant, fall back to grant
	if err := k.feegrantKeeper.UpdateDIDAllowance(ctx, granter, granteeDID, periodicAllowance); err != nil {
		if err := k.feegrantKeeper.GrantDIDAllowance(ctx, granter, granteeDID, periodicAllowance); err != nil {
			return errorsmod.Wrapf(err, "failed to grant periodic DID allowance from %s to %s", granter, granteeDID)
		}
	}

	k.logger.Info("granted periodic DID allowance",
		types.AttributeKeyDeveloper, granter,
		types.AttributeKeyUserDid, granteeDID,
		types.AttributeKeySubscriptionAmount, spendLimit.String(),
		types.AttributeKeySubscriptionPeriod, period.String(),
	)

	return nil
}

// validateDeveloperCredits checks if a developer has enough credits to grant the requested amount.
func (k *Keeper) validateDeveloperCredits(ctx context.Context, developerAddr sdk.AccAddress, requestedAmount math.Int) error {
	totalGranted, err := k.getTotalDevGranted(ctx, developerAddr)
	if err != nil {
		return err
	}

	creditBalance := k.bankKeeper.GetBalance(ctx, developerAddr, appparams.MicroCreditDenom)
	availableCredits := creditBalance.Amount.Sub(totalGranted)

	if availableCredits.LT(requestedAmount) {
		return types.ErrInvalidAddress.Wrapf(
			"insufficient available credits: requested %s, available %s (balance: %s, already granted: %s)",
			requestedAmount.String(),
			availableCredits.String(),
			creditBalance.Amount.String(),
			totalGranted.String(),
		)
	}

	return nil
}

// updateDeveloperTotalGranted adjusts stored total granted by delta; add=true adds, false subtracts.
func (k *Keeper) updateDeveloperTotalGranted(ctx context.Context, developerAddr sdk.AccAddress, delta math.Int, add bool) error {
	current, err := k.getTotalDevGranted(ctx, developerAddr)
	if err != nil {
		return err
	}

	var newTotal math.Int
	if add {
		newTotal = current.Add(delta)
	} else {
		newTotal, err = current.SafeSub(delta)
		if err != nil {
			return errorsmod.Wrapf(err, "subtract %s from developer total granted %s", delta, current)
		}
		if newTotal.IsNegative() {
			return errorsmod.Wrapf(types.ErrInvalidAmount, "subtract %s from developer total granted %s would result in negative amount", delta, current)
		}
	}

	key := types.TotalDevGrantedKey(developerAddr)
	totalDevGranted := &types.TotalDevGranted{
		Developer:    developerAddr.String(),
		TotalGranted: newTotal.Uint64(),
	}

	b := k.cdc.MustMarshal(totalDevGranted)
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store.Set(key, b)

	// Update total dev granted amount gauge
	telemetry.ModuleSetGauge(
		types.ModuleName,
		float32(newTotal.Int64()),
		developerAddr.String(),
		metrics.TotalDevGranted,
	)

	return nil
}

// getTotalDevGranted retrieves the stored total granted amount for a developer.
func (k *Keeper) getTotalDevGranted(ctx context.Context, developerAddr sdk.AccAddress) (math.Int, error) {
	key := types.TotalDevGrantedKey(developerAddr)
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))

	bz := store.Get(key)
	if bz == nil {
		return math.ZeroInt(), nil
	}

	var totalDevGranted types.TotalDevGranted
	err := k.cdc.Unmarshal(bz, &totalDevGranted)
	if err != nil {
		return math.ZeroInt(), errorsmod.Wrapf(err, "unmarshal developer total granted amount")
	}

	return math.NewIntFromUint64(totalDevGranted.TotalGranted), nil
}

// addLockupForRegistration adds more lockup to cover the required amount for user registration.
func (k *Keeper) addLockupForRegistration(ctx context.Context, developerAddr sdk.AccAddress, requiredAmount math.Int) error {
	currentTotal, err := k.getTotalDevGranted(ctx, developerAddr)
	if err != nil {
		return err
	}

	creditBalance := k.bankKeeper.GetBalance(ctx, developerAddr, appparams.MicroCreditDenom)
	availableCredits := creditBalance.Amount.Sub(currentTotal)
	additionalNeeded := requiredAmount.Sub(availableCredits)
	if additionalNeeded.LTE(math.ZeroInt()) {
		return nil
	}

	// Prefer adding to an existing validator where the developer already has lockups
	var (
		chosenValAddr sdk.ValAddress
		found         bool
		maxLockupAmt  = math.ZeroInt()
	)
	k.mustIterateLockups(ctx, func(delAddr sdk.AccAddress, valAddr sdk.ValAddress, lockup types.Lockup) {
		if !delAddr.Equals(developerAddr) {
			return
		}
		if !found || lockup.Amount.GT(maxLockupAmt) {
			found = true
			chosenValAddr = valAddr
			maxLockupAmt = lockup.Amount
		}
	})

	if !found {
		validators, err := k.stakingKeeper.GetAllValidators(ctx)
		if err != nil {
			return types.ErrInvalidAmount.Wrapf("failed to get validators: %v", err)
		}
		if len(validators) == 0 {
			return types.ErrInvalidAmount.Wrap("no validators available for lockup")
		}
		valAddr, err := sdk.ValAddressFromBech32(validators[0].GetOperator())
		if err != nil {
			return types.ErrInvalidAmount.Wrapf("invalid validator address: %v", err)
		}
		chosenValAddr = valAddr
	}

	err = k.Lock(ctx, developerAddr, chosenValAddr, additionalNeeded)
	if err != nil {
		return types.ErrInvalidAmount.Wrapf("failed to add lockup: %v", err)
	}

	return nil
}
