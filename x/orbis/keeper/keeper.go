package keeper

import (
	"fmt"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"

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
	capKeeper     *capabilitykeeper.ScopedKeeper
}

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	logger log.Logger,
	authority string,
	accountKeeper types.AccountKeeper,
	acpKeeper *acpkeeper.Keeper,
	capKeeper *capabilitykeeper.ScopedKeeper,
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
		capKeeper:     capKeeper,
	}
}

func (k *Keeper) GetAuthority() string {
	return k.authority
}

func (k *Keeper) GetAcpKeeper() *acpkeeper.Keeper {
	return k.acpKeeper
}

func (k *Keeper) GetScopedKeeper() *capabilitykeeper.ScopedKeeper {
	return k.capKeeper
}

// InitializeCapabilityKeeper sets the scoped capability keeper after construction.
// Required because depinject constructs the keeper before the capability keeper is available.
// Panics if already initialized.
func (k *Keeper) InitializeCapabilityKeeper(keeper *capabilitykeeper.ScopedKeeper) {
	if k.capKeeper != nil {
		panic("capability keeper already initialized")
	}
	k.capKeeper = keeper
}

// SetPolicyId stores the module-level orbis policy id.
func (k *Keeper) SetPolicyId(ctx sdk.Context, policyId string) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store.Set([]byte(types.PolicyIdKey), []byte(policyId))
}

// GetPolicyId returns the module-level orbis policy id, or "" if not yet initialized.
func (k *Keeper) GetPolicyId(ctx sdk.Context) string {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get([]byte(types.PolicyIdKey))
	if bz == nil {
		return ""
	}
	return string(bz)
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
