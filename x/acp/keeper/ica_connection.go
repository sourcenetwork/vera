package keeper

import (
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

// icaConnectionStore returns a prefix store for ICA connections.
func (k *Keeper) icaConnectionStore(ctx sdk.Context) prefix.Store {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	return prefix.NewStore(storeAdapter, []byte(types.ICAConnectionKeyPrefix))
}

// SetICAConnection stores an ICA connection.
func (k *Keeper) SetICAConnection(ctx sdk.Context, icaAddress, controllerAddress, controllerChainID, connectionID string) error {
	if icaAddress == "" {
		return types.New("ICA address cannot be empty", types.ErrorType_BAD_INPUT)
	}

	connection := types.ICAConnection{
		IcaAddress:        icaAddress,
		ControllerAddress: controllerAddress,
		ControllerChainId: controllerChainID,
		ConnectionId:      connectionID,
	}

	bz, err := k.cdc.Marshal(&connection)
	if err != nil {
		return errorsmod.Wrapf(err, "marshal ICA connection for address %s", icaAddress)
	}

	store := k.icaConnectionStore(ctx)
	store.Set([]byte(icaAddress), bz)
	return nil
}

// GetICAConnection retrieves an ICA connection by address.
func (k *Keeper) GetICAConnection(ctx sdk.Context, icaAddress string) (types.ICAConnection, bool) {
	store := k.icaConnectionStore(ctx)
	bz := store.Get([]byte(icaAddress))
	if bz == nil {
		return types.ICAConnection{}, false
	}

	var connection types.ICAConnection
	k.cdc.MustUnmarshal(bz, &connection)

	return connection, true
}

// GetAllICAConnections retrieves all ICA connections.
func (k *Keeper) GetAllICAConnections(ctx sdk.Context) []types.ICAConnection {
	var connections []types.ICAConnection

	connectionsCallback := func(icaAddress string, connection types.ICAConnection) {
		connections = append(connections, connection)
	}

	k.mustIterateICAConnections(ctx, connectionsCallback)

	return connections
}

// mustIterateICAConnections iterates over all ICA connections in the store and performs the provided callback function.
func (k *Keeper) mustIterateICAConnections(ctx sdk.Context,
	cb func(icaAddress string, connection types.ICAConnection)) {

	store := k.icaConnectionStore(ctx)
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var connection types.ICAConnection
		k.cdc.MustUnmarshal(iterator.Value(), &connection)
		icaAddress := string(iterator.Key())
		cb(icaAddress, connection)
	}
}
