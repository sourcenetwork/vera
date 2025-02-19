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

// GetAllockups returns all lockups in the store.
func (k Keeper) GetAllLockups(ctx context.Context) []types.Lockup {
	var lockups []types.Lockup

	lockupsCallback := func(delAddr sdk.AccAddress, valAddr sdk.ValAddress, lockup types.Lockup) {
		lockups = append(lockups, lockup)
	}

	k.MustIterateLockups(ctx, lockupsCallback)

	return lockups
}

// GetAllUnlockingLockups returns all unlocking lockups in the store.
func (k Keeper) GetAllUnlockingLockups(ctx context.Context) []types.UnlockingLockup {
	var unlockingLockups []types.UnlockingLockup

	unlockingLockupsCallback := func(delAddr sdk.AccAddress, valAddr sdk.ValAddress, creationHeight int64, lockup types.UnlockingLockup) {
		unlockingLockups = append(unlockingLockups, lockup)
	}

	k.MustIterateUnlockingLockups(ctx, unlockingLockupsCallback)

	return unlockingLockups
}

// SetLockup sets a lockup in the state based on the LockupKey.
func (k Keeper) SetLockup(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, amt math.Int) {
	lockup := &types.Lockup{
		DelegatorAddress: delAddr.String(),
		ValidatorAddress: valAddr.String(),
		Amount:           amt,
	}

	key := types.LockupKey(delAddr, valAddr)
	b := k.cdc.MustMarshal(lockup)
	store := k.lockupStore(ctx, false)
	store.Set(key, b)
}

// SetUnlockingLockup sets an unlocking lockup in the state based on the UnlockingLockupKey.
func (k Keeper) SetUnlockingLockup(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, creationHeight int64,
	amt math.Int, completionTime time.Time, unlockTime time.Time) {

	unlockingLockup := &types.UnlockingLockup{
		DelegatorAddress: delAddr.String(),
		ValidatorAddress: valAddr.String(),
		CreationHeight:   creationHeight,
		Amount:           amt,
		CompletionTime:   completionTime,
		UnlockTime:       unlockTime,
	}

	key := types.UnlockingLockupKey(delAddr, valAddr, creationHeight)
	b := k.cdc.MustMarshal(unlockingLockup)
	store := k.lockupStore(ctx, true)
	store.Set(key, b)
}

func (k Keeper) GetLockups(ctx context.Context, delAddr sdk.AccAddress) []types.Lockup {
	var lockups []types.Lockup

	cb := func(d sdk.AccAddress, valAddr sdk.ValAddress, lockup types.Lockup) {
		if d.Equals(delAddr) {
			lockups = append(lockups, lockup)
		}
	}

	k.MustIterateLockups(ctx, cb)

	return lockups
}

