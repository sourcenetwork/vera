package lanes

import (
	"context"
	"fmt"
	"strings"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/skip-mev/block-sdk/v2/block/base"
	appparams "github.com/sourcenetwork/sourcehub/app/params"
)

const (
	// LaneName defines the name of the priority lane.
	LaneName = "priority"
)

// NewPriorityLane returns a new priority lane.
func NewPriorityLane[C comparable](
	cfg base.LaneConfig,
	txPriority base.TxPriority[C],
	matchFn base.MatchHandler,
) *base.BaseLane {
	options := []base.LaneOption{
		base.WithMatchHandler(matchFn),
		base.WithMempoolConfigs[C](cfg, txPriority),
	}

	lane, err := base.NewBaseLane(
		cfg,
		LaneName,
		options...,
	)
	if err != nil {
		panic(err)
	}

	return lane
}

// TxPriority defines a transaction prioritization strategy for the priority lane.
// Transactions are ranked by priority group and gas price (higher is better).
func TxPriority() base.TxPriority[string] {
	return base.TxPriority[string]{
		GetTxPriority: func(_ context.Context, tx sdk.Tx) string {
			priorityGroup := getPriorityGroup(tx)
			gasPrice := getGasPrice(tx)
			gasPriceStr := formatGasPrice(gasPrice)
			return priorityGroup + ":" + gasPriceStr
		},
		Compare: func(a, b string) int {
			if a > b {
				return 1
			} else if a < b {
				return -1
			}
			return 0
		},
		MinValue: "0:00000000000000000000000000000000",
	}
}

// getPriorityGroup returns a string that defines the transaction priority.
// Prioritizes transactions based on the modules their messages belong to.
func getPriorityGroup(tx sdk.Tx) string {
	msgs := tx.GetMsgs()
	if len(msgs) == 0 {
		return "0"
	}
	// Start with highest possible priority and reduce txPriority based on the found messages,
	// so that the system can not be abused by sending a mix or high and low priority messages
	minPriority := "3"
	for _, msg := range msgs {
		msgType := sdk.MsgTypeURL(msg)
		switch {
		case strings.HasPrefix(msgType, "/sourcehub.acp."):
			// Keep minPriority at 3 for acp module messages
		case strings.HasPrefix(msgType, "/sourcehub.tier."):
			// Reduce minPriority to 2 if tier module message found
			if minPriority > "2" {
				minPriority = "2"
			}
		case strings.HasPrefix(msgType, "/sourcehub.bulletin."):
			// Reduce minPriority to 1 if bulletin module message found
			if minPriority > "1" {
				minPriority = "1"
			}
		default:
			// Return lowest priority if found a message from other modules
			return "0"
		}
	}
	// Return the lowest priority based on the messages found in the tx
	return minPriority
}

// getGasPrice extracts the gas price from the transaction.
func getGasPrice(tx sdk.Tx) math.LegacyDec {
	// Cast tx to FeeTx, return LegacyZeroDec if fails
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return math.LegacyZeroDec()
	}
	// Get the fee and gas limit, return LegacyZeroDec if invalid
	fee := feeTx.GetFee()
	gasLimit := feeTx.GetGas()
	if gasLimit == 0 || len(fee) == 0 {
		return math.LegacyZeroDec()
	}
	// Calculate and return the gas price (e.g, total fee / gas limit)
	return math.LegacyNewDecFromInt(fee.AmountOf(appparams.DefaultBondDenom)).Quo(math.LegacyNewDec(int64(gasLimit)))
}

// formatGasPrice ensures lexicographic sorting of gas prices.
func formatGasPrice(gasPrice math.LegacyDec) string {
	// Convert to string and remove the decimal point
	gasPriceStr := strings.ReplaceAll(gasPrice.String(), ".", "")
	// Ensure gas price does not exceed 32 characters
	if len(gasPriceStr) > 32 {
		gasPriceStr = gasPriceStr[:32]
	}
	// Ensure proper zero-padding to 32 characters
	return fmt.Sprintf("%032s", gasPriceStr)
}
