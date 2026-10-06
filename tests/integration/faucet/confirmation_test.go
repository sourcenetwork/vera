package faucet

import (
	"context"
	"testing"
	"time"

	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func requireCommittedFaucetTx(t *testing.T, conn grpc.ClientConnInterface, txHash string) {
	t.Helper()
	// The faucet returns after CheckTx; only this transaction's committed result establishes success.
	txClient := txtypes.NewServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	var committed *txtypes.GetTxResponse
	for {
		var err error
		committed, err = txClient.GetTx(ctx, &txtypes.GetTxRequest{Hash: txHash})
		if status.Code(err) != codes.NotFound {
			require.NoError(t, err, "query grant transaction %s", txHash)
			break
		}
		select {
		case <-ctx.Done():
			t.Fatalf("grant transaction %s was not committed: %v", txHash, ctx.Err())
		case <-ticker.C:
		}
	}
	require.NotNil(t, committed)
	require.NotNil(t, committed.TxResponse)
	require.Positive(t, committed.TxResponse.Height)
	require.Equal(t, uint32(0), committed.TxResponse.Code, "grant transaction failed in block: %s", committed.TxResponse.RawLog)
}
