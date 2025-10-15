package cmd

import (
	"log"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
)

// NewTestnetCmd creates a root testnet command with subcommands to run an in-process testnet or initialize
// validator configuration files for running a multi-validator testnet in a separate process.
func NewProbeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "probe",
		Short:                      "probe is a liveness probe which asserts that the chain is not stuck at the 0 block",
		SuggestionsMinimumDistance: 2,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			resp, err := clientCtx.Client.Status(cmd.Context())
			if err != nil {
				return err
			}
			height := resp.SyncInfo.LatestBlockHeight
			log.Printf("Latest node height: %v", height)
			if height == 0 {
				log.Fatalf("Node liveness check failed: latest height %v", height)
			}

			return nil
		},
	}
	return cmd
}
