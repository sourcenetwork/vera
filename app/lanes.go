package app

import (
	"cosmossdk.io/math"

	signerextraction "github.com/skip-mev/block-sdk/v2/adapters/signer_extraction_adapter"
	"github.com/skip-mev/block-sdk/v2/block/base"
	"github.com/sourcenetwork/sourcehub/app/lanes"
)

// CreatePriorityLane creates a lane that matches all txs and occupies 100% of the block space.
func CreatePriorityLane(app *App) (priorityLane *base.BaseLane) {
	// signerAdapter is used to extract the expected signers from a transaction
	signerAdapter := signerextraction.NewDefaultAdapter()

	// Create a priority lane configuration that occupies 100% of the block space
	priorityLaneConfig := base.LaneConfig{
		Logger:          app.Logger(),
		TxEncoder:       app.txConfig.TxEncoder(),
		TxDecoder:       app.txConfig.TxDecoder(),
		MaxBlockSpace:   math.LegacyMustNewDecFromStr("1.0"),
		SignerExtractor: signerAdapter,
		MaxTxs:          0,
	}

	// Create TxPriority for the priority lane
	priorityLaneTxPriority := lanes.TxPriority()

	// Use default match handler to match all transactions
	defaultLaneMatchHandler := base.DefaultMatchHandler()

	// Create priority lane
	priorityLane = lanes.NewPriorityLane(
		priorityLaneConfig,
		priorityLaneTxPriority,
		defaultLaneMatchHandler,
	)

	return priorityLane
}
