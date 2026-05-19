package orbis

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/sourcenetwork/sourcehub/x/orbis/keeper"
	"github.com/sourcenetwork/sourcehub/x/orbis/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx sdk.Context, k *keeper.Keeper, genState types.GenesisState) {
	if err := k.SetParams(ctx, genState.Params); err != nil {
		panic(err)
	}

	for _, ring := range genState.Rings {
		k.SetRing(ctx, ring)
	}
	for _, document := range genState.Documents {
		k.SetDocument(ctx, document)
	}
	for _, keyDerivation := range genState.KeyDerivations {
		k.SetKeyDerivation(ctx, keyDerivation)
	}
}

// ExportGenesis returns the module's exported genesis.
func ExportGenesis(ctx sdk.Context, k *keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)
	genesis.Rings = k.GetAllRings(ctx)
	genesis.Documents = k.GetAllDocuments(ctx)
	genesis.KeyDerivations = k.GetAllKeyDerivations(ctx)
	return genesis
}
