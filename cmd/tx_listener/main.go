package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/sourcenetwork/sourcehub/sdk"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "tx-listener [comet-rpc-addr]",
	Short: "listens to proposed txs and unmarshal results into structured SourceHub msgs",
	Long: `tx-listener is a cli utility which connects to SourceHub's cometbft rpc connection
	and listens for Tx processing events.
	The received events are expanded and the Tx results are unmarshaled into the correct
	Msg response types.

	This is meant to be used a development tool to monitor the result of executed Txs by SourceHub.
	`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var opts []sdk.Opt
		if len(args) == 1 {
			opts = append(opts, sdk.WithCometRPCAddr(args[0]))
		}
		client, err := sdk.NewClient(opts...)
		if err != nil {
			log.Fatal(err)
		}

		listener := client.TxListener()

		ctx := context.Background()

		ch, errCh, err := listener.ListenTxs(ctx)
		defer listener.Close()
		if err != nil {
			log.Fatal(err)
		}

		for {
			select {
			case result := <-ch:
				bytes, err := json.MarshalIndent(result, "", "  ")
				if err != nil {
					log.Fatalf("failed to marshal result: %v", err)
				}
				log.Print(string(bytes))
			case err := <-errCh:
				log.Printf("ERROR in Tx: %v", err)
			case <-listener.Done():
				log.Printf("Client terminated")
				return
			case <-ctx.Done():
				log.Printf("Ctx terminated")
				return
			}
		}
	},
}

func main() {
	rootCmd.Execute()
}
