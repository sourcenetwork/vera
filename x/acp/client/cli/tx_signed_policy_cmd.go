package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"

	"github.com/sourcenetwork/vera/x/acp/types"
)

var _ = strconv.Itoa(0)

func CmdSignedPolicyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "signed-policy-cmd jws-payload",
		Short: "Broadcast a SignedPolicyCmd msg from a JWS",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			creator := clientCtx.GetFromAddress().String()
			msg := types.NewMsgSignedPolicyCmdFromJWS(creator, args[0])
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
