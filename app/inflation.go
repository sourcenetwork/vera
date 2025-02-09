package app

import (
	"context"

	"cosmossdk.io/math"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	appparams "github.com/sourcenetwork/sourcehub/app/params"
	tierkeeper "github.com/sourcenetwork/sourcehub/x/tier/keeper"
)

func ProvideInflationCalculationFn(
	tierKeeper tierkeeper.Keeper,
	bankKeeper bankkeeper.Keeper,
	stakingKeeper *stakingkeeper.Keeper,
) minttypes.InflationCalculationFn {
	// Adjust bonded ratio based on the x/tier module developer stake so that it does not affect rewards.
	return func(ctx context.Context, minter minttypes.Minter, params minttypes.Params, bondedRatio math.LegacyDec) math.LegacyDec {
		devStake := tierKeeper.GetTotalLockupsAmount(ctx)
		totalSupply := bankKeeper.GetSupply(ctx, appparams.DefaultBondDenom).Amount

		totalBonded, err := stakingKeeper.TotalBondedTokens(ctx)
		if err != nil {
			totalBonded = math.ZeroInt()
		}

		adjustedBonded := totalBonded.Sub(devStake)
		adjustedBondedRatio := adjustedBonded.ToLegacyDec().QuoInt(totalSupply)

		return minter.NextInflationRate(params, adjustedBondedRatio)
	}
}
