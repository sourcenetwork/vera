package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/sourcenetwork/sourcehub/app/metrics"
	appparams "github.com/sourcenetwork/sourcehub/app/params"
	"github.com/sourcenetwork/sourcehub/x/tier/types"
)

// handleSlashingEvents monitors and handles slashing events.
// In case of double_sign, existing lockup records are updated to reflect changes after slashing.
// Otherwise, in addition to updating existing lockup records, slashed tokens are covered via insurance lockups.
// Note: beginBlockers in app_config.go must have tier module after slashing for events to be handled correctly.
func (k *Keeper) handleSlashingEvents(ctx context.Context) {
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
				err := k.handleDoubleSign(ctx, validatorAddr, slashedAmount)
				if err != nil {
					metrics.ModuleIncrInternalErrorCounter(types.ModuleName, metrics.HandleDoubleSign, err)
					k.Logger().Error("Failed to handle double sign event", "error", err)
				}
			} else {
				err := k.handleMissingSignature(ctx, validatorAddr, slashedAmount)
				if err != nil {
					metrics.ModuleIncrInternalErrorCounter(types.ModuleName, metrics.HandleMissingSignature, err)
					k.Logger().Error("Failed to handle missing signature event", "error", err)
				}
			}
		}
	}
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
