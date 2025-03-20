package cli

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"

	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

func CmdQueryOjectOwner() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "object-owner [policy-id] [resource] [object-id]",
		Short: "Queries an object for its owner",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			polId := args[0]
			resource := args[1]
			objId := args[2]

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			req := types.QueryObjectOwnerRequest{
				PolicyId: polId,
				Object:   coretypes.NewObject(resource, objId),
			}

			res, err := queryClient.ObjectOwner(cmd.Context(), &req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
