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

func CmdCreatePolicy() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-policy [marshaling_type] {policy_file | -}",
		Short: "Broadcast message CreatePolicy",
		Long: `
                       Broadcast message CreatePolicy.

                       marshaling_type specifies the marshaling format for the policy.
                       Defaults to SHORT_YAML.
                       See PolicyMarshalingType for accepted values.

                       policy_file specifies a file whose contents is the policy.
                       - to read from stdin.
                       Note: if reading from stdin make sure flag --yes is set.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			var policyFile string
			var marshalingType coretypes.PolicyMarshalingType

			if len(args) == 2 {
				marshalingType = coretypes.PolicyMarshalingType_UNKNOWN
				policyFile = args[1]

				marshalingTypeCode, ok := coretypes.PolicyMarshalingType_value[args[0]]
				if ok {
					marshalingType = coretypes.PolicyMarshalingType(marshalingTypeCode)
				}
			} else {
				policyFile = args[0]
				marshalingType = coretypes.PolicyMarshalingType_SHORT_YAML
			}

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

			msg := &types.MsgCreatePolicy{
				Creator:     clientCtx.GetFromAddress().String(),
				Policy:      string(policy),
				MarshalType: marshalingType,
			}
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
