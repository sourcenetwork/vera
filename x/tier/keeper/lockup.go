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
// It is only used for exporting all lockups as part of the app state.
func (k Keeper) GetAllLockups(ctx context.Context) []types.Lockup {
	var lockups []types.Lockup

	lockupsCallback := func(delAddr sdk.AccAddress, valAddr sdk.ValAddress, lockup types.Lockup) {
		lockups = append(lockups, lockup)
	}

	unlockingLockupsCallback := func(delAddr sdk.AccAddress, valAddr sdk.ValAddress, creationHeight int64, lockup types.Lockup) {
		lockups = append(lockups, lockup)
	}

	k.MustIterateUnlockingLockups(ctx, unlockingLockupsCallback)
	k.MustIterateLockups(ctx, lockupsCallback)

	return lockups
}

// SaveLockup stores lockup or unlocking lockup based on the specified params.
// It is used in SubtractUnlockingLockup to override the same record considering existing creationHeight,
// as well as for importing lockups from the GenesisState.Lockups as part of the InitGenesis().
func (k Keeper) SaveLockup(ctx context.Context, unlocking bool, delAddr sdk.AccAddress, valAddr sdk.ValAddress, amt math.Int,
	creationHeight int64, unbondTime *time.Time, unlockTime *time.Time) {

	var unbTime, unlTime *time.Time
	if unbondTime != nil {
		utcTime := unbondTime.UTC()
		unbTime = &utcTime
	}
	if unlockTime != nil {
		utcTime := unlockTime.UTC()
		unlTime = &utcTime
	}
	lockup := &types.Lockup{
		DelegatorAddress: delAddr.String(),
		ValidatorAddress: valAddr.String(),
		Amount:           amt,
		CreationHeight:   creationHeight,
		UnbondTime:       unbTime,
		UnlockTime:       unlTime,
	}

	// use different key for unlocking lockups
	var key []byte
	if unlocking {
		key = types.UnlockingLockupKey(delAddr, valAddr, creationHeight)
	} else {
		key = types.LockupKey(delAddr, valAddr)
	}

	b := k.cdc.MustMarshal(lockup)
	store := k.lockupStore(ctx, unlocking)
	store.Set(key, b)
}

// SetLockup stores or updates a lockup in the state based on the key from LockupKey/UnlockingLockupKey.
// We normalize lockup times to UTC before saving to the store for consistentcy.
func (k Keeper) SetLockup(ctx context.Context, unlocking bool, delAddr sdk.AccAddress, valAddr sdk.ValAddress, amt math.Int, unbondTime *time.Time) (int64, *time.Time, time.Time) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	params := k.GetParams(ctx)
	creationHeight := sdkCtx.BlockHeight()
	epochDuration := *params.EpochDuration

	unlockTime := sdkCtx.BlockTime().Add(epochDuration * time.Duration(params.UnlockingEpochs))
	// use unbondTime from stakingKeeper.Undelegate() if present, set it to match unlockTime otherwise
	var unbTime *time.Time
	if unbondTime != nil {
		utcTime := unbondTime.UTC()
		unbTime = &utcTime
	} else {
		unbTime = &unlockTime
	}

	lockup := &types.Lockup{
		DelegatorAddress: delAddr.String(),
		ValidatorAddress: valAddr.String(),
		Amount:           amt,
		CreationHeight:   creationHeight,
		UnbondTime:       unbTime,
		UnlockTime:       &unlockTime,
	}

	// use different key for unlocking lockups
	var key []byte
	if unlocking {
		key = types.UnlockingLockupKey(delAddr, valAddr, creationHeight)
	} else {
		key = types.LockupKey(delAddr, valAddr)
	}

	b := k.cdc.MustMarshal(lockup)
	store := k.lockupStore(ctx, unlocking)
	store.Set(key, b)

	return creationHeight, unbTime, unlockTime
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

// GetUnlockingLockup returns existing unlocking lockup data if found, otherwise returns defaults.
func (k Keeper) GetUnlockingLockup(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, creationHeight int64) (
	found bool, amt math.Int, unbondTime time.Time, unlockTime time.Time) {

	key := types.UnlockingLockupKey(delAddr, valAddr, creationHeight)
	store := k.lockupStore(ctx, true)
	b := store.Get(key)
	if b == nil {
		return false, math.ZeroInt(), time.Time{}, time.Time{}
	}

	var lockup types.Lockup
	k.cdc.MustUnmarshal(b, &lockup)

	return true, lockup.Amount, *lockup.UnbondTime, *lockup.UnlockTime
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
	lockedAmt := k.GetLockupAmount(ctx, delAddr, valAddr)
	amt = amt.Add(lockedAmt)
	k.SetLockup(ctx, false, delAddr, valAddr, amt, nil)
}

