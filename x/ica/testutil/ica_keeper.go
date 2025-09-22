package testutil

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sourcenetwork/sourcehub/x/ica/types"
)

// ICAKeeperStub is a stub implementation of the ICA keeper for testing.
type ICAKeeperStub struct {
	connections map[string]types.ICAConnection
}

func NewICAKeeperStub() *ICAKeeperStub {
	return &ICAKeeperStub{
		connections: make(map[string]types.ICAConnection),
	}
}

func (iks *ICAKeeperStub) GetICAConnection(ctx sdk.Context, icaAddress string) (types.ICAConnection, bool) {
	connection, found := iks.connections[icaAddress]
	return connection, found
}

func (iks *ICAKeeperStub) SetICAConnection(ctx sdk.Context, icaAddress, controllerAddress, controllerChainID, connectionID string) error {
	iks.connections[icaAddress] = types.ICAConnection{
		IcaAddress:        icaAddress,
		ControllerAddress: controllerAddress,
		ControllerChainId: controllerChainID,
		ConnectionId:      connectionID,
	}
	return nil
}
