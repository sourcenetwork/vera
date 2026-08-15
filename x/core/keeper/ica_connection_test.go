package keeper

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	connectiontypes "github.com/cosmos/ibc-go/v10/modules/core/03-connection/types"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
	tmtypes "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"
	"github.com/sourcenetwork/vera/x/core/types"
	"github.com/stretchr/testify/require"
)

type mockICAHostKeeper struct {
	icaAddress string
	found      bool
}

func (m *mockICAHostKeeper) GetInterchainAccountAddress(ctx sdk.Context, connectionID, portID string) (string, bool) {
	return m.icaAddress, m.found
}

type mockConnectionKeeper struct {
	connection connectiontypes.ConnectionEnd
	found      bool
}

func (m *mockConnectionKeeper) GetConnection(ctx sdk.Context, connectionID string) (connectiontypes.ConnectionEnd, bool) {
	return m.connection, m.found
}

type mockClientKeeper struct {
	clientState ibcexported.ClientState
	found       bool
}

func (m *mockClientKeeper) GetClientState(ctx sdk.Context, clientID string) (ibcexported.ClientState, bool) {
	return m.clientState, m.found
}

func TestSetAndGetICAConnection(t *testing.T) {
	ctx, k, _ := setupKeeper(t)

	icaAddress := "source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et"
	controllerAddress := "shinzo1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9"
	controllerChainID := "shinzo-1"
	connectionID := "connection-0"

	err := k.SetICAConnection(ctx, icaAddress, controllerAddress, controllerChainID, connectionID)
	require.NoError(t, err)

	connection, found := k.GetICAConnection(ctx, icaAddress)
	require.True(t, found)
	require.Equal(t, icaAddress, connection.IcaAddress)
	require.Equal(t, controllerAddress, connection.ControllerAddress)
	require.Equal(t, controllerChainID, connection.ControllerChainId)
	require.Equal(t, connectionID, connection.ConnectionId)
}

func TestGetICAConnectionNotFound(t *testing.T) {
	ctx, k, _ := setupKeeper(t)

	icaAddress := "source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et"

	connection, found := k.GetICAConnection(ctx, icaAddress)
	require.False(t, found)
	require.Equal(t, types.ICAConnection{}, connection)
}

func TestGetAllICAConnections(t *testing.T) {
	ctx, k, _ := setupKeeper(t)

	icaAddress1 := "source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et"
	controllerAddress1 := "shinzo1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9"
	controllerChainID1 := "shinzo-1"
	connectionID1 := "connection-0"

	icaAddress2 := "source1n34fvpteuanu2nx2a4hql4jvcrcnal3gsrjppy"
	controllerAddress2 := "shinzo1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm"
	controllerChainID2 := "shinzo-2"
	connectionID2 := "connection-1"

	err := k.SetICAConnection(ctx, icaAddress1, controllerAddress1, controllerChainID1, connectionID1)
	require.NoError(t, err)
	err = k.SetICAConnection(ctx, icaAddress2, controllerAddress2, controllerChainID2, connectionID2)
	require.NoError(t, err)

	connections := k.GetAllICAConnections(ctx)
	require.Len(t, connections, 2)

	var connection1, connection2 types.ICAConnection
	for _, conn := range connections {
		if conn.IcaAddress == icaAddress1 {
			connection1 = conn
		} else if conn.IcaAddress == icaAddress2 {
			connection2 = conn
		}
	}

	require.Equal(t, icaAddress1, connection1.IcaAddress)
	require.Equal(t, controllerAddress1, connection1.ControllerAddress)
	require.Equal(t, controllerChainID1, connection1.ControllerChainId)
	require.Equal(t, connectionID1, connection1.ConnectionId)

	require.Equal(t, icaAddress2, connection2.IcaAddress)
	require.Equal(t, controllerAddress2, connection2.ControllerAddress)
	require.Equal(t, controllerChainID2, connection2.ControllerChainId)
	require.Equal(t, connectionID2, connection2.ConnectionId)
}

