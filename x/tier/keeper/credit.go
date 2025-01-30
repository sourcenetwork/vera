package keeper

import (
	"context"
	"errors"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	appparams "github.com/sourcenetwork/sourcehub/app/params"
	"github.com/sourcenetwork/sourcehub/x/tier/types"
)

// mintCredit mints a coin and sends it to the specified address.
func (k Keeper) mintCredit(ctx context.Context, addr sdk.AccAddress, amt math.Int) error {
	if _, err := sdk.AccAddressFromBech32(addr.String()); err != nil {
		return errorsmod.Wrap(err, "invalid address")
	}

	if amt.LTE(math.ZeroInt()) {
		return errors.New("invalid amount")
	}

	coins := sdk.NewCoins(sdk.NewCoin(appparams.MicroCreditDenom, amt))
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
func (k Keeper) proratedCredit(ctx context.Context, delAddr sdk.AccAddress, lockingAmt math.Int) math.Int {
	rates := k.GetParams(ctx).RewardRates
	lockedAmt := k.TotalAmountByAddr(ctx, delAddr)
	epochInfo := k.epochsKeeper.GetEpochInfo(ctx, types.EpochIdentifier)

	return calculateProratedCredit(
		rates,
		lockedAmt,
		lockingAmt,
		epochInfo.CurrentEpochStartTime,
		sdk.UnwrapSDKContext(ctx).BlockTime(),
		epochInfo.Duration,
	)
}

// burnAllCredits burns all the reward credits in the system.
// It is called at the end of each epoch.
func (k Keeper) burnAllCredits(ctx context.Context) error {
	// Note that we can't simply iterate through the lockup records because credits
	// are transferrable and can be stored in accounts that are not tracked by lockups.
	// Instead, we iterate through all the balances to find and burn the credits.
	var err error

	cb := func(addr sdk.AccAddress, coin sdk.Coin) (stop bool) {
		if coin.Denom != appparams.MicroCreditDenom {
			return false
		}

		coins := sdk.NewCoins(coin)

		err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, addr, types.ModuleName, coins)
		if err != nil {
			err = errorsmod.Wrapf(err, "send %s from %s to module", coins, addr)
			return true
		}

		err = k.bankKeeper.BurnCoins(ctx, types.ModuleName, coins)
		if err != nil {
			err = errorsmod.Wrapf(err, "burn %s", coins)
			return true
		}

		return false
	}

	k.bankKeeper.IterateAllBalances(ctx, cb)

	return err
}

// resetAllCredits resets all the credits in the system.
func (k Keeper) resetAllCredits(ctx context.Context) error {
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
		lockedAmts[delAddr.String()] = amt.Add(lockup.Amount)
	}

	k.MustIterateLockups(ctx, cb)

	rates := k.GetParams(ctx).RewardRates

	for delStrAddr, amt := range lockedAmts {
		delAddr := sdk.MustAccAddressFromBech32(delStrAddr)
		credit := calculateCredit(rates, math.ZeroInt(), amt)
		err := k.mintCredit(ctx, delAddr, credit)
		if err != nil {
			return errorsmod.Wrapf(err, "mint %s to %s", credit, delAddr)
		}
	}

	return nil
}
