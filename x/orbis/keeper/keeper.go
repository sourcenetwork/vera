package keeper

import (
	"fmt"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	acpkeeper "github.com/sourcenetwork/sourcehub/x/acp/keeper"
	"github.com/sourcenetwork/sourcehub/x/orbis/types"
)

type Keeper struct {
	cdc          codec.BinaryCodec
	storeService store.KVStoreService
	logger       log.Logger

	authority string

	accountKeeper types.AccountKeeper
	acpKeeper     *acpkeeper.Keeper
}

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	logger log.Logger,
	authority string,
	accountKeeper types.AccountKeeper,
	acpKeeper *acpkeeper.Keeper,
) Keeper {
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address: %s", authority))
	}

	return Keeper{
		cdc:           cdc,
		storeService:  storeService,
		logger:        logger,
		authority:     authority,
		accountKeeper: accountKeeper,
		acpKeeper:     acpKeeper,
	}
}

func (k *Keeper) GetAuthority() string {
	return k.authority
}

func (k *Keeper) GetAcpKeeper() *acpkeeper.Keeper {
	return k.acpKeeper
}

func (k *Keeper) Logger() log.Logger {
	return k.logger.With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func namespaceID(namespace string) string {
	return types.GetNamespaceID(namespace)
}

func (k *Keeper) RingBytes(ring types.Ring) ([]byte, error) {
	return k.cdc.Marshal(&ring)
}
