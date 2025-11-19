package acp

import (
	"math/rand"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"

	"github.com/sourcenetwork/sourcehub/testutil/sample"
	acpsimulation "github.com/sourcenetwork/sourcehub/x/acp/simulation"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

// avoid unused import issue
var (
	_ = acpsimulation.FindAccount
	_ = rand.Rand{}
	_ = sample.AccAddress
	_ = sdk.AccAddress{}
	_ = simulation.MsgEntryKind
)

const (
	opWeightMsgCreatePolicy = "op_weight_msg_create_policy"
	// TODO: Determine the simulation weight value
	defaultWeightMsgCreatePolicy int = 100

	opWeightMsgCheckAccess = "op_weight_msg_check_access"
	// TODO: Determine the simulation weight value
	defaultWeightMsgCheckAccess int = 100

	opWeightMsgPolicyCmd = "op_weight_msg_policy_cmd"
	// TODO: Determine the simulation weight value
	defaultWeightMsgPolicyCmd int = 100

	opWeightMsgMsgEditPolicy = "op_weight_msg_msg_edit_policy"
	// TODO: Determine the simulation weight value
	defaultWeightMsgMsgEditPolicy int = 100

	// this line is used by starport scaffolding # simapp/module/const
)

// GenerateGenesisState creates a randomized GenState of the module.
func (AppModule) GenerateGenesisState(simState *module.SimulationState) {
	accs := make([]string, len(simState.Accounts))
	for i, acc := range simState.Accounts {
		accs[i] = acc.Address.String()
	}
	acpGenesis := types.GenesisState{
		Params: types.DefaultParams(),
		// this line is used by starport scaffolding # simapp/module/genesisState
	}
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(&acpGenesis)
}

// RegisterStoreDecoder registers a decoder.
func (am AppModule) RegisterStoreDecoder(_ simtypes.StoreDecoderRegistry) {}

// ProposalContents doesn't return any content functions for governance proposals.
func (AppModule) ProposalContents(_ module.SimulationState) []simtypes.WeightedProposalContent {
	return nil
}

// WeightedOperations returns the all the gov module operations with their respective weights.
func (am AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
	operations := make([]simtypes.WeightedOperation, 0)

	var weightMsgCreatePolicy int
	simState.AppParams.GetOrGenerate(opWeightMsgCreatePolicy, &weightMsgCreatePolicy, nil,
		func(_ *rand.Rand) {
			weightMsgCreatePolicy = defaultWeightMsgCreatePolicy
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgCreatePolicy,
		acpsimulation.SimulateMsgCreatePolicy(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgCheckAccess int
	simState.AppParams.GetOrGenerate(opWeightMsgCheckAccess, &weightMsgCheckAccess, nil,
		func(_ *rand.Rand) {
			weightMsgCheckAccess = defaultWeightMsgCheckAccess
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgCheckAccess,
		acpsimulation.SimulateMsgCheckAccess(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgPolicyCmd int
	simState.AppParams.GetOrGenerate(opWeightMsgPolicyCmd, &weightMsgPolicyCmd, nil,
		func(_ *rand.Rand) {
			weightMsgPolicyCmd = defaultWeightMsgPolicyCmd
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgPolicyCmd,
		acpsimulation.SimulateMsgPolicyCmd(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgMsgEditPolicy int
	simState.AppParams.GetOrGenerate(opWeightMsgMsgEditPolicy, &weightMsgMsgEditPolicy, nil,
		func(_ *rand.Rand) {
			weightMsgMsgEditPolicy = defaultWeightMsgMsgEditPolicy
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgMsgEditPolicy,
		acpsimulation.SimulateMsgMsgEditPolicy(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	// this line is used by starport scaffolding # simapp/module/operation

	return operations
}

// ProposalMsgs returns msgs used for governance proposals for simulations.
func (am AppModule) ProposalMsgs(simState module.SimulationState) []simtypes.WeightedProposalMsg {
	return []simtypes.WeightedProposalMsg{
		simulation.NewWeightedProposalMsg(
			opWeightMsgCreatePolicy,
			defaultWeightMsgCreatePolicy,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				acpsimulation.SimulateMsgCreatePolicy(am.accountKeeper, am.bankKeeper, am.keeper)
				return nil
			},
		),
		simulation.NewWeightedProposalMsg(
			opWeightMsgCheckAccess,
			defaultWeightMsgCheckAccess,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				acpsimulation.SimulateMsgCheckAccess(am.accountKeeper, am.bankKeeper, am.keeper)
				return nil
			},
		),
		simulation.NewWeightedProposalMsg(
			opWeightMsgPolicyCmd,
			defaultWeightMsgPolicyCmd,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				acpsimulation.SimulateMsgPolicyCmd(am.accountKeeper, am.bankKeeper, am.keeper)
				return nil
			},
		),
		simulation.NewWeightedProposalMsg(
			opWeightMsgMsgEditPolicy,
			defaultWeightMsgMsgEditPolicy,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				acpsimulation.SimulateMsgMsgEditPolicy(am.accountKeeper, am.bankKeeper, am.keeper)
				return nil
			},
		),
		// this line is used by starport scaffolding # simapp/module/OpMsg
	}
}
