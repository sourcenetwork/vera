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

// BeginBlocker handles slashing events and processes tier module staking rewards.
func (k *Keeper) BeginBlocker(ctx context.Context) error {
	err := k.handleSlashingEvents(ctx)
	if err != nil {
		k.Logger().Error("Failed to handle slashing event", "error", err)
	}

	err = k.processRewards(ctx)
	if err != nil {
		k.Logger().Error("Failed to process rewards", "error", err)
	}

	return nil
}

// handleSlashingEvents monitors and handles slashing events.
// In case of double_sign, existing lockup records are updated to reflect changes after slashing.
// Otherwise, in addition to updating existing lockup records, slashed tokens are covered via insurance lockups.
func (k *Keeper) handleSlashingEvents(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	events := sdkCtx.EventManager().Events()

	for _, event := range events {
		if event.Type == "slash" {
			var validatorAddr, reason, slashedAmount string

			for _, attr := range event.Attributes {
				switch string(attr.Key) {
				case "address":
					validatorAddr = string(attr.Value)
				case "reason":
					reason = string(attr.Value)
				case "burned":
					slashedAmount = string(attr.Value)
				}
			}

			if reason == slashingtypes.AttributeValueDoubleSign {
				return k.handleDoubleSign(ctx, validatorAddr, slashedAmount)
			} else {
				return k.handleMissingSignature(ctx, validatorAddr, slashedAmount)
			}
		}
	}

	return nil
}

// handleDoubleSign adjusts existing lockup records based on the tier module share of the slashed amount.
func (k *Keeper) handleDoubleSign(ctx context.Context, validatorAddr string, slashedAmount string) error {
	tierModuleAddr := authtypes.NewModuleAddress(types.ModuleName)
	valAddr, err := sdk.ValAddressFromBech32(validatorAddr)
	if err != nil {
		return err
	}

	// Get total slashed amount
	totalSlashed, err := math.LegacyNewDecFromStr(slashedAmount)
	if err != nil {
		return err
	}
	if totalSlashed.IsZero() {
		return fmt.Errorf("Total slashed amount is zero")
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

	// Get tier module delegation
	tierDelegation, err := k.GetStakingKeeper().GetDelegation(ctx, tierModuleAddr, valAddr)
	if err != nil {
		return err
	}

	// Get tier module delegation shares
	tierShares := tierDelegation.Shares
	if tierShares.IsZero() {
		return fmt.Errorf("No delegation from the tier module")
	}

	// Get tier module stake from the delegation shares
	tierStake := validator.TokensFromSharesTruncated(tierShares)

	// Calculate the amount slashed from the tier module stake
	tierStakeSlashed := totalSlashed.Mul(tierStake.Quo(totalStake))
	if tierStakeSlashed.IsZero() {
		return fmt.Errorf("Tier module slashed amount is zero")
	}

	// Get the rate by which every individual lockup record should be adjusted
	slashingRate := tierStake.Sub(tierStakeSlashed).Quo(tierStake)

	// Adjust affected lockups based on the slashed amount (no insurance lockups created since coverageRate is 0)
	return k.adjustLockups(ctx, valAddr, slashingRate, math.LegacyZeroDec())
}

// handleMissingSignature adjusts existing lockup records based on the tier module share of the slashed amount
// and covers tier module share of the slashed tokens from the insurance pool.
func (k *Keeper) handleMissingSignature(ctx context.Context, validatorAddr string, slashedAmount string) error {
	tierModuleAddr := authtypes.NewModuleAddress(types.ModuleName)
	valAddr, err := sdk.ValAddressFromBech32(validatorAddr)
	if err != nil {
		return err
	}

	// Get total slashed amount
	totalSlashed, err := math.LegacyNewDecFromStr(slashedAmount)
	if err != nil {
		return err
	}
	if totalSlashed.IsZero() {
		return fmt.Errorf("Total slashed amount is zero")
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

	// Get tier module delegation
	tierDelegation, err := k.GetStakingKeeper().GetDelegation(ctx, tierModuleAddr, valAddr)
	if err != nil {
		return err
	}

	// Get tier module delegation shares
	tierShares := tierDelegation.Shares
	if tierShares.IsZero() {
		return fmt.Errorf("No delegation from the tier module")
	}

	// Get tier module stake from the delegation shares
	tierStake := validator.TokensFromSharesTruncated(tierShares)

	// Calculate tier module share of the slashed amount
	tierStakeSlashed := totalSlashed.Mul(tierStake.Quo(totalStake)).Ceil().TruncateInt()
	if tierStakeSlashed.IsZero() {
		return fmt.Errorf("Tier module slashed amount is zero")
	}

	insurancePoolAddr := authtypes.NewModuleAddress(types.InsurancePoolName)
	insurancePoolBalance := k.GetBankKeeper().GetBalance(ctx, insurancePoolAddr, appparams.DefaultBondDenom)
	coveredAmount := tierStakeSlashed

	// If tierStakeSlashed exceeds insurancePoolBalance, cover as much as there is on the insurance pool balance
	if insurancePoolBalance.Amount.LT(tierStakeSlashed) {
		coveredAmount = insurancePoolBalance.Amount
	}

	// Delegate covered amount back to the same validator on behalf of the insurance pool module account
	_, err = k.GetStakingKeeper().Delegate(
		ctx,
		insurancePoolAddr,
		coveredAmount,
		stakingtypes.Unbonded,
		validator,
		true,
	)
	if err != nil {
		return err
	}

	// Calculate the proportional rate to reduce each individual lockup after slashing
	slashingRate := tierStake.Sub(tierStakeSlashed.ToLegacyDec()).Quo(tierStake)

	// Calculate the fraction of the original tier stake that is covered by the insurance pool
	coverageRate := coveredAmount.ToLegacyDec().Quo(tierStake)

	// Adjust affected lockups based on the slashed amount and create/update associated insurance lockups based on the coverageRate
	return k.adjustLockups(ctx, valAddr, slashingRate, coverageRate)
}

// processRewards processes block rewards every ProcessRewardsInterval blocks.
// 2% of the claimed rewards is sent to the developer pool (DeveloperPoolFee).
// 1% is sent to the insurance pool (InsurancePoolFee) if insurance pool balance is below InsurancePoolThreshold,
// otherwise the InsurancePoolFee (1%) is also sent to the developer pool.
// Remaining 97% of the rewards is burned.
func (k *Keeper) processRewards(ctx context.Context) error {
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

	return err
}
