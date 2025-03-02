package lanes

import (
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/skip-mev/block-sdk/v2/block/base"
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

// DefaultMatchHandler returns the default match handler for the priority lane.
// The default implementation matches transactions related to the x/acp module.
func DefaultMatchHandler() base.MatchHandler {
	return func(_ sdk.Context, tx sdk.Tx) bool {
		msgs := tx.GetMsgs()
		// Return false if there are no messages
		if len(msgs) == 0 {
			return false
		}
		// Check message types of all messages in the transaction
		for _, msg := range msgs {
			msgType := sdk.MsgTypeURL(msg)
			// Reject the transaction if one of the messages is not related to the x/acp module
			if !strings.HasPrefix(msgType, "/sourcehub.acp.") {
				return false
			}
		}
		// Accept the transaction only if all messages belong to the x/acp module
		return false
	}
}
