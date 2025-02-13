package types

import time "time"

// Tier module constants
const (
	EpochIdentifier               = ModuleName
	DefaultUnlockingEpochs        = 2
	DefaultDeveloperPoolFee       = 2
	DefaultInsurancePoolFee       = 1
	DefaultInsurancePoolThreshold = 100_000_000_000
	DefaultEpochDuration          = time.Minute * 5
)
