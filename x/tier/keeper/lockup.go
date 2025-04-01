package keeper

import (
	"context"
	"time"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/sourcenetwork/sourcehub/x/tier/types"
)

// GetAllLockups returns all lockups in the store.
func (k *Keeper) GetAllLockups(ctx context.Context) []types.Lockup {
	var lockups []types.Lockup

	lockupsCallback := func(delAddr sdk.AccAddress, valAddr sdk.ValAddress, lockup types.Lockup) {
		lockups = append(lockups, lockup)
	}

	k.mustIterateLockups(ctx, lockupsCallback)

	return lockups
}

// GetAllUnlockingLockups returns all unlocking lockups in the store.
func (k *Keeper) GetAllUnlockingLockups(ctx context.Context) []types.UnlockingLockup {
	var unlockingLockups []types.UnlockingLockup

	unlockingLockupsCallback := func(delAddr sdk.AccAddress, valAddr sdk.ValAddress, creationHeight int64, lockup types.UnlockingLockup) {
		unlockingLockups = append(unlockingLockups, lockup)
	}

	k.mustIterateUnlockingLockups(ctx, unlockingLockupsCallback)

	return unlockingLockups
}

// setLockup sets a lockup in the store based on the LockupKey.
func (k *Keeper) setLockup(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, amount math.Int) {
	lockup := &types.Lockup{
		DelegatorAddress: delAddr.String(),
		ValidatorAddress: valAddr.String(),
		Amount:           amount,
	}

	key := types.LockupKey(delAddr, valAddr)
	b := k.cdc.MustMarshal(lockup)
	store := k.lockupStore(ctx, false)
	store.Set(key, b)
}

// SetUnlockingLockup sets an unlocking lockup in the store based on the UnlockingLockupKey.
func (k *Keeper) SetUnlockingLockup(
	ctx context.Context,
	delAddr sdk.AccAddress,
	valAddr sdk.ValAddress,
	creationHeight int64,
	amount math.Int,
	completionTime, unlockTime time.Time,
) {
	unlockingLockup := &types.UnlockingLockup{
		DelegatorAddress: delAddr.String(),
		ValidatorAddress: valAddr.String(),
		CreationHeight:   creationHeight,
		Amount:           amount,
		CompletionTime:   completionTime,
		UnlockTime:       unlockTime,
	}

	key := types.UnlockingLockupKey(delAddr, valAddr, creationHeight)
	b := k.cdc.MustMarshal(unlockingLockup)
	store := k.lockupStore(ctx, true)
	store.Set(key, b)
}

// GetLockup returns a pointer to existing lockup, or nil if not found.
func (k *Keeper) GetLockup(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) *types.Lockup {
	key := types.LockupKey(delAddr, valAddr)
	store := k.lockupStore(ctx, false)
	b := store.Get(key)
	if b == nil {
		return nil
	}

	var lockup types.Lockup
	k.cdc.MustUnmarshal(b, &lockup)

	return &lockup
}

// GetLockupAmount returns existing lockup amount, or math.ZeroInt() if not found.
func (k *Keeper) GetLockupAmount(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) math.Int {
	key := types.LockupKey(delAddr, valAddr)
	store := k.lockupStore(ctx, false)
	b := store.Get(key)
	if b == nil {
		return math.ZeroInt()
	}

	var lockup types.Lockup
	k.cdc.MustUnmarshal(b, &lockup)

	return lockup.Amount
}

// hasLockup checks if Lockup with specified delAddr/valAddr exists.
func (k *Keeper) hasLockup(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) bool {
	key := types.LockupKey(delAddr, valAddr)
	store := k.lockupStore(ctx, false)
	b := store.Get(key)

	return b != nil
}

// HasUnlockingLockup checks if UnlockingLockup with specified delAddr/valAddr/creationHeight exists.
func (k *Keeper) HasUnlockingLockup(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, creationHeight int64) bool {
	key := types.UnlockingLockupKey(delAddr, valAddr, creationHeight)
	store := k.lockupStore(ctx, true)
	b := store.Get(key)

	return b != nil
}

