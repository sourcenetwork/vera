package cli

import (
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/spf13/cobra"

	acptypes "github.com/sourcenetwork/sourcehub/x/acp/types"
)

func CmdQueryVerifyAccessRequest() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify-access-request [policyId] [actor] {resource:object#relation}",
		Short: "Verifies an access request against a policy and its relationships",
		Long: `
		Builds an AccessRequest for from policyId, actor and the set of Operations
		(ie. object, relation pairs).
		The AccessRequest is evaluated and returns true iff all Operations were authorized
		by the authorization engine.
		`,
		Args: cobra.MinimumNArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			policyId := args[0]
			actorId := args[1]
			var operations []*coretypes.Operation
			for _, operationStr := range args[2:] {
				resource, operationStr, _ := strings.Cut(operationStr, ":")
				objId, relation, _ := strings.Cut(operationStr, "#")
				operation := &coretypes.Operation{
					Object:     coretypes.NewObject(resource, objId),
					Permission: relation,
				}
				operations = append(operations, operation)
			}
			queryClient := acptypes.NewQueryClient(clientCtx)
			req := acptypes.QueryVerifyAccessRequestRequest{
				PolicyId: policyId,
				AccessRequest: &coretypes.AccessRequest{
					Operations: operations,
					Actor:      &coretypes.Actor{Id: actorId},
				},
			}
			resp, err := queryClient.VerifyAccessRequest(cmd.Context(), &req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
