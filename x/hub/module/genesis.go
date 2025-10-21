package hub

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/sourcenetwork/sourcehub/x/hub/keeper"
	"github.com/sourcenetwork/sourcehub/x/hub/types"
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

	// Initialize JWS tokens
	for _, jwsToken := range genState.JwsTokens {
		token := jwsToken // Create a copy to avoid pointer issues
		if err := k.SetJWSToken(ctx, &token); err != nil {
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

	// Export JWS tokens
	jwsTokens, err := k.GetAllJWSTokens(ctx)
	if err != nil {
		panic(err)
	}
	// Convert pointers to values for proto
	for _, token := range jwsTokens {
		if token != nil {
			genesis.JwsTokens = append(genesis.JwsTokens, *token)
		}
	}

	return genesis
}
