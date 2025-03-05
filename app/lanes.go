package app

import (
	"cosmossdk.io/math"

	signerextraction "github.com/skip-mev/block-sdk/v2/adapters/signer_extraction_adapter"
	"github.com/skip-mev/block-sdk/v2/block/base"
	defaultlane "github.com/skip-mev/block-sdk/v2/lanes/base"
	"github.com/sourcenetwork/sourcehub/app/lanes"
)

// CreateLanes creates and returns two lanes: priorityLane and defaultLane.
// Priority lane uses PriorityMatchHandler to match all transactions related to the x/acp and x/tier modules.
// Default lane is the default BaseLane implementation.
func CreateLanes(app *App) (priorityLane *base.BaseLane, defaultLane *base.BaseLane) {
	// signerAdapter is used to extract the expected signers from a transaction
	signerAdapter := signerextraction.NewDefaultAdapter()

	// Create a priority lane configuration that occupies 40% of the block space
	priorityLaneConfig := base.LaneConfig{
		Logger:          app.Logger(),
		TxEncoder:       app.txConfig.TxEncoder(),
		TxDecoder:       app.txConfig.TxDecoder(),
		MaxBlockSpace:   math.LegacyMustNewDecFromStr("0.4"),
		SignerExtractor: signerAdapter,
		MaxTxs:          0,
	}

	// Create a default lane configuration that occupies 60% of the block space
	defaultLaneConfig := base.LaneConfig{
		Logger:          app.Logger(),
		TxEncoder:       app.txConfig.TxEncoder(),
		TxDecoder:       app.txConfig.TxDecoder(),
		MaxBlockSpace:   math.LegacyMustNewDecFromStr("0.6"),
		SignerExtractor: signerAdapter,
		MaxTxs:          0,
	}

	// Create TxPriority for the priority lane
	priorityLaneTxPriority := lanes.TxPriority()

	// Create match handler for the priority lane
	priorityLaneMatchHandler := lanes.PriorityMatchHandler()

	// Create match handler for the default lane
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
