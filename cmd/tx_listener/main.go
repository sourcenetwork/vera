package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

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
		addr := sdk.DefaultCometRPCAddr
		var opts []sdk.Opt
		if len(args) == 1 {
			addr = args[0]
		}
		opts = append(opts, sdk.WithCometRPCAddr(addr))
		client, err := sdk.NewClient(opts...)
		if err != nil {
			log.Fatal(err)
		}

		listener := client.TxListener()

		ctx := context.Background()

		listener.ListenAsync(ctx, func(ev *sdk.Event, err error) {
			if err != nil {
				log.Printf("ERROR in Tx: %v", err)
			} else {
				bytes, err := json.MarshalIndent(ev, "", "  ")
				if err != nil {
					log.Fatalf("failed to marshal result: %v", err)
				}
				log.Print(string(bytes))
			}
		})
		log.Printf("Listening to RPC at %v", addr)

		done := make(chan os.Signal, 1)
		signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)
		fmt.Println("Blocking, press ctrl+c to continue...")

		<-done
		fmt.Println("Received interrupt: terminating listener")
		listener.Close()
	},
}

func main() {
	rootCmd.Execute()
}
