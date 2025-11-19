package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"

	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

var _ = strconv.Itoa(0)

const BearerFlag = "bearer-token"

func bearerDispatcher(cmd *cobra.Command, polId string, polCmd *types.PolicyCmd) error {
	clientCtx, err := client.GetClientTxContext(cmd)
	if err != nil {
		return err
	}

	flag := cmd.Flag(BearerFlag)
	token := flag.Value.String()

	creator := clientCtx.GetFromAddress().String()
	msg := types.NewMsgBearerPolicyCmd(creator, token, polId, polCmd)
	return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
}

func CmdBearerPolicyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bearer-policy-cmd",
		Short: "Broadcast a BearerPolicyCommand msg",
	}

	flags := cmd.PersistentFlags()
	flags.String(BearerFlag, "", "specifies the bearer token to be broadcast with the command")

	cmd.AddCommand(CmdRegisterObject(bearerDispatcher))
	cmd.AddCommand(CmdArchiveObject(bearerDispatcher))
	cmd.AddCommand(CmdSetRelationship(bearerDispatcher))
	cmd.AddCommand(CmdDeleteRelationship(bearerDispatcher))
	cmd.AddCommand(CmdCreateCommitment(bearerDispatcher))
	cmd.AddCommand(CmdRevealRegistration(bearerDispatcher))
	cmd.AddCommand(CmdFlagHijack(bearerDispatcher))
	cmd.AddCommand(CmdUnarchiveObject(bearerDispatcher))
	return cmd
}
