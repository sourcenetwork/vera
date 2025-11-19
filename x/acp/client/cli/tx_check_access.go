package cli

import (
	"strconv"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/spf13/cobra"

	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

var _ = strconv.Itoa(0)

func CmdCheckAccess() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check-access [policy-id] [subject] {resource:objId#permission}",
		Short: "Broadcast message CheckAccess",
		Args:  cobra.MinimumNArgs(3),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			policyId := args[0]
			subject := args[1]

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

			accessRequest := &coretypes.AccessRequest{
				Operations: operations,
				Actor: &coretypes.Actor{
					Id: subject,
				},
			}

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgCheckAccess{
				Creator:       clientCtx.GetFromAddress().String(),
				PolicyId:      policyId,
				AccessRequest: accessRequest,
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
