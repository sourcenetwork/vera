package module

import (
	"fmt"

	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	"github.com/cosmos/cosmos-sdk/version"
	"github.com/sourcenetwork/sourcehub/x/feegrant"
)

func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: feegrant.Query_serviceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "Allowance",
					Use:       "grant [granter] [grantee]",
					Short:     "Query details of a single grant",
					Long:      "Query details for a grant. You can find the fee-grant of a granter and grantee.",
					Example:   fmt.Sprintf(`$ %s query feegrant grant [granter] [grantee]`, version.AppName),
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "granter"},
						{ProtoField: "grantee"},
					},
				},
				{
					RpcMethod: "Allowances",
					Use:       "grants-by-grantee [grantee]",
					Short:     "Query all grants of a grantee",
					Long:      "Queries all the grants for a grantee address.",
					Example:   fmt.Sprintf(`$ %s query feegrant grants-by-grantee [grantee]`, version.AppName),
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "grantee"},
					},
				},
				{
					RpcMethod: "AllowancesByGranter",
					Use:       "grants-by-granter [granter]",
					Short:     "Query all grants by a granter",
					Example:   fmt.Sprintf(`$ %s query feegrant grants-by-granter [granter]`, version.AppName),
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "granter"},
					},
				},
				{
					RpcMethod: "DIDAllowance",
					Use:       "did-grant [granter] [grantee-did]",
					Short:     "Query details of a single DID grant",
					Long:      "Query details for a DID grant. You can find the fee-grant of a granter and grantee DID.",
					Example:   fmt.Sprintf(`$ %s query feegrant did-grant [granter] [grantee-did]`, version.AppName),
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "granter"},
						{ProtoField: "grantee_did"},
					},
				},
				{
					RpcMethod: "DIDAllowancesByGranter",
					Use:       "did-grants-by-granter [granter]",
					Short:     "Query all DID grants by a granter",
					Long:      "Queries all the DID grants issued by a granter address.",
					Example:   fmt.Sprintf(`$ %s query feegrant did-grants-by-granter [granter]`, version.AppName),
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "granter"},
					},
				},
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service: feegrant.Msg_serviceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "RevokeAllowance",
					Use:       "revoke [granter] [grantee]",
					Short:     "Revoke a fee grant",
					Long:      "Revoke fee grant from a granter to a grantee. Note, the '--from' flag is ignored as it is implied from [granter]",
					Example:   fmt.Sprintf(`$ %s tx feegrant revoke [granter] [grantee]`, version.AppName),
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "granter"},
						{ProtoField: "grantee"},
					},
				},
				{
					RpcMethod: "PruneAllowances",
					Use:       "prune",
					Short:     "Prune expired allowances",
					Long:      "Prune up to 75 expired allowances in order to reduce the size of the store when the number of expired allowances is large.",
					Example:   fmt.Sprintf(`$ %s tx feegrant prune --from [mykey]`, version.AppName),
				},
				{
					RpcMethod: "ExpireDIDAllowance",
					Use:       "expire-did [granter] [grantee-did]",
					Short:     "Expire a DID fee grant",
					Long:      "Expire fee grant from a granter to a grantee DID. Note, the '--from' flag is ignored as it is implied from [granter]",
					Example:   fmt.Sprintf(`$ %s tx feegrant expire-did [granter] [grantee-did]`, version.AppName),
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "granter"},
						{ProtoField: "grantee_did"},
					},
				},
				{
					RpcMethod: "PruneDIDAllowances",
					Use:       "prune-did",
					Short:     "Prune expired DID allowances",
					Long:      "Prune expired DID allowances in order to reduce the size of the store when the number of expired DID allowances is large.",
					Example:   fmt.Sprintf(`$ %s tx feegrant prune-did --from [mykey]`, version.AppName),
				},
			},
			EnhanceCustomCommand: true,
		},
	}
}