// GetUnlockingLockup returns the UnlockingLockup if it exists, or nil otherwise.
func (k *Keeper) GetUnlockingLockup(
	ctx context.Context,
	delAddr sdk.AccAddress,
	valAddr sdk.ValAddress,
	creationHeight int64,
) *types.UnlockingLockup {
	key := types.UnlockingLockupKey(delAddr, valAddr, creationHeight)
	store := k.lockupStore(ctx, true)
	b := store.Get(key)
	if b == nil {
		return nil
	}

	var unlockingLockup types.UnlockingLockup
	k.cdc.MustUnmarshal(b, &unlockingLockup)

	return &unlockingLockup
}

// removeLockup removes existing Lockup (delAddr/valAddr/).
func (k *Keeper) removeLockup(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) {
	key := types.LockupKey(delAddr, valAddr)
	store := k.lockupStore(ctx, false)
	store.Delete(key)
}

// removeUnlockingLockup removes existing unlocking lockup (delAddr/valAddr/creationHeight/).
func (k *Keeper) removeUnlockingLockup(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, creationHeight int64) {
	key := types.UnlockingLockupKey(delAddr, valAddr, creationHeight)
	store := k.lockupStore(ctx, true)
	store.Delete(key)
}

// AddLockup adds provided amount to the existing Lockup.
func (k *Keeper) AddLockup(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, amount math.Int) error {
	total := k.GetTotalLockupsAmount(ctx)
	total = total.Add(amount)
	err := k.setTotalLockupsAmount(ctx, total)
	if err != nil {
		return err
	}

	lockup := k.GetLockup(ctx, delAddr, valAddr)
	if lockup != nil {
		amount = amount.Add(lockup.Amount)
	}

	k.setLockup(ctx, delAddr, valAddr, amount)

	return nil
}

// subtractLockup subtracts provided amount from the existing Lockup.
func (k *Keeper) subtractLockup(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, amount math.Int) error {
	lockup := k.GetLockup(ctx, delAddr, valAddr)
	if lockup == nil {
		return types.ErrNotFound.Wrap("subtract lockup")
	}

	// Adjust the actual lockup amount by insurance lockup amount
	insuranceLockupAmount := math.ZeroInt()
	insuranceLockup := k.getInsuranceLockup(ctx, delAddr, valAddr)
	if insuranceLockup != nil {
		insuranceLockupAmount = insuranceLockup.Amount
	}

	// Subtracted amount plus insurance lockup amount must not be larger than the lockedAmt
	if amount.Add(insuranceLockupAmount).GT(lockup.Amount) {
		return types.ErrInvalidAmount.Wrap("subtract lockup")
	}

	total := k.GetTotalLockupsAmount(ctx)
	newTotal, err := total.SafeSub(amount)
	if err != nil {
		return errorsmod.Wrapf(err, "subtract %s from total lockups amount %s", amount, total)
	}

	err = k.setTotalLockupsAmount(ctx, newTotal)
	if err != nil {
		return err
	}

	// Remove lockup record completely if subtracted amount is equal to lockedAmt
	if amount.Equal(lockup.Amount) {
		k.removeLockup(ctx, delAddr, valAddr)
		return nil
	}

	// Subtract amount from the lockedAmt otherwise
	newAmt, err := lockup.Amount.SafeSub(amount)
	if err != nil {
		return errorsmod.Wrapf(err, "subtract %s from locked amount %s", amount, lockup.Amount)
	}

	k.setLockup(ctx, delAddr, valAddr, newAmt)

	return nil
}

