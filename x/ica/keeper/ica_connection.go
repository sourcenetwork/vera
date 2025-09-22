package keeper

import (
	"strings"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	icatypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/types"
	tmtypes "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"

	"github.com/sourcenetwork/sourcehub/x/ica/types"
)

// icaConnectionStore returns a prefix store for ICA connections.
func (k *Keeper) icaConnectionStore(ctx sdk.Context) prefix.Store {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	return prefix.NewStore(storeAdapter, []byte(types.ICAConnectionKeyPrefix))
}

// SetICAConnection stores an ICA connection.
func (k *Keeper) SetICAConnection(ctx sdk.Context, icaAddress, controllerAddress, controllerChainID, connectionID string) error {
	if icaAddress == "" {
		return errorsmod.Wrap(types.ErrInvalidInput, "ICA address cannot be empty")
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
func (k *Keeper) mustIterateICAConnections(ctx sdk.Context, cb func(icaAddress string, connection types.ICAConnection)) {
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

// HandleICAChannelOpen processes ICA channel opening and stores connection information.
func (k *Keeper) HandleICAChannelOpen(
	ctx sdk.Context,
	connectionID, controllerPortID string,
	icaHostKeeper types.ICAHostKeeper,
	connectionKeeper types.ConnectionKeeper,
	clientKeeper types.ClientKeeper,
) error {
	// Get ICA address (should exist after OnChanOpenTry)
	icaAddr, found := icaHostKeeper.GetInterchainAccountAddress(ctx, connectionID, controllerPortID)
	if !found {
		return errorsmod.Wrapf(
			types.ErrInvalidInput,
			"ICA address not found for connection_id %s and controller_port_id %s",
			connectionID,
			controllerPortID,
		)
	}

	// Check if connection already exists to avoid duplicates
	if _, exists := k.GetICAConnection(ctx, icaAddr); exists {
		return errorsmod.Wrapf(types.ErrInvalidInput, "ICA connection already exists for ica_address %s", icaAddr)
	}

	// Extract controller address from port ID (icacontroller-{address})
	if !strings.HasPrefix(controllerPortID, icatypes.ControllerPortPrefix) {
		return errorsmod.Wrapf(
			types.ErrInvalidInput,
			"invalid controller port ID prefix: %s, expected prefix: %s",
			controllerPortID, icatypes.ControllerPortPrefix,
		)
	}

	// Get controller chain ID from connection
	conn, found := connectionKeeper.GetConnection(ctx, connectionID)
	if !found {
		return errorsmod.Wrapf(types.ErrInvalidInput, "connection not found: %s", connectionID)
	}

	clientState, ok := clientKeeper.GetClientState(ctx, conn.ClientId)
	if !ok {
		return errorsmod.Wrapf(types.ErrInvalidInput, "client state not found: %s", conn.ClientId)
	}

	// Extract chain ID from Tendermint client state
	tmClientState, ok := clientState.(*tmtypes.ClientState)
	if !ok {
		return errorsmod.Wrapf(
			types.ErrInvalidInput,
			"client state is not Tendermint type for client_id %s, got type %T",
			conn.ClientId,
			clientState,
		)
	}

	controllerChainID := tmClientState.GetChainID()
	if controllerChainID == "" {
		return errorsmod.Wrapf(types.ErrInvalidInput, "controller chain ID is empty for client_id %s", conn.ClientId)
	}

	controllerAddr := strings.TrimPrefix(controllerPortID, icatypes.ControllerPortPrefix)
	if err := k.SetICAConnection(ctx, icaAddr, controllerAddr, controllerChainID, connectionID); err != nil {
		return errorsmod.Wrapf(err, "failed to store ICA connection for ica_address %s", icaAddr)
	}

	return nil
}
