package types

import time "time"

// Tier module constants
const (
	EpochIdentifier               = ModuleName      // tier
	DefaultEpochDuration          = time.Minute * 5 // 5 minutes
	DefaultUnlockingEpochs        = 2               // 2 epochs
	DefaultDeveloperPoolFee       = 2               // 2%
	DefaultInsurancePoolFee       = 1               // 1%
	DefaultInsurancePoolThreshold = 100_000_000_000 // 100,000 open
	DefaultProcessRewardsInterval = 1000            // process rewards every 1000 blocks
)