// subtractUnlockingLockup subtracts provided amount from the existing UnlockingLockup.
func (k *Keeper) subtractUnlockingLockup(
	ctx context.Context,
	delAddr sdk.AccAddress,
	valAddr sdk.ValAddress,
	creationHeight int64,
	amount math.Int,
) error {
	unlockingLockup := k.GetUnlockingLockup(ctx, delAddr, valAddr, creationHeight)
	if unlockingLockup == nil {
		return types.ErrNotFound.Wrap("subtract unlocking lockup")
	}

	// Subtracted amount must not be larger than the lockedAmt
	if amount.GT(unlockingLockup.Amount) {
		return types.ErrInvalidAmount.Wrap("subtract unlocking lockup")
	}

	// Remove lockup record completely if subtracted amount is equal to lockedAmt
	if amount.Equal(unlockingLockup.Amount) {
		k.removeUnlockingLockup(ctx, delAddr, valAddr, creationHeight)
		return nil
	}

	// Subtract amount from the lockedAmt otherwise
	newAmt, err := unlockingLockup.Amount.SafeSub(amount)
	if err != nil {
		return errorsmod.Wrapf(err, "subtract %s from unlocking lockup with amount %s", amount, unlockingLockup.Amount)
	}

	k.SetUnlockingLockup(ctx, delAddr, valAddr, creationHeight, newAmt, unlockingLockup.CompletionTime, unlockingLockup.UnlockTime)

	return nil
}

// totalLockedAmountByAddr returns the total lockup amount by the provided delAddr.
func (k *Keeper) totalLockedAmountByAddr(ctx context.Context, delAddr sdk.AccAddress) math.Int {
	amount := math.ZeroInt()

	cb := func(d sdk.AccAddress, valAddr sdk.ValAddress, lockup types.Lockup) {
		if d.Equals(delAddr) {
			amount = amount.Add(lockup.Amount)
		}
	}

	k.mustIterateLockups(ctx, cb)

	return amount
}

// iterateUnlockingLockups iterates over all unlocking lockups in the store and performs the provided callback function.
// The iterator itself doesn't return an error, but the callback does.
// If the callback returns an error, the iteration stops and the error is returned.
func (k *Keeper) iterateUnlockingLockups(ctx context.Context,
	cb func(delAddr sdk.AccAddress, valAddr sdk.ValAddress, creationHeight int64, lockup types.UnlockingLockup) error) error {

	store := k.lockupStore(ctx, true)
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var unlockingLockup types.UnlockingLockup
		k.cdc.MustUnmarshal(iterator.Value(), &unlockingLockup)

		delAddr, valAddr, creationHeight := types.UnlockingLockupKeyToAddressesAtHeight(iterator.Key())

		err := cb(delAddr, valAddr, creationHeight, unlockingLockup)
		if err != nil {
			return errorsmod.Wrapf(err, "%s/%s/%d, amount: %s", delAddr, valAddr, creationHeight, unlockingLockup.Amount)
		}
	}

	return nil
}

// mustIterateLockups iterates over all lockups in the store and performs the provided callback function.
func (k *Keeper) mustIterateLockups(ctx context.Context,
	cb func(delAddr sdk.AccAddress, valAddr sdk.ValAddress, lockup types.Lockup)) {

	store := k.lockupStore(ctx, false)
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var lockup types.Lockup
		k.cdc.MustUnmarshal(iterator.Value(), &lockup)
		delAddr, valAddr := types.LockupKeyToAddresses(iterator.Key())
		cb(delAddr, valAddr, lockup)
	}
}

// mustIterateUnlockingLockups iterates over all unlocking lockups in the store and performs the provided callback function.
func (k *Keeper) mustIterateUnlockingLockups(ctx context.Context,
	cb func(delAddr sdk.AccAddress, valAddr sdk.ValAddress, creationHeight int64, unlockingLockup types.UnlockingLockup)) {

	store := k.lockupStore(ctx, true)
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var unlockingLockup types.UnlockingLockup
		k.cdc.MustUnmarshal(iterator.Value(), &unlockingLockup)
		delAddr, valAddr, creationHeight := types.UnlockingLockupKeyToAddressesAtHeight(iterator.Key())
		cb(delAddr, valAddr, creationHeight, unlockingLockup)
	}
}

// lockupStore returns a prefix store for Lockup / UnlockingLockup.
func (k *Keeper) lockupStore(ctx context.Context, unlocking bool) prefix.Store {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	storePrefix := types.KeyPrefix(unlocking)
	return prefix.NewStore(storeAdapter, storePrefix)
}

