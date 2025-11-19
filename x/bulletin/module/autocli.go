package bulletin

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	modulev1 "github.com/sourcenetwork/sourcehub/api/sourcehub/bulletin"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: modulev1.Query_ServiceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "Params",
					Use:       "params",
					Short:     "Shows the parameters of the module",
				},
				{
					RpcMethod:      "Namespaces",
					Use:            "namespaces",
					Short:          "Query all namespaces",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{},
				},
				{
					RpcMethod:      "Namespace",
					Use:            "namespace [namespace]",
					Short:          "Query namespace by name",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "namespace"}},
				},
				{
					RpcMethod:      "NamespaceCollaborators",
					Use:            "namespace-collaborators [namespace]",
					Short:          "Query all namespace collaborators",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "namespace"}},
				},
				{
					RpcMethod:      "NamespacePosts",
					Use:            "namespace-posts [namespace]",
					Short:          "Query all posts within the namespace",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "namespace"}},
				},
				{
					RpcMethod:      "Post",
					Use:            "post [namespace] [id]",
					Short:          "Query post by namespace and id",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "namespace"}, {ProtoField: "id"}},
				},
				{
					RpcMethod:      "Posts",
					Use:            "posts",
					Short:          "Query all posts",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{},
				},
				// this line is used by ignite scaffolding # autocli/query
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service:              modulev1.Msg_ServiceDesc.ServiceName,
			EnhanceCustomCommand: true, // only required if you want to use the custom command
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "UpdateParams",
					Skip:      true, // skipped because authority gated
				},
				{
					RpcMethod:      "CreatePost",
					Use:            "create-post [namespace] [payload] [proof]",
					Short:          "Add a new post to the specified namespace",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "namespace"}, {ProtoField: "payload"}, {ProtoField: "proof"}},
				},
				{
					RpcMethod:      "RegisterNamespace",
					Use:            "register-namespace [namespace]",
					Short:          "Register a new namespace",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "namespace"}},
				},
				{
					RpcMethod:      "AddCollaborator",
					Use:            "add-collaborator [namespace] [collaborator]",
					Short:          "Add a new collaborator to the specified namespace",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "namespace"}, {ProtoField: "collaborator"}},
				},
				{
					RpcMethod:      "RemoveCollaborator",
					Use:            "remove-collaborator [namespace] [collaborator]",
					Short:          "Remove existing collaborator from the specified namespace",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "namespace"}, {ProtoField: "collaborator"}},
				},
				// this line is used by ignite scaffolding # autocli/tx
			},
		},
	}
}
