package keeper

import (
	"context"
	"errors"
	"fmt"
	"time"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/sourcenetwork/sourcehub/app/metrics"
	appparams "github.com/sourcenetwork/sourcehub/app/params"
	"github.com/sourcenetwork/sourcehub/x/tier/types"
)

// getTotalCreditAmount retrieves the total credit amount from the store.
func (k *Keeper) getTotalCreditAmount(ctx context.Context) (total math.Int) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get(types.TotalCreditsKey)
	if bz == nil {
		return math.ZeroInt()
	}

	err := total.Unmarshal(bz)
	if err != nil {
		return math.ZeroInt()
	}

	if total.IsNegative() {
		return math.ZeroInt()
	}

	return total
}

// setTotalCreditAmount updates the total credit amount in the store.
func (k *Keeper) setTotalCreditAmount(ctx context.Context, total math.Int) error {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz, err := total.Marshal()
	if err != nil {
		return errorsmod.Wrapf(err, "marshal total credit amount")
	}

	store.Set(types.TotalCreditsKey, bz)

	// Update total credit amount gauge if we are not resetting the total
	if total.IsPositive() {
		telemetry.ModuleSetGauge(
			types.ModuleName,
			float32(total.Int64()),
			metrics.TotalCredits,
		)
	}

	return nil
}

// mintCredit mints ucredit amount and sends it to the specified address.
func (k *Keeper) mintCredit(ctx context.Context, addr sdk.AccAddress, amount math.Int) error {
	if _, err := sdk.AccAddressFromBech32(addr.String()); err != nil {
		return errorsmod.Wrap(err, "invalid address")
	}

	if amount.LTE(math.ZeroInt()) {
		return errors.New("invalid amount")
	}

	coins := sdk.NewCoins(sdk.NewCoin(appparams.MicroCreditDenom, amount))
	err := k.bankKeeper.MintCoins(ctx, types.ModuleName, coins)
	if err != nil {
		return errorsmod.Wrap(err, "mint coins")
	}

	err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, addr, coins)
	if err != nil {
		return errorsmod.Wrap(err, "send coins from module to account")
	}

	return nil
}

// proratedCredit calculates the credits earned on the lockingAmt.
func (k *Keeper) proratedCredit(ctx context.Context, delAddr sdk.AccAddress, lockingAmt math.Int) math.Int {
	rates := k.GetParams(ctx).RewardRates
	epochInfo := k.epochsKeeper.GetEpochInfo(ctx, types.EpochIdentifier)

	lockedAmt := k.totalLockedAmountByAddr(ctx, delAddr)
	insuredAmt := k.totalInsuredAmountByAddr(ctx, delAddr)
	totalAmt := lockedAmt.Add(insuredAmt)

	return calculateProratedCredit(
		rates,
		totalAmt,
		lockingAmt,
		epochInfo.CurrentEpochStartTime,
		sdk.UnwrapSDKContext(ctx).BlockTime(),
		epochInfo.Duration,
	)
}

// burnAllCredits burns all the reward credits in the system.
// It is called at the end of each epoch.
func (k *Keeper) burnAllCredits(ctx context.Context, epochNumber int64) (err error) {
	start := time.Now()

	defer func() {
		metrics.ModuleMeasureSinceWithCounter(
			types.ModuleName,
			metrics.BurnAllCredits,
			start,
			err,
			[]metrics.Label{
				metrics.NewLabel(metrics.Epoch, fmt.Sprintf("%d", epochNumber)),
			},
		)
	}()

	// We iterate through all the balances to find and burn the credits. It could be
	// improved to iterate through the lockup records because credits are NOT transferrable.
	unusedCredits := math.ZeroInt()
	cb := func(addr sdk.AccAddress, coin sdk.Coin) (stop bool) {
		if coin.Denom != appparams.MicroCreditDenom {
			return false
		}

		coins := sdk.NewCoins(coin)

		err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, addr, types.ModuleName, coins)
		if err != nil {
			err = errorsmod.Wrapf(err, "send %s ucredit from %s to module", coins, addr)
			return true
		}

		err = k.bankKeeper.BurnCoins(ctx, types.ModuleName, coins)
		if err != nil {
			err = errorsmod.Wrapf(err, "burn %s ucredit", coins)
			return true
		}

		unusedCredits.Add(coin.Amount)

		return false
	}

	k.bankKeeper.IterateAllBalances(ctx, cb)

	totalCredits := k.getTotalCreditAmount(ctx)
	if totalCredits.IsPositive() {
		creditUtilization, err := unusedCredits.ToLegacyDec().Quo(totalCredits.ToLegacyDec()).Float64()
		if err != nil {
			return errorsmod.Wrap(err, "calculate credit utilization")
		}

		// Update credit utilization gauge
		telemetry.ModuleSetGauge(
			types.ModuleName,
			float32(creditUtilization),
			metrics.CreditUtilization, fmt.Sprintf("%s_%d", metrics.Epoch, epochNumber),
		)
	}

	// Reset total credit amount to 0 after burning
	k.setTotalCreditAmount(ctx, math.ZeroInt())

	return err
}

// resetAllCredits resets all the credits in the system.
func (k *Keeper) resetAllCredits(ctx context.Context) error {
	defer telemetry.ModuleMeasureSince(types.ModuleName, time.Now(), metrics.ResetAllCredits, metrics.Latency)

	// Reward to a delegator is calculated based on the total locked amount
	// to all validators. Since each lockup entry only records locked amount
	// for a single validator, we need to iterate through all the lockups to
	// calculate the total locked amount for each delegator.
	lockedAmts := make(map[string]math.Int)

	cb := func(delAddr sdk.AccAddress, valAddr sdk.ValAddress, lockup types.Lockup) {
		amt, ok := lockedAmts[delAddr.String()]
		if !ok {
			amt = math.ZeroInt()
		}
		// Include associated insurance lockup amounts to allocate credits correctly
		insuranceLockupAmount := k.getInsuranceLockupAmount(ctx, delAddr, valAddr)
		lockedAmts[delAddr.String()] = amt.Add(lockup.Amount).Add(insuranceLockupAmount)
	}

	k.mustIterateLockups(ctx, cb)

	rates := k.GetParams(ctx).RewardRates

	totalCredit := math.ZeroInt()
	for delStrAddr, amount := range lockedAmts {
		delAddr := sdk.MustAccAddressFromBech32(delStrAddr)
		credit := calculateCredit(rates, math.ZeroInt(), amount)
		err := k.mintCredit(ctx, delAddr, credit)
		if err != nil {
			return errorsmod.Wrapf(err, "mint %s ucredit to %s", credit, delAddr)
		}
		totalCredit.Add(credit)
	}

	// Set total credit amount
	err := k.setTotalCreditAmount(ctx, totalCredit)
	if err != nil {
		return errorsmod.Wrap(err, "set total credit amount")
	}

	return nil
}
