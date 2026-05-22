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
					Use:            "document [namespace] [id]",
					Short:          "Query document by namespace and id",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "namespace"}, {ProtoField: "id"}},
				},
				{
					RpcMethod:      "Documents",
					Use:            "documents [namespace]",
					Short:          "Query documents within a namespace",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "namespace"}},
				},
				{
					RpcMethod:      "KeyDerivation",
					Use:            "key-derivation [namespace] [id]",
					Short:          "Query key derivation by namespace and id",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "namespace"}, {ProtoField: "id"}},
				},
				{
					RpcMethod:      "KeyDerivations",
					Use:            "key-derivations [namespace]",
					Short:          "Query key derivations within a namespace",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "namespace"}},
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
					Use:            "create-ring [namespace] [ring_pk] [threshold]",
					Short:          "Create an Orbis ring",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "namespace"}, {ProtoField: "ring_pk"}, {ProtoField: "threshold"}},
				},
				{
					RpcMethod:      "UpdateRingByAcp",
					Use:            "update-ring-by-acp [ring_id]",
					Short:          "Update pending ring reshare fields via external ACP policy",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "ring_id"}},
				},
				{
					RpcMethod:      "FinalizeRingReshareByThresholdSignature",
					Use:            "finalize-ring-reshare [ring_id] [signature_scheme] [signature]",
					Short:          "Finalize a ring reshare using a threshold signature",
					Long:           "Finalize a ring reshare using a threshold signature. The signature argument is a bytes field and must be base64-encoded.",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "ring_id"}, {ProtoField: "signature_scheme"}, {ProtoField: "signature"}},
				},
				{
					RpcMethod:      "StoreDocument",
					Use:            "store-document [namespace] [ring_id] [document] [proof] [policy_id] [resource] [permission]",
					Short:          "Store an encrypted Orbis document",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "namespace"}, {ProtoField: "ring_id"}, {ProtoField: "document"}, {ProtoField: "proof"}, {ProtoField: "policy_id"}, {ProtoField: "resource"}, {ProtoField: "permission"}},
				},
				{
					RpcMethod:      "StoreKeyDerivation",
					Use:            "store-key-derivation [namespace] [ring_id] [derivation] [policy_id] [resource] [permission]",
					Short:          "Store an Orbis key derivation",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "namespace"}, {ProtoField: "ring_id"}, {ProtoField: "derivation"}, {ProtoField: "policy_id"}, {ProtoField: "resource"}, {ProtoField: "permission"}},
				},
				{
					RpcMethod:      "CreateNodeInfo",
					Use:            "create-node-info [peer_id] [controller_key]",
					Short:          "Create a node info record (store key derived from signing key)",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "peer_id"}, {ProtoField: "controller_key"}},
				},
				{
					RpcMethod:      "UpdateNodeInfo",
					Use:            "update-node-info [node_key]",
					Short:          "Update a node info record (signer must match stored controller key)",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "node_key"}},
				},
			},
		},
	}
}