// GetLockup returns a pointer to existing lockup, or nil if not found.
func (k Keeper) GetLockup(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) *types.Lockup {
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
func (k Keeper) GetLockupAmount(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) math.Int {
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

// HasLockup returns true if a provided delAddr/valAddr/ lockup exists.
func (k Keeper) HasLockup(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) bool {
	key := types.LockupKey(delAddr, valAddr)
	store := k.lockupStore(ctx, false)
	b := store.Get(key)

	return b != nil
}

// HasUnlockingLockup returns true if a provided delAddr/valAddr/creationHeight unlocking lockup exists.
func (k Keeper) HasUnlockingLockup(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, creationHeight int64) bool {
	key := types.UnlockingLockupKey(delAddr, valAddr, creationHeight)
	store := k.lockupStore(ctx, true)
	b := store.Get(key)

	return b != nil
}

// GetUnlockingLockup returns existing unlocking lockup data if found, otherwise returns defaults.
func (k Keeper) GetUnlockingLockup(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, creationHeight int64) *types.UnlockingLockup {
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

// removeLockup removes existing lockup (delAddr/valAddr/).
func (k Keeper) removeLockup(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) {
	key := types.LockupKey(delAddr, valAddr)
	store := k.lockupStore(ctx, false)
	store.Delete(key)
}

// removeUnlockingLockup removes existing unlocking lockup (delAddr/valAddr/creationHeight/).
func (k Keeper) removeUnlockingLockup(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, creationHeight int64) {
	key := types.UnlockingLockupKey(delAddr, valAddr, creationHeight)
	store := k.lockupStore(ctx, true)
	store.Delete(key)
}

// AddLockup adds provided amt to the existing delAddr/valAddr lockup.
func (k Keeper) AddLockup(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, amt math.Int) {
	lockup := k.GetLockup(ctx, delAddr, valAddr)
	if lockup != nil {
		amt = amt.Add(lockup.Amount)
	}
	k.SetLockup(ctx, delAddr, valAddr, amt)
}

// SubtractLockup subtracts provided amt from the existing delAddr/valAddr lockup.
func (k Keeper) SubtractLockup(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, amt math.Int) error {
	lockup := k.GetLockup(ctx, delAddr, valAddr)
	if lockup == nil {
		return types.ErrNotFound.Wrap("subtract lockup")
	}

	// Subtracted amt must not be larger than the lockedAmt
	if amt.GT(lockup.Amount) {
		return types.ErrInvalidAmount.Wrap("subtract lockup")
	}

	// Remove lockup record completely if subtracted amt is equal to lockedAmt
	if amt.Equal(lockup.Amount) {
		k.removeLockup(ctx, delAddr, valAddr)
		return nil
	}

	// Subtract amt from the lockedAmt othwerwise
	newAmt, err := lockup.Amount.SafeSub(amt)
	if err != nil {
		return errorsmod.Wrapf(err, "subtract %s from locked amount %s", amt, lockup.Amount)
	}

	k.SetLockup(ctx, delAddr, valAddr, newAmt)

	return nil
}

// SubtractUnlockingLockup subtracts provided amt from the existing unlocking lockup (delAddr/valAddr/creationHeight/).
func (k Keeper) SubtractUnlockingLockup(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, creationHeight int64, amt math.Int) error {
	unlockingLockup := k.GetUnlockingLockup(ctx, delAddr, valAddr, creationHeight)
	if unlockingLockup == nil {
		return types.ErrNotFound.Wrap("subtract unlocking lockup")
	}

	// Subtracted amt must not be larger than the lockedAmt
	if amt.GT(unlockingLockup.Amount) {
		return types.ErrInvalidAmount.Wrap("subtract unlocking lockup")
	}

	// Remove lockup record completely if subtracted amt is equal to lockedAmt
	if amt.Equal(unlockingLockup.Amount) {
		k.removeUnlockingLockup(ctx, delAddr, valAddr, creationHeight)
		return nil
	}

	// Subtract amt from the lockedAmt othwerwise
	newAmt, err := unlockingLockup.Amount.SafeSub(amt)
	if err != nil {
		return errorsmod.Wrapf(err, "subtract %s from unlocking lockup with amount %s", amt, unlockingLockup.Amount)
	}

	k.SetUnlockingLockup(ctx, delAddr, valAddr, creationHeight, newAmt, unlockingLockup.CompletionTime, unlockingLockup.UnlockTime)

	return nil
}

// TotalAmountByAddr returns the total amount delegated by the provided delAddr.
func (k Keeper) TotalAmountByAddr(ctx context.Context, delAddr sdk.AccAddress) math.Int {
	amt := math.ZeroInt()

	cb := func(d sdk.AccAddress, valAddr sdk.ValAddress, lockup types.Lockup) {
		if d.Equals(delAddr) {
			amt = amt.Add(lockup.Amount)
		}
	}

	k.MustIterateLockups(ctx, cb)

	return amt
}

// IterateLockups iterates over all lockups in the store and performs the provided callback function.
// The iterator itself doesn't return an error, but the callback does.
// If the callback returns an error, the iteration stops and the error is returned.
func (k Keeper) IterateLockups(ctx context.Context, cb func(delAddr sdk.AccAddress, valAddr sdk.ValAddress, lockup types.Lockup) error) error {
	store := k.lockupStore(ctx, false)
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var delAddr sdk.AccAddress
		var valAddr sdk.ValAddress
		var lockup types.Lockup
		k.cdc.MustUnmarshal(iterator.Value(), &lockup)

		delAddr, valAddr = types.LockupKeyToAddresses(iterator.Key())

		err := cb(delAddr, valAddr, lockup)
		if err != nil {
			return errorsmod.Wrapf(err, "%s/%s/, amt: %s", delAddr, valAddr, lockup.Amount)
		}
	}

	return nil
}

// IterateUnlockingLockups iterates over all unlocking lockups in the store and performs the provided callback function.
// The iterator itself doesn't return an error, but the callback does.
// If the callback returns an error, the iteration stops and the error is returned.
func (k Keeper) IterateUnlockingLockups(ctx context.Context,
	cb func(delAddr sdk.AccAddress, valAddr sdk.ValAddress, creationHeight int64, lockup types.UnlockingLockup) error) error {

	store := k.lockupStore(ctx, true)
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var delAddr sdk.AccAddress
		var valAddr sdk.ValAddress
		var creationHeight int64
		var unlockingLockup types.UnlockingLockup
		k.cdc.MustUnmarshal(iterator.Value(), &unlockingLockup)

		delAddr, valAddr, creationHeight = types.UnlockingLockupKeyToAddressesAtHeight(iterator.Key())

		err := cb(delAddr, valAddr, creationHeight, unlockingLockup)
		if err != nil {
			return errorsmod.Wrapf(err, "%s/%s/, amt: %s", delAddr, valAddr, unlockingLockup.Amount)
		}
	}

	return nil
}

// MustIterateLockups iterates over all lockups in the store and performs the provided callback function.
func (k Keeper) MustIterateLockups(ctx context.Context,
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

// MustIterateUnlockingLockups iterates over all unlocking lockups in the store and performs the provided callback function.
func (k Keeper) MustIterateUnlockingLockups(ctx context.Context,
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

func (k Keeper) lockupStore(ctx context.Context, unlocking bool) prefix.Store {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	storePrefix := types.KeyPrefix(unlocking)
	return prefix.NewStore(storeAdapter, storePrefix)
}
