package orbis

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	modulev1 "github.com/sourcenetwork/sourcehub/api/sourcehub/orbis"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: modulev1.Query_ServiceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{RpcMethod: "Params", Use: "params", Short: "Shows the parameters of the module"},
				{
					RpcMethod:      "Ring",
					Use:            "ring [id]",
					Short:          "Query ring by id",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "id"}},
				},
				{RpcMethod: "Rings", Use: "rings", Short: "Query rings"},
				{
					RpcMethod:      "Document",
					Use:            "document [id]",
					Short:          "Query document by id",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "id"}},
				},
				{
					RpcMethod: "Documents",
					Use:       "documents",
					Short:     "Query documents",
				},
				{
					RpcMethod:      "KeyDerivation",
					Use:            "key-derivation [id]",
					Short:          "Query key derivation by id",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "id"}},
				},
				{
					RpcMethod: "KeyDerivations",
					Use:       "key-derivations",
					Short:     "Query key derivations",
				},
				{
					RpcMethod:      "NodeInfo",
					Use:            "node-info [node_key]",
					Short:          "Query node info by node key",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "node_key"}},
				},
				{
					RpcMethod:      "NodeDemerits",
					Use:            "node-demerits [ring_id] [node_key]",
					Short:          "Query a node's demerit score in a ring",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "ring_id"}, {ProtoField: "node_key"}},
				},
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service:              modulev1.Msg_ServiceDesc.ServiceName,
			EnhanceCustomCommand: true,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{RpcMethod: "UpdateParams", Skip: true},
				{
					RpcMethod:      "CreateRing",
					Use:            "create-ring [threshold] [pss_interval] [policy_id] [current_version]",
					Short:          "Create an Orbis ring",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "threshold"}, {ProtoField: "pss_interval"}, {ProtoField: "policy_id"}, {ProtoField: "current_version"}},
				},
				{
					RpcMethod:      "CancelPendingRing",
					Use:            "cancel-pending-ring [ring_id]",
					Short:          "Cancel an unfinished Orbis DKG",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "ring_id"}},
				},
				{
					RpcMethod:      "StartRingReshareByAcp",
					Use:            "start-ring-reshare-by-acp [ring_id]",
					Short:          "Start a ring reshare via external ACP policy",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "ring_id"}},
				},
				{
					RpcMethod:      "SetRingPssIntervalByAcp",
					Use:            "set-ring-pss-interval-by-acp [ring_id] [pss_interval]",
					Short:          "Set a ring PSS interval via external ACP policy",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "ring_id"}, {ProtoField: "pss_interval"}},
				},
				{
					RpcMethod:      "ScheduleRingUpgradeByAcp",
					Use:            "schedule-ring-upgrade-by-acp [ring_id] [next_version] [activation_time]",
					Short:          "Schedule a ring protocol upgrade via external ACP policy",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "ring_id"}, {ProtoField: "next_version"}, {ProtoField: "activation_time"}},
				},
				{
					RpcMethod:      "CancelRingUpgradeByAcp",
					Use:            "cancel-ring-upgrade-by-acp [ring_id]",
					Short:          "Cancel a pending ring protocol upgrade via external ACP policy",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "ring_id"}},
				},
				{
					RpcMethod:      "SetRingReportingByAcp",
					Use:            "set-ring-reporting-by-acp [ring_id]",
					Short:          "Set ring reporting policy via external ACP policy",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "ring_id"}},
				},
				{
					RpcMethod:      "AddRingTrustedAuthRelayByAcp",
					Use:            "add-ring-trusted-auth-relay-by-acp [ring_id] [relay_did]",
					Short:          "Add a ring authentication relay via external ACP policy",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "ring_id"}, {ProtoField: "relay_did"}},
				},
				{
					RpcMethod:      "RemoveRingTrustedAuthRelayByAcp",
					Use:            "remove-ring-trusted-auth-relay-by-acp [ring_id] [relay_did]",
					Short:          "Revoke a ring authentication relay via external ACP policy",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "ring_id"}, {ProtoField: "relay_did"}},
				},
				{
					RpcMethod:      "FinalizeRingReshareByThresholdSignature",
					Use:            "finalize-ring-reshare [ring_id] [signature_scheme] [signature]",
					Short:          "Finalize a ring reshare using a threshold signature",
					Long:           "Finalize a ring reshare using a threshold signature. The signature argument is a bytes field and must be base64-encoded.",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "ring_id"}, {ProtoField: "signature_scheme"}, {ProtoField: "signature"}},
				},
				{
					RpcMethod: "SubmitReport",
					Use:       "submit-report",
					Short:     "Submit an MPC fault report",
					Long:      "Submit an MPC fault report. The report field is a message value and the signature field must be base64-encoded.",
				},
				{
					RpcMethod:      "StoreDocument",
					Use:            "store-document [ring_id] [document] [proof] [policy_id] [resource] [permission]",
					Short:          "Store an encrypted Orbis document",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "ring_id"}, {ProtoField: "document"}, {ProtoField: "proof"}, {ProtoField: "policy_id"}, {ProtoField: "resource"}, {ProtoField: "permission"}},
				},
				{
					RpcMethod:      "StoreKeyDerivation",
					Use:            "store-key-derivation [ring_id] [derivation] [policy_id] [resource] [permission]",
					Short:          "Store an Orbis key derivation",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "ring_id"}, {ProtoField: "derivation"}, {ProtoField: "policy_id"}, {ProtoField: "resource"}, {ProtoField: "permission"}},
				},
				{
					RpcMethod:      "CreateNodeInfo",
					Use:            "create-node-info [peer_id] [controller_key]",
					Short:          "Create a node info record (store key derived from signing key)",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "peer_id"}, {ProtoField: "controller_key"}},
				},
				{
					RpcMethod:      "UpdateNodePeerId",
					Use:            "update-node-peer-id [node_key] [peer_id]",
					Short:          "Update a node peer ID",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "node_key"}, {ProtoField: "peer_id"}},
				},
				{
					RpcMethod:      "TransferNodeController",
					Use:            "transfer-node-controller [node_key] [controller_key]",
					Short:          "Transfer control of a node info record",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "node_key"}, {ProtoField: "controller_key"}},
				},
				{
					RpcMethod:      "AddNodeToWhitelist",
					Use:            "add-node-to-whitelist [node_key]",
					Short:          "Add one policy or ring to a node whitelist",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "node_key"}},
				},
				{
					RpcMethod:      "RemoveNodeFromWhitelist",
					Use:            "remove-node-from-whitelist [node_key]",
					Short:          "Remove one policy or ring from a node whitelist",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "node_key"}},
				},
				{
					RpcMethod:      "DrainNodeKey",
					Use:            "drain-node-key [node_key]",
					Short:          "Drain a node key's account balance to the controller key",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "node_key"}},
				},
			},
		},
	}
}
