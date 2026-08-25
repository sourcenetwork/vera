package tier

import (
	"math/rand"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"

	"github.com/sourcenetwork/vera/testutil/sample"
	tiersimulation "github.com/sourcenetwork/vera/x/tier/simulation"
	"github.com/sourcenetwork/vera/x/tier/types"
)

// avoid unused import issue
var (
	_ = tiersimulation.FindAccount
	_ = rand.Rand{}
	_ = sample.RandomAccAddress().String()
	_ = sdk.AccAddress{}
	_ = simulation.MsgEntryKind
)

const (
	opWeightMsgLockStake = "op_weight_msg_lock_stake"
	// TODO: Determine the simulation weight value
	defaultWeightMsgLockStake int = 100

	opWeightMsgUnlockStake = "op_weight_msg_unlock_stake"
	// TODO: Determine the simulation weight value
	defaultWeightMsgUnlockStake int = 100

	opWeightMsgRedelegateStake = "op_weight_msg_redelegate_stake"
	// TODO: Determine the simulation weight value
	defaultWeightMsgRedelegateStake int = 100

	opWeightMsgCancelUnlockingStake = "op_weight_msg_cancel_unlocking_stake"
	// TODO: Determine the simulation weight value
	defaultWeightMsgCancelUnlockingStake int = 100

	// this line is used by starport scaffolding # simapp/module/const
)

// GenerateGenesisState creates a randomized GenState of the module.
func (AppModule) GenerateGenesisState(simState *module.SimulationState) {
	accs := make([]string, len(simState.Accounts))
	for i, acc := range simState.Accounts {
		accs[i] = acc.Address.String()
	}
	tierGenesis := types.GenesisState{
		Params: types.DefaultParams(),
		// this line is used by starport scaffolding # simapp/module/genesisState
	}
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(&tierGenesis)
}

// RegisterStoreDecoder registers a decoder.
func (am AppModule) RegisterStoreDecoder(_ simtypes.StoreDecoderRegistry) {}

// WeightedOperations returns the all the gov module operations with their respective weights.
func (am AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
	operations := make([]simtypes.WeightedOperation, 0)

	var weightMsgLockStake int
	simState.AppParams.GetOrGenerate(opWeightMsgLockStake, &weightMsgLockStake, nil,
		func(_ *rand.Rand) {
			weightMsgLockStake = defaultWeightMsgLockStake
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgLockStake,
		tiersimulation.SimulateMsgLockStake(am.bankKeeper, am.keeper),
	))

	var weightMsgUnlockStake int
	simState.AppParams.GetOrGenerate(opWeightMsgUnlockStake, &weightMsgUnlockStake, nil,
		func(_ *rand.Rand) {
			weightMsgUnlockStake = defaultWeightMsgUnlockStake
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgUnlockStake,
		tiersimulation.SimulateMsgUnlockStake(am.bankKeeper, am.keeper),
	))

	var weightMsgRedelegateStake int
	simState.AppParams.GetOrGenerate(opWeightMsgRedelegateStake, &weightMsgRedelegateStake, nil,
		func(_ *rand.Rand) {
			weightMsgRedelegateStake = defaultWeightMsgRedelegateStake
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgRedelegateStake,
		tiersimulation.SimulateMsgRedelegateStake(am.bankKeeper, am.keeper),
	))

	var weightMsgCancelUnlockingStake int
	simState.AppParams.GetOrGenerate(opWeightMsgCancelUnlockingStake, &weightMsgCancelUnlockingStake, nil,
		func(_ *rand.Rand) {
			weightMsgCancelUnlockingStake = defaultWeightMsgCancelUnlockingStake
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgCancelUnlockingStake,
		tiersimulation.SimulateMsgCancelUnlockingStake(am.bankKeeper, am.keeper),
	))

	// this line is used by starport scaffolding # simapp/module/operation

	return operations
}

// ProposalMsgs returns msgs used for governance proposals for simulations.
func (am AppModule) ProposalMsgs(simState module.SimulationState) []simtypes.WeightedProposalMsg {
	return []simtypes.WeightedProposalMsg{
		simulation.NewWeightedProposalMsg(
			opWeightMsgLockStake,
			defaultWeightMsgLockStake,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				tiersimulation.SimulateMsgLockStake(am.bankKeeper, am.keeper)
				return nil
			},
		),
		simulation.NewWeightedProposalMsg(
			opWeightMsgUnlockStake,
			defaultWeightMsgUnlockStake,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				tiersimulation.SimulateMsgUnlockStake(am.bankKeeper, am.keeper)
				return nil
			},
		),
		simulation.NewWeightedProposalMsg(
			opWeightMsgRedelegateStake,
			defaultWeightMsgRedelegateStake,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				tiersimulation.SimulateMsgRedelegateStake(am.bankKeeper, am.keeper)
				return nil
			},
		),
		simulation.NewWeightedProposalMsg(
			opWeightMsgCancelUnlockingStake,
			defaultWeightMsgCancelUnlockingStake,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				tiersimulation.SimulateMsgCancelUnlockingStake(am.bankKeeper, am.keeper)
				return nil
			},
		),
		// this line is used by starport scaffolding # simapp/module/OpMsg
	}
}
