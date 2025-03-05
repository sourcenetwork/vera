package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	appparams "github.com/sourcenetwork/sourcehub/app/params"
	"github.com/sourcenetwork/sourcehub/x/tier/types"
)

func (k *Keeper) repaySlashedTierStake(ctx context.Context, validatorAddr string, burnedAmount string) error {
	tierModuleAddr := authtypes.NewModuleAddress(types.ModuleName)
	valAddr := types.MustValAddressFromBech32(validatorAddr)

	// Get the tier module delegation
	tierDelegation, err := k.GetStakingKeeper().GetDelegation(ctx, tierModuleAddr, valAddr)
	if err != nil {
		return err
	}

	// Get the tier delegation shares
	tierStake := tierDelegation.Shares
	if tierStake.IsZero() {
		return fmt.Errorf("No delegation from the tier module")
	}

	// Get the slashed validator
	validator, err := k.GetStakingKeeper().GetValidator(ctx, valAddr)
	if err != nil {
		return err
	}

	// Get the total stake of the slashed validator
	totalStake := validator.Tokens.ToLegacyDec()
	if totalStake.IsZero() {
		return fmt.Errorf("No stake for the validator: %s", validatorAddr)
	}

	// Get burned (slashed) amount
	totalBurned, err := math.LegacyNewDecFromStr(burnedAmount)
	if err != nil {
		return err
	}
	if totalBurned.IsZero() {
		return fmt.Errorf("Total burned (slashed) amount is zero")
	}

	// Calculate tier module share of the burned (slashed) amount
	tierShareBurned := totalBurned.Mul(tierStake.Quo(totalStake)).TruncateInt()
	if tierShareBurned.IsZero() {
		return fmt.Errorf("Tier module burned (slashed) amount is zero")
	}

	// Make sure that the insurance pool has sufficient balance
	insurancePoolAddr := authtypes.NewModuleAddress(types.InsurancePoolName)
	insurancePoolBalance := k.GetBankKeeper().GetBalance(ctx, insurancePoolAddr, appparams.DefaultBondDenom)
	if insurancePoolBalance.Amount.LT(tierShareBurned) {
		// TODO: should we try to repay from the developer pool if insurance pool has insufficient balance?
		return fmt.Errorf("Insurance pool has insufficient balance: %d", insurancePoolBalance.Amount.Int64())
	}

	// Send tier module share of the burned (slasned) amount from the insurance pool to the tier module account
	repayCoins := sdk.NewCoins(sdk.NewCoin(appparams.DefaultBondDenom, tierShareBurned))
	err = k.GetBankKeeper().SendCoinsFromModuleToModule(ctx, types.InsurancePoolName, types.ModuleName, repayCoins)
	if err != nil {
		return err
	}

	// Delegate the tier module share of the burned (slasned) amount from back to the same validator
	_, err = k.GetStakingKeeper().Delegate(ctx, tierModuleAddr, tierShareBurned, stakingtypes.Unbonded, validator, true)
	if err != nil {
		return err
	}

	return nil
}

func (k *Keeper) handleSlashingEvents(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	events := sdkCtx.EventManager().Events()

	for _, event := range events {
		if event.Type == "slash" {
			var validatorAddr, reason, burnedAmount string

			for _, attr := range event.Attributes {
				switch string(attr.Key) {
				case "address":
					validatorAddr = string(attr.Value)
				case "reason":
					reason = string(attr.Value)
				case "burned":
					burnedAmount = string(attr.Value)
				}
			}

			if reason != slashingtypes.AttributeValueDoubleSign {
				return fmt.Errorf("Slashed tokens are repaid only in case of double signing. Reason was: %s", reason)
			}

			return k.repaySlashedTierStake(ctx, validatorAddr, burnedAmount)
		}
	}

	return nil
}

// BeginBlocker claims tier module staking rewards every N blocks.
// 2% of the claimed rewards is sent to the developer pool (DeveloperPoolFee).
// 1% is sent to the insurance pool (InsurancePoolFee) if insurance pool balance is below InsurancePoolThreshold,
// otherwise the InsurancePoolFee (1%) is also sent to the developer pool.
// Remaining 97% of the rewards is burned.
func (k *Keeper) BeginBlocker(ctx context.Context) error {
	err := k.handleSlashingEvents(ctx)
	if err != nil {
		k.Logger().Error("Failed to handle slashing event", "error", err)
	}

	params := k.GetParams(ctx)

	// Process rewards every N blocks
	height := sdk.UnwrapSDKContext(ctx).BlockHeight()
	if height%params.ProcessRewardsInterval != 0 {
		return nil
	}

	tierModuleAddr := authtypes.NewModuleAddress(types.ModuleName)
	// Iterate over all active delegations where the tier module account is the delegator
	// The max number of iterations is the number of validators it has delegated to
	err = k.GetStakingKeeper().IterateDelegations(ctx, tierModuleAddr, func(index int64, delegation stakingtypes.DelegationI) bool {
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