// SubtractLockup subtracts provided amt from the existing delAddr/valAddr lockup.
func (k Keeper) SubtractLockup(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, amt math.Int) error {
	lockedAmt := k.GetLockupAmount(ctx, delAddr, valAddr)

	// subtracted amt must not be larger than the lockedAmt
	if amt.GT(lockedAmt) {
		return types.ErrInvalidAmount.Wrap("invalid amount")
	}

	// remove lockup record completely if subtracted amt is equal to lockedAmt
	if amt.Equal(lockedAmt) {
		k.removeLockup(ctx, delAddr, valAddr)
		return nil
	}

	// subtract amt from the lockedAmt othwerwise
	newAmt, err := lockedAmt.SafeSub(amt)
	if err != nil {
		return errorsmod.Wrapf(err, "subtract %s from locked amount %s", amt, lockedAmt)
	}

	k.SetLockup(ctx, false, delAddr, valAddr, newAmt, nil)

	return nil
}

// SubtractUnlockingLockup subtracts provided amt from the existing unlocking lockup (delAddr/valAddr/creationHeight/).
func (k Keeper) SubtractUnlockingLockup(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, creationHeight int64, amt math.Int) error {
	// get full unlocking lockup record because we must pass valid time(s) to SaveLockup
	found, lockedAmt, unbondTime, unlockTime := k.GetUnlockingLockup(ctx, delAddr, valAddr, creationHeight)

	// return early if not found
	if !found {
		return nil
	}

	// subtracted amt must not be larger than the lockedAmt
	if amt.GT(lockedAmt) {
		return types.ErrInvalidAmount.Wrap("invalid amount")
	}

	// remove lockup record completely if subtracted amt is equal to lockedAmt
	if amt.Equal(lockedAmt) {
		k.removeUnlockingLockup(ctx, delAddr, valAddr, creationHeight)
		return nil
	}

	// subtract amt from the lockedAmt othwerwise
	newAmt, err := lockedAmt.SafeSub(amt)
	if err != nil {
		return errorsmod.Wrapf(err, "subtract %s from unlocking lockup locked amount %s", amt, lockedAmt)
	}

	k.SaveLockup(ctx, true, delAddr, valAddr, newAmt, creationHeight, &unbondTime, &unlockTime)

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
func (k Keeper) IterateLockups(ctx context.Context, unlocking bool,
	cb func(delAddr sdk.AccAddress, valAddr sdk.ValAddress, creationHeight int64, lockup types.Lockup) error) error {

	store := k.lockupStore(ctx, unlocking)
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var delAddr sdk.AccAddress
		var valAddr sdk.ValAddress
		var creationHeight int64
		var lockup types.Lockup
		k.cdc.MustUnmarshal(iterator.Value(), &lockup)

		if unlocking {
			delAddr, valAddr, creationHeight = types.LockupKeyToAddressesAtHeight(iterator.Key())
		} else {
			delAddr, valAddr = types.LockupKeyToAddresses(iterator.Key())
		}

		err := cb(delAddr, valAddr, creationHeight, lockup)
		if err != nil {
			return errorsmod.Wrapf(err, "%s/%s/, amt: %s", delAddr, valAddr, lockup.Amount)
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
	cb func(delAddr sdk.AccAddress, valAddr sdk.ValAddress, creationHeight int64, lockup types.Lockup)) {

	store := k.lockupStore(ctx, true)
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var lockup types.Lockup
		k.cdc.MustUnmarshal(iterator.Value(), &lockup)
		delAddr, valAddr, creationHeight := types.LockupKeyToAddressesAtHeight(iterator.Key())
		cb(delAddr, valAddr, creationHeight, lockup)
	}
}

func (k Keeper) lockupStore(ctx context.Context, unlocking bool) prefix.Store {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	storePrefix := types.KeyPrefix(unlocking)
	return prefix.NewStore(storeAdapter, storePrefix)
}
