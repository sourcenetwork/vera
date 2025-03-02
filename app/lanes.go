package app

import (
	"cosmossdk.io/math"

	signerextraction "github.com/skip-mev/block-sdk/v2/adapters/signer_extraction_adapter"
	"github.com/skip-mev/block-sdk/v2/block/base"
	defaultlane "github.com/skip-mev/block-sdk/v2/lanes/base"
	"github.com/sourcenetwork/sourcehub/app/lanes"
)

// CreateLanes creates and returns two lanes: priorityLane and defaultLane.
// Priority lane is a custom lane that matches all transactions related to the x/acp module.
// Default lane is the default BaseLane implementation.
func CreateLanes(app *App) (priorityLane *base.BaseLane, defaultLane *base.BaseLane) {
	// signerAdapter is used to extract the expected signers from a transaction
	signerAdapter := signerextraction.NewDefaultAdapter()

	// Create a priority lane configuration that consumes 40% of the block space
	priorityLaneConfig := base.LaneConfig{
		Logger:          app.Logger(),
		TxEncoder:       app.txConfig.TxEncoder(),
		TxDecoder:       app.txConfig.TxDecoder(),
		MaxBlockSpace:   math.LegacyMustNewDecFromStr("0.4"),
		SignerExtractor: signerAdapter,
		MaxTxs:          0,
	}

	// Create a default lane configuration that consumes 60% of the block space
	defaultLaneConfig := base.LaneConfig{
		Logger:          app.Logger(),
		TxEncoder:       app.txConfig.TxEncoder(),
		TxDecoder:       app.txConfig.TxDecoder(),
		MaxBlockSpace:   math.LegacyMustNewDecFromStr("0.6"),
		SignerExtractor: signerAdapter,
		MaxTxs:          0,
	}

	// Create the TxPriority for the priority lane
	priorityLaneTxPriority := lanes.TxPriority()

	// Create the match handler for the priority lane
	priorityLaneMatchHandler := lanes.PriorityMatchHandler()

	// Create the match handlers for the default lane
	defaultLaneMatchHandler := base.DefaultMatchHandler()

	// Create priority lane
	priorityLane = lanes.NewPriorityLane(
		priorityLaneConfig,
		priorityLaneTxPriority,
		priorityLaneMatchHandler,
	)

	// Create default lane
	defaultLane = defaultlane.NewDefaultLane(
		defaultLaneConfig,
		defaultLaneMatchHandler,
	)

	return priorityLane, defaultLane
}
