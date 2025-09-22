package ica

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/sourcenetwork/sourcehub/x/ica/keeper"
	"github.com/sourcenetwork/sourcehub/x/ica/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx sdk.Context, k *keeper.Keeper, genState types.GenesisState) {
	if err := k.SetParams(ctx, genState.Params); err != nil {
		panic(err)
	}

	// Initialize ica connections
	for _, icaConnection := range genState.IcaConnections {
		if err := k.SetICAConnection(
			ctx,
			icaConnection.IcaAddress,
			icaConnection.ControllerAddress,
			icaConnection.ControllerChainId,
			icaConnection.ConnectionId,
		); err != nil {
			panic(err)
		}
	}
}

// ExportGenesis returns the module's exported genesis.
func ExportGenesis(ctx sdk.Context, k *keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)

	// Export ica connections
	genesis.IcaConnections = k.GetAllICAConnections(ctx)

	return genesis
}
