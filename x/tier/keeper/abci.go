package keeper

import (
	"context"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	appparams "github.com/sourcenetwork/sourcehub/app/params"
	"github.com/sourcenetwork/sourcehub/x/tier/types"
)

// BeginBlocker claims tier module staking rewards every N blocks.
// 2% of the claimed rewards is sent to the developer pool (DeveloperPoolFee).
// 1% is sent to the insurance pool (InsurancePoolFee) if insurance pool balance is below InsurancePoolThreshold,
// otherwise the InsurancePoolFee (1%) is also sent to the developer pool.
// Remaining 97% of the rewards is burned.
func (k *Keeper) BeginBlocker(ctx context.Context) error {
	params := k.GetParams(ctx)

	// Process rewards every N blocks
	height := sdk.UnwrapSDKContext(ctx).BlockHeight()
	if height%params.ProcessRewardsInterval != 0 {
		return nil
	}

	tierModuleAddr := authtypes.NewModuleAddress(types.ModuleName)
	// Iterate over all active delegations where the tier module account is the delegator
	// The max number of iterations is the number of validators it has delegated to
	err := k.GetStakingKeeper().IterateDelegations(ctx, tierModuleAddr, func(index int64, delegation stakingtypes.DelegationI) bool {
		// Claim rewards for the tier module from this validator
		valAddr := types.MustValAddressFromBech32(delegation.GetValidatorAddr())
		rewards, err := k.GetDistributionKeeper().WithdrawDelegationRewards(ctx, tierModuleAddr, valAddr)
		if err != nil {
			k.Logger().Error("Failed to claim tier module staking rewards", "error", err)
			return false
		}

		// Proceed to the next record if there are no rewards
		if rewards.IsZero() {
			k.Logger().Info("No tier module staking rewards in validator", "validator", valAddr)
			return false
		}

		totalAmount := rewards.AmountOf(appparams.DefaultBondDenom)
		amountToDevPool := totalAmount.MulRaw(params.DeveloperPoolFee).QuoRaw(100)
		amountToInsurancePool := totalAmount.MulRaw(params.InsurancePoolFee).QuoRaw(100)
		amountToBurn := totalAmount.Sub(amountToDevPool).Sub(amountToInsurancePool)

		// Send InsurancePoolFee to the insurance pool if threshold not reached, update amountToDevPool otherwise
		if !amountToInsurancePool.IsZero() {
			insurancePoolAddr := authtypes.NewModuleAddress(types.InsurancePoolName)
			insurancePoolBalance := k.GetBankKeeper().GetBalance(ctx, insurancePoolAddr, appparams.DefaultBondDenom)
			if insurancePoolBalance.Amount.Add(amountToInsurancePool).LTE(math.NewInt(params.InsurancePoolThreshold)) {
				insuranceCoins := sdk.NewCoins(sdk.NewCoin(appparams.DefaultBondDenom, amountToInsurancePool))
				err := k.GetBankKeeper().SendCoinsFromModuleToModule(ctx, types.ModuleName, types.InsurancePoolName, insuranceCoins)
				if err != nil {
					k.Logger().Error("Failed to send rewards to the insurance pool", "error", err)
					return false
				}
			} else {
				amountToDevPool = amountToDevPool.Add(amountToInsurancePool)
			}
		}

		// Send DeveloperPoolFee to the developer pool
		if !amountToDevPool.IsZero() {
			devPoolCoins := sdk.NewCoins(sdk.NewCoin(appparams.DefaultBondDenom, amountToDevPool))
			err := k.GetBankKeeper().SendCoinsFromModuleToModule(ctx, types.ModuleName, types.DeveloperPoolName, devPoolCoins)
			if err != nil {
				k.Logger().Error("Failed to send rewards to the developer pool", "error", err)
				return false
			}
		}

		// Burn remaining tier module staking rewards
		if !amountToBurn.IsZero() {
			burnCoins := sdk.NewCoins(sdk.NewCoin(appparams.DefaultBondDenom, amountToBurn))
			err := k.GetBankKeeper().BurnCoins(ctx, types.ModuleName, burnCoins)
			if err != nil {
				k.Logger().Error("Failed to burn tier module staking rewards", "error", err)
				return false
			}
		}

		return false
	})

	if err != nil {
		k.Logger().Error("Error iterating over tier module delegations", "error", err)
		return err
	}

	return nil
}
