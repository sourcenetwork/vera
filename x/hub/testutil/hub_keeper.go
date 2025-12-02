package testutil

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sourcenetwork/sourcehub/x/hub/types"
)

// HubKeeperStub is a stub implementation of the ICA keeper for testing.
type HubKeeperStub struct {
	connections map[string]types.ICAConnection
}

func NewHubKeeperStub() *HubKeeperStub {
	return &HubKeeperStub{
		connections: make(map[string]types.ICAConnection),
	}
}

func (iks *HubKeeperStub) GetICAConnection(ctx sdk.Context, icaAddress string) (types.ICAConnection, bool) {
	connection, found := iks.connections[icaAddress]
	return connection, found
}

func (iks *HubKeeperStub) SetICAConnection(ctx sdk.Context, icaAddress, controllerAddress, controllerChainID, connectionID string) error {
	iks.connections[icaAddress] = types.ICAConnection{
		IcaAddress:        icaAddress,
		ControllerAddress: controllerAddress,
		ControllerChainId: controllerChainID,
		ConnectionId:      connectionID,
	}
	return nil
}
