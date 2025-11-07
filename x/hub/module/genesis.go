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
	for i := range genState.JwsTokens {
		if err := k.SetJWSToken(ctx, &genState.JwsTokens[i]); err != nil {
			panic(err)
		}
	}

	if err := k.SetChainConfig(ctx, genState.ChainConfig); err != nil {
		panic(err)
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

	genesis.JwsTokens = make([]types.JWSTokenRecord, 0, len(jwsTokens))
	for _, token := range jwsTokens {
		if token != nil {
			genesis.JwsTokens = append(genesis.JwsTokens, *token)
		}
	}

	genesis.ChainConfig = k.GetChainConfig(ctx)

	return genesis
}