// GetTotalLockupsAmount retrieves the total lockup amount from the store.
func (k *Keeper) GetTotalLockupsAmount(ctx context.Context) (total math.Int) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get(types.TotalLockupsKey)
	if bz == nil {
		return math.ZeroInt()
	}

	err := total.Unmarshal(bz)
	if err != nil {
		return math.ZeroInt()
	}

	if total.IsNegative() {
		return math.ZeroInt()
	}

	return total
}

// setTotalLockupsAmount updates the total lockup amount in the store.
func (k *Keeper) setTotalLockupsAmount(ctx context.Context, total math.Int) error {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz, err := total.Marshal()
	if err != nil {
		return errorsmod.Wrapf(err, "marshal total lockups amount")
	}

	store.Set(types.TotalLockupsKey, bz)

	return nil
}

// adjustLockups iterates over existing lockups and adjusts them based on the provided slashingRate.
// If non-zero coverageRate is provided, creates/updates associated insurance lockup records.
func (k *Keeper) adjustLockups(ctx context.Context, validatorAddr sdk.ValAddress, slashingRate, coverageRate math.LegacyDec) error {
	store := k.lockupStore(ctx, false)
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		delAddr, valAddr := types.LockupKeyToAddresses(iterator.Key())

		if !valAddr.Equals(validatorAddr) {
			continue
		}

		err := k.adjustLockup(ctx, delAddr, valAddr, slashingRate, coverageRate)
		if err != nil {
			return err
		}
	}

	return nil
}

// adjustLockup adjusts the existing lockup amount based on the provided slashingRate.
// If non-zero coverageRate is provided, creates/updates associated insurance lockup record.
func (k *Keeper) adjustLockup(
	ctx context.Context,
	delAddr sdk.AccAddress,
	valAddr sdk.ValAddress,
	slashingRate, coverageRate math.LegacyDec,
) error {
	lockup := k.GetLockup(ctx, delAddr, valAddr)
	if lockup == nil {
		return types.ErrNotFound.Wrap("adjust lockup")
	}

	// Calculate lockup amount after slashing, rounding down to ensure unlocks are handled correctly
	amountAfterSlashing := lockup.Amount.ToLegacyDec().Mul(slashingRate).TruncateInt()

	// No need to update the store if the amount after slashing equals to original lockup amount
	if amountAfterSlashing.Equal(lockup.Amount) {
		return nil
	}

	// Get existing total lockups amount and subtract lockup.Amount from it
	total := k.GetTotalLockupsAmount(ctx)
	newTotal, err := total.SafeSub(lockup.Amount)
	if err != nil {
		return errorsmod.Wrapf(err, "subtract %s from total lockups amount %s", lockup.Amount, total)
	}

	// Add amountAfterSlashing to newTotal and update the store
	newTotal = newTotal.Add(amountAfterSlashing)
	err = k.setTotalLockupsAmount(ctx, newTotal)
	if err != nil {
		return err
	}

	// Remove lockup record and associated insurance lockup if amount after slashing is zero
	if amountAfterSlashing.IsZero() {
		k.removeLockup(ctx, delAddr, valAddr)
		k.removeInsuranceLockup(ctx, delAddr, valAddr)
		return nil
	}

	// Update the lockup with amount after slashing otherwise
	k.setLockup(ctx, delAddr, valAddr, amountAfterSlashing)

	// Calculate covered amount and update insurance lockup if coverage rate is provided
	if coverageRate.IsPositive() {
		coveredAmount := lockup.Amount.ToLegacyDec().Mul(coverageRate).TruncateInt()
		insuranceLockup := k.getInsuranceLockup(ctx, delAddr, valAddr)
		if insuranceLockup != nil {
			coveredAmount = coveredAmount.Add(insuranceLockup.Amount)
		}
		k.setInsuranceLockup(ctx, delAddr, valAddr, coveredAmount)
	}

	return nil
}
