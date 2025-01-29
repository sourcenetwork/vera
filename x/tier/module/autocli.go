package tier

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	modulev1beta1 "github.com/sourcenetwork/sourcehub/api/sourcehub/tier/v1beta1"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: modulev1beta1.Query_ServiceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "Params",
					Use:       "params",
					Short:     "Shows the parameters of the module",
				},
				{
					RpcMethod:      "Lockup",
					Use:            "lockup [delegator-address] [validator-address]",
					Short:          "Query a locked stake based on address and validator address",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "delegator_address"}, {ProtoField: "validator_address"}},
				},
				{
					RpcMethod:      "Lockups",
					Use:            "lockups [delegator-address]",
					Short:          "Query all locked stakes made by the delegator",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "delegator_address"}},
				},
				// {
				// 	RpcMethod:      "LockupsTo",
				// 	Use:            "lockups-to [validator-address]",
				// 	Short:          "Query all lockups made to one validator",
				// 	PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "validatorAddress"}},
				// },
				{
					RpcMethod:      "UnlockingLockup",
					Use:            "unlocking-lockup [delegator-address] [validator-address] [creation-height]",
					Short:          "Query an unlocking stake based on address and validator addres",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "delegator_address"}, {ProtoField: "validator_address"}, {ProtoField: "creation_height"}},
				},
				{
					RpcMethod:      "UnlockingLockups",
					Use:            "unlocking-lockups [delegator-address]",
					Short:          "Query all unlocking stakes made by the delegator",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "delegator_address"}},
				},
				// {
				// 	RpcMethod:      "UnlockingLockupsFrom",
				// 	Use:            "unlocking-lockups-from [validator-address]",
				// 	Short:          "Query all unlocking-lockups made from a validator",
				// 	PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "delegator_address"}},
				// },
				// this line is used by ignite scaffolding # autocli/query
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service:              modulev1beta1.Msg_ServiceDesc.ServiceName,
			EnhanceCustomCommand: true, // only required if you want to use the custom command
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "UpdateParams",
					Skip:      true, // skipped because authority gated
				},
				{
					RpcMethod:      "Lock",
					Use:            "lock [validator-address] [stake]",
					Short:          "Send a lock tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "validator_address"}, {ProtoField: "stake"}},
				},
				{
					RpcMethod:      "Unlock",
					Use:            "unlock [validator-address] [amount]",
					Short:          "Send a unlock tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "validator_address"}, {ProtoField: "stake"}},
				},
				{
					RpcMethod:      "Redelegate",
					Use:            "redelegate [src-validator-address] [dst-validator-address] [stake]",
					Short:          "Send a redelegate tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "src_validator_address"}, {ProtoField: "dst_validator_address"}, {ProtoField: "stake"}},
				},
				{
					RpcMethod:      "CancelUnlocking",
					Use:            "cancel-unlocking [validator-address] [stake] [creation-height]",
					Short:          "Send a cancel-unlocking tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "validator_address"}, {ProtoField: "stake"}, {ProtoField: "creation_height"}},
				},
				// this line is used by ignite scaffolding # autocli/tx
			},
		},
	}
}
