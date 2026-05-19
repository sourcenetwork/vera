package orbis

import (
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
)

var _ module.AppModuleSimulation = AppModule{}

// GenerateGenesisState creates a randomized GenState of the module.
func (AppModule) GenerateGenesisState(*module.SimulationState) {}

// RegisterStoreDecoder registers a decoder for orbis module's types.
func (am AppModule) RegisterStoreDecoder(_ simtypes.StoreDecoderRegistry) {}

// ProposalContents doesn't return any content functions for governance proposals.
func (AppModule) ProposalContents(_ module.SimulationState) []simtypes.WeightedProposalContent {
	return nil
}

// WeightedOperations returns all the operations from the module with their respective weights.
func (am AppModule) WeightedOperations(module.SimulationState) []simtypes.WeightedOperation {
	return nil
}

// ProposalMsgs returns msgs used for governance proposals for simulations.
func (am AppModule) ProposalMsgs(module.SimulationState) []simtypes.WeightedProposalMsg {
	return nil
}
