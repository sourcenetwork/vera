package keeper

import (
	"fmt"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"

	"github.com/sourcenetwork/vera/x/epochs/types"
)

type (
	Keeper struct {
		storeService store.KVStoreService
		hooks        types.EpochHooks
		logger       log.Logger
	}
)

// NewKeeper returns a new keeper by codec and storeKey inputs.
func NewKeeper(
	storeService store.KVStoreService,
	logger log.Logger,
) *Keeper {
	return &Keeper{
		storeService: storeService,
		logger:       logger,
	}
}

// Set the epoch hooks.
func (k *Keeper) SetHooks(eh types.EpochHooks) *Keeper {
	if k.hooks != nil {
		panic("cannot set epochs hooks twice")
	}

	k.hooks = eh

	return k
}

func (k Keeper) Logger() log.Logger {
	return k.logger.With("module", fmt.Sprintf("x/%s", types.ModuleName))
}
