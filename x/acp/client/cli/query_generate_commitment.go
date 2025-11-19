package cli

import (
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/spf13/cobra"

	acptypes "github.com/sourcenetwork/sourcehub/x/acp/types"
)

func CmdQueryGenerateCommitment() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate-commitment policyId actorId [resource:object]+",
		Short: "Generates a hex encoded commitment for the given objects",
		Long:  ` `,
		Args:  cobra.MinimumNArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			policyId := args[0]
			actorId := args[1]
			var objs []*coretypes.Object
			for _, operationStr := range args[2:] {
				resource, object, _ := strings.Cut(operationStr, ":")
				obj := coretypes.NewObject(resource, object)
				objs = append(objs, obj)
			}
			queryClient := acptypes.NewQueryClient(clientCtx)
			req := acptypes.QueryGenerateCommitmentRequest{
				PolicyId: policyId,
				Objects:  objs,
				Actor:    coretypes.NewActor(actorId),
			}
			resp, err := queryClient.GenerateCommitment(cmd.Context(), &req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
