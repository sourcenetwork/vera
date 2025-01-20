package keeper

import (
	"time"

	"cosmossdk.io/math"

	"github.com/sourcenetwork/sourcehub/x/tier/types"
)

// calculateCredit calculates the reward earned on the lockingAmt.
// lockingAmt is stacked up on top of the lockedAmt to earn at the highest eligible reward.
func calculateCredit(rateList []types.Rate, lockedAmt, lockingAmt math.Int) math.Int {
	credit := math.ZeroInt()
	stakedAmt := lockedAmt.Add(lockingAmt)

	// Iterate from the highest reward rate to the lowest.
	for _, r := range rateList {
		// Continue if the total lock does not reach the current rate requirement.
		if stakedAmt.LT(r.Amount) {
			continue
		}

		lower := math.MaxInt(r.Amount, lockedAmt)
		diff := stakedAmt.Sub(lower)

		diffDec := math.LegacyNewDecFromInt(diff)
		rateDec := math.LegacyNewDec(r.Rate)

		// rateDec MUST have 2 decimals of precision for the calculation to be correct.
		amt := diffDec.Mul(rateDec).Quo(math.LegacyNewDec(100))
		credit = credit.Add(amt.TruncateInt())

		// Subtract the lock that has been rewarded.
		stakedAmt = stakedAmt.Sub(diff)
		lockingAmt = lockingAmt.Sub(diff)

		// Break if all the new lock has been rewarded.
		if lockingAmt.IsZero() {
			break
		}
	}

	return credit
}

func calculateProratedCredit(
	rates []types.Rate,
	lockedAmt, lockingAmt math.Int,
	currentEpochStartTime, currentBlockTime time.Time,
	epochDuration time.Duration,
) math.Int {
	// Calculate the reward credits earned on the new lock.
	credit := calculateCredit(rates, lockedAmt, lockingAmt)

	// Prorate the credit based on the time elapsed in the current epoch.
	sinceCurrentEpoch := currentBlockTime.Sub(currentEpochStartTime).Milliseconds()
	epochDurationMs := epochDuration.Milliseconds()

	if epochDurationMs == 0 {
		return math.ZeroInt()
	}

	// This check is required because is possible that sinceCurrentEpoch can be greater than epochDuration
	// (e.g. chain paused for longer than the epoch duration or misconfigured epoch duration).
	if sinceCurrentEpoch < epochDurationMs {
		credit = credit.MulRaw(sinceCurrentEpoch).QuoRaw(epochDurationMs)
	}

	return credit
}
