package keeper

import (
	"context"

	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sourcenetwork/sourcehub/x/tier/types"
)

// GetAllInsuranceLockups returns all insurance lockups in the store.
func (k *Keeper) GetAllInsuranceLockups(ctx context.Context) []types.Lockup {
	var lockups []types.Lockup

	lockupsCallback := func(delAddr sdk.AccAddress, valAddr sdk.ValAddress, lockup types.Lockup) {
		lockups = append(lockups, lockup)
	}

	k.mustIterateInsuranceLockups(ctx, lockupsCallback)

	return lockups
}

// totalInsuredAmountByAddr returns the total insurance lockups amount associated with the provided delAddr.
func (k *Keeper) totalInsuredAmountByAddr(ctx context.Context, delAddr sdk.AccAddress) math.Int {
	amount := math.ZeroInt()

	cb := func(d sdk.AccAddress, valAddr sdk.ValAddress, insuranceLockup types.Lockup) {
		if d.Equals(delAddr) {
			amount = amount.Add(insuranceLockup.Amount)
		}
	}

	k.mustIterateInsuranceLockups(ctx, cb)

	return amount
}

// AddInsuranceLockup adds provided amount to the existing insurance Lockup.
func (k *Keeper) AddInsuranceLockup(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, amount math.Int) {
	insuranceLockup := k.getInsuranceLockup(ctx, delAddr, valAddr)
	if insuranceLockup != nil {
		amount = amount.Add(insuranceLockup.Amount)
	}

	k.setInsuranceLockup(ctx, delAddr, valAddr, amount)
}

// getInsuranceLockup returns existing insurance lockup, or nil if not found.
func (k *Keeper) getInsuranceLockup(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) *types.Lockup {
	key := types.LockupKey(delAddr, valAddr)
	store := k.insuranceLockupStore(ctx)
	b := store.Get(key)
	if b == nil {
		return nil
	}

	var lockup types.Lockup
	k.cdc.MustUnmarshal(b, &lockup)

	return &lockup
}

// getInsuranceLockupAmount returns existing insurance lockup amount, or math.ZeroInt() if not found.
func (k *Keeper) getInsuranceLockupAmount(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) math.Int {
	key := types.LockupKey(delAddr, valAddr)
	store := k.insuranceLockupStore(ctx)
	b := store.Get(key)
	if b == nil {
		return math.ZeroInt()
	}

	var insuranceLockup types.Lockup
	k.cdc.MustUnmarshal(b, &insuranceLockup)

	return insuranceLockup.Amount
}

// setInsuranceLockup sets an insurance lockup in the store based on the LockupKey.
func (k *Keeper) setInsuranceLockup(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, amount math.Int) {
	lockup := &types.Lockup{
		DelegatorAddress: delAddr.String(),
		ValidatorAddress: valAddr.String(),
		Amount:           amount,
	}

	key := types.LockupKey(delAddr, valAddr)
	b := k.cdc.MustMarshal(lockup)
	store := k.insuranceLockupStore(ctx)
	store.Set(key, b)
}

// removeInsuranceLockup removes existing insurance Lockup.
func (k *Keeper) removeInsuranceLockup(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) {
	key := types.LockupKey(delAddr, valAddr)
	store := k.insuranceLockupStore(ctx)
	store.Delete(key)
}

// mustIterateInsuranceLockups iterates over all insurance lockups in the store and performs the provided callback function.
func (k *Keeper) mustIterateInsuranceLockups(ctx context.Context,
	cb func(delAddr sdk.AccAddress, valAddr sdk.ValAddress, insuranceLockup types.Lockup)) {

	store := k.insuranceLockupStore(ctx)
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var insuranceLockup types.Lockup
		k.cdc.MustUnmarshal(iterator.Value(), &insuranceLockup)
		delAddr, valAddr := types.LockupKeyToAddresses(iterator.Key())
		cb(delAddr, valAddr, insuranceLockup)
	}
}

// insuranceLockupStore returns a prefix store for insurance Lockup.
func (k *Keeper) insuranceLockupStore(ctx context.Context) prefix.Store {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	return prefix.NewStore(storeAdapter, []byte(types.InsuranceLockupKeyPrefix))
}
