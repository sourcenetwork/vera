package cli

import (
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/spf13/cobra"

	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

var _ = strconv.Itoa(0)

func CmdEditPolicy() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit-policy policy_id {policy_file | -}",
		Short: "Broadcast message EditPolicy",
		Long: `
                       Broadcast message EditPolicy.

					   policy_id is the id of the policy that is going to be edited.

                       policy_file specifies a file whose contents is the policy.
                       - to read from stdin.
                       Note: if reading from stdin make sure flag --yes is set.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			policyId := args[0]
			policyFile := args[1]

			var file io.Reader
			if policyFile != "-" {
				sysFile, err := os.Open(policyFile)
				if err != nil {
					return fmt.Errorf("could not open policy file: %w", err)
				}
				defer sysFile.Close()
				file = sysFile
			} else {
				file = os.Stdin
			}

			policy, err := io.ReadAll(file)
			if err != nil {
				return fmt.Errorf("could not read policy file: %w", err)
			}

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgEditPolicy{
				Creator:     clientCtx.GetFromAddress().String(),
				Policy:      string(policy),
				PolicyId:    policyId,
				MarshalType: coretypes.PolicyMarshalingType_SHORT_YAML,
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