func TestMustIterateICAConnections(t *testing.T) {
	ctx, k, _ := setupKeeper(t)

	icaAddress := "source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et"
	controllerAddress := "shinzo1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9"
	controllerChainID := "shinzo-1"
	connectionID := "connection-0"

	err := k.SetICAConnection(ctx, icaAddress, controllerAddress, controllerChainID, connectionID)
	require.NoError(t, err)

	count := 0
	k.mustIterateICAConnections(ctx, func(iterIcaAddress string, connection types.ICAConnection) {
		require.Equal(t, icaAddress, iterIcaAddress)
		require.Equal(t, icaAddress, connection.IcaAddress)
		require.Equal(t, controllerAddress, connection.ControllerAddress)
		require.Equal(t, controllerChainID, connection.ControllerChainId)
		require.Equal(t, connectionID, connection.ConnectionId)
		count++
	})

	require.Equal(t, 1, count)
}

func TestSetICAConnectionOverwrite(t *testing.T) {
	ctx, k, _ := setupKeeper(t)

	icaAddress := "source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et"
	originalControllerAddress := "shinzo1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9"
	originalControllerChainID := "shinzo-1"
	originalConnectionID := "connection-0"

	updatedControllerAddress := "shinzo1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm"
	updatedControllerChainID := "shinzo-2"
	updatedConnectionID := "connection-1"

	// Set original connection
	err := k.SetICAConnection(ctx, icaAddress, originalControllerAddress, originalControllerChainID, originalConnectionID)
	require.NoError(t, err)

	// Overwrite with updated values
	err = k.SetICAConnection(ctx, icaAddress, updatedControllerAddress, updatedControllerChainID, updatedConnectionID)
	require.NoError(t, err)

	// Verify updated values
	connection, found := k.GetICAConnection(ctx, icaAddress)
	require.True(t, found)
	require.Equal(t, icaAddress, connection.IcaAddress)
	require.Equal(t, updatedControllerAddress, connection.ControllerAddress)
	require.Equal(t, updatedControllerChainID, connection.ControllerChainId)
	require.Equal(t, updatedConnectionID, connection.ConnectionId)
}

func TestSetICAConnectionEmptyAddress(t *testing.T) {
	ctx, k, _ := setupKeeper(t)

	err := k.SetICAConnection(ctx, "", "shinzo1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9", "shinzo-1", "connection-0")
	require.Error(t, err)
	require.Contains(t, err.Error(), "ICA address cannot be empty")

	// Verify no connection was stored
	connection, found := k.GetICAConnection(ctx, "")
	require.False(t, found)
	require.Equal(t, types.ICAConnection{}, connection)
}

func TestHandleICAChannelOpen(t *testing.T) {
	ctx, k, _ := setupKeeper(t)

	icaHostKeeper := &mockICAHostKeeper{
		icaAddress: "source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et",
		found:      true,
	}

	connectionKeeper := &mockConnectionKeeper{
		connection: connectiontypes.ConnectionEnd{
			ClientId: "07-tendermint-0",
		},
		found: true,
	}

	clientKeeper := &mockClientKeeper{
		clientState: &tmtypes.ClientState{
			ChainId: "shinzo-1",
		},
		found: true,
	}

	connectionID := "connection-0"
	controllerPortID := "icacontroller-source1cy0p47z24ejzvq55pu3lesxwf73xnrnd0lyxme"
	err := k.HandleICAChannelOpen(ctx, connectionID, controllerPortID, icaHostKeeper, connectionKeeper, clientKeeper)
	require.NoError(t, err)

	// Verify the connection was stored
	connection, found := k.GetICAConnection(ctx, icaHostKeeper.icaAddress)
	require.True(t, found)
	require.Equal(t, icaHostKeeper.icaAddress, connection.IcaAddress)
	require.Equal(t, "source1cy0p47z24ejzvq55pu3lesxwf73xnrnd0lyxme", connection.ControllerAddress)
	require.Equal(t, "shinzo-1", connection.ControllerChainId)
	require.Equal(t, connectionID, connection.ConnectionId)
}
