package signed_policy_cmd

import (
	"context"
	"fmt"

	"github.com/cometbft/cometbft/rpc/client"
)

// LogicalClock models a Provider for logical timestamps.
// Timestamps must be monotonically increasing and are used as reference
// for the total ordering of events in the system.
//
// This abstraction is general purpose but for the current context of Vera
// this primarily means the current system block height.
type LogicalClock interface {

	// GetTimestamp returns an integer for the current timestamp in the system.
	GetTimestampNow(ctx context.Context) (uint64, error)
}

var _ LogicalClock = (*abciLogicalClock)(nil)

func LogicalClockFromCometClient(client client.Client) LogicalClock {
	return &abciLogicalClock{
		rpcClient: client,
	}
}

type abciLogicalClock struct {
	rpcClient client.Client
}

func (c *abciLogicalClock) GetTimestampNow(ctx context.Context) (uint64, error) {
	resp, err := c.rpcClient.ABCIInfo(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch latest block: %v", err)
	}

	return uint64(resp.Response.LastBlockHeight), nil
}
