package keeper

import (
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
