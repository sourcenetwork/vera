package keeper

import (
	"context"

	"github.com/cosmos/cosmos-sdk/runtime"

	"github.com/sourcenetwork/vera/x/core/types"
)

// SetChainConfig sets the immutable chain config
// specified at genesis
func (k *Keeper) SetChainConfig(ctx context.Context, cfg types.ChainConfig) error {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := storeAdapter.Get(types.ChainConfigKey)
	if bz != nil {
		return types.ErrConfigSet
	}
	bz = k.cdc.MustMarshal(&cfg)
	storeAdapter.Set(types.ChainConfigKey, bz)
	return nil
}

func (k *Keeper) GetChainConfig(ctx context.Context) types.ChainConfig {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := storeAdapter.Get(types.ChainConfigKey)
	if bz == nil {
		return types.DefaultGenesis().ChainConfig
	}
	cfg := types.ChainConfig{}
	k.cdc.MustUnmarshal(bz, &cfg)
	return cfg
}
