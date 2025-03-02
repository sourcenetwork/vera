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
// Prioritizes the x/acp module transactions over the x/tier module transactions.
func getPriorityGroup(tx sdk.Tx) string {
	// Check message types of all messages in the transaction
	for _, msg := range tx.GetMsgs() {
		msgType := sdk.MsgTypeURL(msg)
		// Set higher priority for messages that belong to the x/acp module
		if strings.HasPrefix(msgType, "/sourcehub.acp.") {
			return "2"
		}
	}
	// Set lower priority for messages that belong to the x/tier module
	return "1"
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

// PriorityMatchHandler returns the default match handler for the priority lane.
// The default implementation matches transactions related to the x/acp or x/tier modules.
func PriorityMatchHandler() base.MatchHandler {
	return func(_ sdk.Context, tx sdk.Tx) bool {
		msgs := tx.GetMsgs()
		// Return false if there are no messages
		if len(msgs) == 0 {
			return false
		}
		// Check message types of all messages in the transaction
		for _, msg := range msgs {
			msgType := sdk.MsgTypeURL(msg)
			// Reject the transaction if one of the messages is not related to the x/acp or x/tier modules
			if !strings.HasPrefix(msgType, "/sourcehub.acp.") && !strings.HasPrefix(msgType, "/sourcehub.tier.") {
				return false
			}
		}
		// Accept the transaction only if all messages belong to the x/acp module
		return true
	}
}
