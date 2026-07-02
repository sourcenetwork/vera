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
		if ring.Reporting.Equal(types.ReportingConfig{}) {
			ring.Reporting = types.ReportingConfigFromDefaults(genState.Params.DefaultReporting)
		}
		k.SetRing(ctx, ring)
	}
	for _, document := range genState.Documents {
		k.SetDocument(ctx, document)
	}
	for _, keyDerivation := range genState.KeyDerivations {
		k.SetKeyDerivation(ctx, keyDerivation)
	}
	for _, entry := range genState.NodeInfos {
		if entry.NodeInfo != nil {
			k.SetNodeInfo(ctx, entry.NodeKey, *entry.NodeInfo)
		}
	}
	for _, entry := range genState.NodeDemerits {
		k.SetNodeDemerits(ctx, entry.RingId, entry.NodeKey, entry.Points, entry.WindowStartedAt)
	}
	for _, entry := range genState.AcceptedReportPairs {
		k.SetAcceptedReportPair(ctx, entry.ReportId, entry.SessionId, entry.ExpiresAt)
	}
}

// ExportGenesis returns the module's exported genesis.
func ExportGenesis(ctx sdk.Context, k *keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)
	genesis.Rings = k.GetAllRings(ctx)
	genesis.Documents = k.GetAllDocuments(ctx)
	genesis.KeyDerivations = k.GetAllKeyDerivations(ctx)
	genesis.NodeInfos = k.GetAllNodeInfos(ctx)
	genesis.NodeDemerits = k.GetAllNodeDemerits(ctx)
	genesis.AcceptedReportPairs = k.GetAllAcceptedReportPairs(ctx)
	return genesis
}
