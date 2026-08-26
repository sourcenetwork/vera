package testutil

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sourcenetwork/vera/x/core/types"
)

// CoreKeeperStub is a stub implementation of the ICA keeper for testing.
type CoreKeeperStub struct {
	connections map[string]types.ICAConnection
}

func NewCoreKeeperStub() *CoreKeeperStub {
	return &CoreKeeperStub{
		connections: make(map[string]types.ICAConnection),
	}
}

func (iks *CoreKeeperStub) GetICAConnection(ctx sdk.Context, icaAddress string) (types.ICAConnection, bool) {
	connection, found := iks.connections[icaAddress]
	return connection, found
}

func (iks *CoreKeeperStub) SetICAConnection(ctx sdk.Context, icaAddress, controllerAddress, controllerChainID, connectionID string) error {
	iks.connections[icaAddress] = types.ICAConnection{
		IcaAddress:        icaAddress,
		ControllerAddress: controllerAddress,
		ControllerChainId: controllerChainID,
		ConnectionId:      connectionID,
	}
	return nil
}
