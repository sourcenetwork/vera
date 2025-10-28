package keeper

import (
	"context"

	"github.com/cosmos/cosmos-sdk/runtime"

	"github.com/sourcenetwork/sourcehub/x/hub/types"
)

// SetAllowZeroFeeTxs sets the immutable allow_zero_fee_txs flag.
// Should only be called during genesis initialization.
func (k *Keeper) SetAllowZeroFeeTxs(ctx context.Context, allow bool) error {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	value := byte(0x00)
	if allow {
		value = 0x01
	}
	storeAdapter.Set(types.AllowZeroFeeTxsKey, []byte{value})
	return nil
}

// IsZeroFeeTxsAllowed returns whether zero-fee transactions are allowed.
func (k *Keeper) IsZeroFeeTxsAllowed(ctx context.Context) bool {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := storeAdapter.Get(types.AllowZeroFeeTxsKey)
	return len(bz) > 0 && bz[0] == 0x01
}

// SetIgnoreBearerAuth sets the immutable ignore_bearer_auth flag.
// Should only be called during genesis initialization.
func (k *Keeper) SetIgnoreBearerAuth(ctx context.Context, ignore bool) error {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	value := byte(0x00)
	if ignore {
		value = 0x01
	}
	storeAdapter.Set(types.IgnoreBearerAuthKey, []byte{value})
	return nil
}

// IsBearerAuthIgnored returns whether bearer auth should be ignored.
func (k *Keeper) IsBearerAuthIgnored(ctx context.Context) bool {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := storeAdapter.Get(types.IgnoreBearerAuthKey)
	return len(bz) > 0 && bz[0] == 0x01
}
