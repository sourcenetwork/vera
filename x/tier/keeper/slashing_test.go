package keeper

import (
	"fmt"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/sourcenetwork/sourcehub/x/tier/types"
	"github.com/stretchr/testify/require"
)

func TestHandleSlashingEvents_NoEvent(t *testing.T) {
	k, ctx := setupKeeper(t)

	ctx = ctx.WithEventManager(sdk.NewEventManager())
	k.handleSlashingEvents(ctx)
}

func TestHandleSlashingEvents_DoubleSign(t *testing.T) {
	k, ctx := setupKeeper(t)

	slashAmount := math.NewInt(10_000)
	initialDelegatorBalance := math.NewInt(100_000)
	initialValidatorBalance := math.NewInt(100_000)

	delAddr, err := sdk.AccAddressFromBech32("source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et")
	require.NoError(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	initializeDelegator(t, &k, ctx, delAddr, initialDelegatorBalance)
	initializeValidator(t, k.GetStakingKeeper().(*stakingkeeper.Keeper), ctx, valAddr, initialValidatorBalance)

	err = k.Lock(ctx, delAddr, valAddr, initialDelegatorBalance)
	require.NoError(t, err)

	ctx = ctx.WithEventManager(sdk.NewEventManager())
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"slash",
		sdk.NewAttribute("address", valAddr.String()),
		sdk.NewAttribute("reason", slashingtypes.AttributeValueDoubleSign),
		sdk.NewAttribute("burned", slashAmount.String()),
	))

	k.handleSlashingEvents(ctx)

	// initialDelegatorBalance - slashAmount * initialDelegatorBalance / totalStaked
	// 100_000 - (10_000 * 100_000 / 200_000) = 95_000
	expectedTotalLockedAmount := math.NewInt(95_000)
	totalLockedAmount := k.GetTotalLockupsAmount(ctx)
	require.Equal(t, expectedTotalLockedAmount, totalLockedAmount)

	// insured amount should be zero because double_sign events are not covered
	insuredAmount := k.getInsuranceLockupAmount(ctx, delAddr, valAddr)
	require.Equal(t, math.ZeroInt(), insuredAmount)

	// insurance pool should not have any delegation
	insurancePoolAddr := authtypes.NewModuleAddress(types.InsurancePoolName)
	_, err = k.GetStakingKeeper().GetDelegation(ctx, insurancePoolAddr, valAddr)
	require.Error(t, err)
}

func TestHandleSlashingEvents_MissingSignature(t *testing.T) {
	k, ctx := setupKeeper(t)

	slashAmount := math.NewInt(10_000)
	insurancePoolBalance := math.NewInt(1_000_000)
	initialDelegatorBalance := math.NewInt(100_000)
	initialValidatorBalance := math.NewInt(100_000)

	delAddr, err := sdk.AccAddressFromBech32("source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et")
	require.NoError(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	initializeDelegator(t, &k, ctx, delAddr, initialDelegatorBalance)
	initializeValidator(t, k.GetStakingKeeper().(*stakingkeeper.Keeper), ctx, valAddr, initialValidatorBalance)
	mintCoinsToModule(t, &k, ctx, types.InsurancePoolName, insurancePoolBalance)

	err = k.Lock(ctx, delAddr, valAddr, initialDelegatorBalance)
	require.NoError(t, err)

	ctx = ctx.WithEventManager(sdk.NewEventManager())
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"slash",
		sdk.NewAttribute("address", valAddr.String()),
		sdk.NewAttribute("reason", slashingtypes.AttributeValueMissingSignature),
		sdk.NewAttribute("burned", slashAmount.String()),
	))

	k.handleSlashingEvents(ctx)

	// initialDelegatorBalance - slashAmount * initialDelegatorBalance / totalStaked
	// 100_000 - (10_000 * 100_000 / 200_000) = 95_000
	expectedTotalLockedAmount := math.NewInt(95_000)
	totalLockedAmount := k.GetTotalLockupsAmount(ctx)
	require.Equal(t, expectedTotalLockedAmount, totalLockedAmount)

	// slashAmount * initialDelegatorBalance / totalStaked
	// 10_000 * 100_000 / 200_000 = 5_000
	expectedInsuredAmount := math.NewInt(5_000)
	insuredAmount := k.getInsuranceLockupAmount(ctx, delAddr, valAddr)
	require.Equal(t, expectedInsuredAmount, insuredAmount)

	// insurance pool should have correct amount delegated
	insurancePoolAddr := authtypes.NewModuleAddress(types.InsurancePoolName)
	delegation, err := k.GetStakingKeeper().GetDelegation(ctx, insurancePoolAddr, valAddr)
	require.NoError(t, err)
	require.Equal(t, expectedInsuredAmount, delegation.Shares.RoundInt())
}

func TestHandleDoubleSign_MultipleDelegators(t *testing.T) {
	k, ctx := setupKeeper(t)

	slashAmount := math.NewInt(10_000)
	initialDelegatorBalance1 := math.NewInt(60_000)
	initialDelegatorBalance2 := math.NewInt(40_000)
	initialDelegatorBalance3 := math.NewInt(100_000)
	initialValidatorBalance := math.ZeroInt()
	totalTierStake := initialDelegatorBalance1.Add(initialDelegatorBalance2)

	delAddr1, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)
	delAddr2, err := sdk.AccAddressFromBech32("source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et")
	require.NoError(t, err)
	delAddr3, err := sdk.AccAddressFromBech32("source18jtkvj0995fy7lggqayg2f5syna92ndq5mkuv4")
	require.NoError(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	initializeDelegator(t, &k, ctx, delAddr1, initialDelegatorBalance1)
	initializeDelegator(t, &k, ctx, delAddr2, initialDelegatorBalance2)
	initializeDelegator(t, &k, ctx, delAddr3, initialDelegatorBalance3)
	initializeValidator(t, k.GetStakingKeeper().(*stakingkeeper.Keeper), ctx, valAddr, initialValidatorBalance)

	// delAddr1 locks via tier module, will be affected
	require.NoError(t, k.Lock(ctx, delAddr1, valAddr, initialDelegatorBalance1))

	// delAddr2 locks via tier module, will also be affected
	require.NoError(t, k.Lock(ctx, delAddr2, valAddr, initialDelegatorBalance2))

	validator, err := k.GetStakingKeeper().GetValidator(ctx, valAddr)
	require.NoError(t, err)

	// delAddr3 delegates normally, will not be affected
	_, err = k.GetStakingKeeper().Delegate(ctx, delAddr3, initialDelegatorBalance3, stakingtypes.Unbonded, validator, true)
	require.NoError(t, err)

	ctx = ctx.WithEventManager(sdk.NewEventManager())
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"slash",
		sdk.NewAttribute("address", valAddr.String()),
		sdk.NewAttribute("reason", slashingtypes.AttributeValueDoubleSign),
		sdk.NewAttribute("burned", slashAmount.String()),
	))

	// handle double sign event
	require.NoError(t, k.handleDoubleSign(ctx, valAddr.String(), slashAmount.String()))

	// tier module stake = 100_000, total stake = 200_000, tier share of slash = 5_000
	expectedSlashed := math.NewInt(5_000)
	require.Equal(t, totalTierStake.Sub(expectedSlashed), k.GetTotalLockupsAmount(ctx))

	// insured amounts for all delegators should be zero because double_sign events are not covered
	insuredAmount := k.getInsuranceLockupAmount(ctx, delAddr1, valAddr)
	require.Equal(t, math.ZeroInt(), insuredAmount)
	insuredAmount = k.getInsuranceLockupAmount(ctx, delAddr2, valAddr)
	require.Equal(t, math.ZeroInt(), insuredAmount)
	insuredAmount = k.getInsuranceLockupAmount(ctx, delAddr3, valAddr)
	require.Equal(t, math.ZeroInt(), insuredAmount)

	// insurance pool should not have any delegation
	insurancePoolAddr := authtypes.NewModuleAddress(types.InsurancePoolName)
	_, err = k.GetStakingKeeper().GetDelegation(ctx, insurancePoolAddr, valAddr)
	require.Error(t, err)
}

func TestHandleMissingSignature_PartialCoverage(t *testing.T) {
	k, ctx := setupKeeper(t)

	// slashed amount is greater than insurance pool balance
	slashAmount := math.NewInt(10_000)
	insuracePoolBalance := math.NewInt(3_000)
	initialDelegatorBalance := math.NewInt(100_000)
	initialValidatorBalance := math.ZeroInt()

	delAddr, err := sdk.AccAddressFromBech32("source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et")
	require.NoError(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	initializeDelegator(t, &k, ctx, delAddr, initialDelegatorBalance)
	initializeValidator(t, k.GetStakingKeeper().(*stakingkeeper.Keeper), ctx, valAddr, initialValidatorBalance)
	mintCoinsToModule(t, &k, ctx, types.InsurancePoolName, insuracePoolBalance)

	err = k.Lock(ctx, delAddr, valAddr, initialDelegatorBalance)
	require.NoError(t, err)

	validator, _ := k.GetStakingKeeper().GetValidator(ctx, valAddr)
	fmt.Println("Validator Tokens:", validator.Tokens)

	tierModuleAddr := authtypes.NewModuleAddress(types.ModuleName)
	delegation, _ := k.GetStakingKeeper().GetDelegation(ctx, tierModuleAddr, valAddr)
	fmt.Println("Tier Shares:", delegation.Shares)

	fmt.Println("Validator DelegatorShares:", validator.DelegatorShares)

	err = k.handleMissingSignature(ctx, valAddr.String(), slashAmount.String())
	require.NoError(t, err)

	// slashed amount from tier stake = 10_000 * 100_000 / 100_000 = 10_000
	expectedTierSlashed := math.NewInt(10_000)

	// lockups should be reduced by 10_000
	totalLockedAmount := k.GetTotalLockupsAmount(ctx)
	require.Equal(t, initialDelegatorBalance.Sub(expectedTierSlashed), totalLockedAmount)

	// insured amount should be equal to insuracePoolBalance (3_000)
	insuredAmount := k.getInsuranceLockupAmount(ctx, delAddr, valAddr)
	require.Equal(t, insuracePoolBalance, insuredAmount)

	// insurance pool delegation should equal to covered amount
	insurancePoolAddr := authtypes.NewModuleAddress(types.InsurancePoolName)
	delegation, err = k.GetStakingKeeper().GetDelegation(ctx, insurancePoolAddr, valAddr)
	require.NoError(t, err)
	require.Equal(t, insuracePoolBalance, delegation.Shares.RoundInt())
}

func TestHandleMissingSignature_MultipleDelegators_PartialCoverage(t *testing.T) {
	k, ctx := setupKeeper(t)

	// slashed amount is greater than insurance pool balance
	slashAmount := math.NewInt(10_000)
	insurancePoolBalance := math.NewInt(1_000)
	initialDelegatorBalance1 := math.NewInt(60_000)
	initialDelegatorBalance2 := math.NewInt(40_000)
	initialDelegatorBalance3 := math.NewInt(100_000)
	initialValidatorBalance := math.ZeroInt()
	totalTierStake := initialDelegatorBalance1.Add(initialDelegatorBalance2)

	delAddr1, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)
	delAddr2, err := sdk.AccAddressFromBech32("source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et")
	require.NoError(t, err)
	delAddr3, err := sdk.AccAddressFromBech32("source18jtkvj0995fy7lggqayg2f5syna92ndq5mkuv4")
	require.NoError(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	initializeDelegator(t, &k, ctx, delAddr1, initialDelegatorBalance1)
	initializeDelegator(t, &k, ctx, delAddr2, initialDelegatorBalance2)
	initializeDelegator(t, &k, ctx, delAddr3, initialDelegatorBalance3)
	initializeValidator(t, k.GetStakingKeeper().(*stakingkeeper.Keeper), ctx, valAddr, initialValidatorBalance)

	mintCoinsToModule(t, &k, ctx, types.InsurancePoolName, insurancePoolBalance)

	// delAddr1 locks via tier module, will be affected
	require.NoError(t, k.Lock(ctx, delAddr1, valAddr, initialDelegatorBalance1))

	// delAddr2 locks via tier module, will also be affected
	require.NoError(t, k.Lock(ctx, delAddr2, valAddr, initialDelegatorBalance2))

	validator, err := k.GetStakingKeeper().GetValidator(ctx, valAddr)
	require.NoError(t, err)

	// delAddr3 delegates normally, will not be affected
	_, err = k.GetStakingKeeper().Delegate(ctx, delAddr3, initialDelegatorBalance3, stakingtypes.Unbonded, validator, true)
	require.NoError(t, err)

	// handle missing signature event
	require.NoError(t, k.handleMissingSignature(ctx, valAddr.String(), slashAmount.String()))

	// tier module stake = 100_000, total stake = 200_000, tier share of slash = 5_000
	expectedSlashed := math.NewInt(5_000)
	require.Equal(t, totalTierStake.Sub(expectedSlashed), k.GetTotalLockupsAmount(ctx))

	// insurance lockup for delAddr1 should be 1_000 * 60_000 / 100_000 = 600
	expectedInsuredAmount1 := math.NewInt(600)
	insuredAmount1 := k.getInsuranceLockupAmount(ctx, delAddr1, valAddr)
	require.Equal(t, expectedInsuredAmount1, insuredAmount1)

	// insurance lockup for delAddr2 should be 1_000 * 40_000 / 100_000 = 400
	expectedInsuredAmount2 := math.NewInt(400)
	insuredAmount2 := k.getInsuranceLockupAmount(ctx, delAddr2, valAddr)
	require.Equal(t, expectedInsuredAmount2, insuredAmount2)

	// no insurance lockup for delAddr3
	insuredAmount3 := k.getInsuranceLockupAmount(ctx, delAddr3, valAddr)
	require.True(t, insuredAmount3.IsZero())

	// total insurance pool delegation should be equal to insurancePoolBalance
	insurancePoolAddr := authtypes.NewModuleAddress(types.InsurancePoolName)
	insuranceDelegation, err := k.GetStakingKeeper().GetDelegation(ctx, insurancePoolAddr, valAddr)
	require.NoError(t, err)
	require.Equal(t, insurancePoolBalance, insuranceDelegation.Shares.RoundInt())
}

func TestHandleMissingSignature_MultipleDelegators_FullCoverage(t *testing.T) {
	k, ctx := setupKeeper(t)

	slashAmount := math.NewInt(10_000)
	insurancePoolBalance := math.NewInt(100_000)
	initialDelegatorBalance1 := math.NewInt(60_000)
	initialDelegatorBalance2 := math.NewInt(40_000)
	initialDelegatorBalance3 := math.NewInt(100_000)
	initialValidatorBalance := math.ZeroInt()
	totalTierStake := initialDelegatorBalance1.Add(initialDelegatorBalance2)

	delAddr1, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)
	delAddr2, err := sdk.AccAddressFromBech32("source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et")
	require.NoError(t, err)
	delAddr3, err := sdk.AccAddressFromBech32("source18jtkvj0995fy7lggqayg2f5syna92ndq5mkuv4")
	require.NoError(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	initializeDelegator(t, &k, ctx, delAddr1, initialDelegatorBalance1)
	initializeDelegator(t, &k, ctx, delAddr2, initialDelegatorBalance2)
	initializeDelegator(t, &k, ctx, delAddr3, initialDelegatorBalance3)
	initializeValidator(t, k.GetStakingKeeper().(*stakingkeeper.Keeper), ctx, valAddr, initialValidatorBalance)

	mintCoinsToModule(t, &k, ctx, types.InsurancePoolName, insurancePoolBalance)

	// delAddr1 locks via tier module, will be affected
	require.NoError(t, k.Lock(ctx, delAddr1, valAddr, initialDelegatorBalance1))

	// delAddr2 locks via tier module, will also be affected
	require.NoError(t, k.Lock(ctx, delAddr2, valAddr, initialDelegatorBalance2))

	validator, err := k.GetStakingKeeper().GetValidator(ctx, valAddr)
	require.NoError(t, err)

	// delAddr3 delegates normally, will not be affected
	_, err = k.GetStakingKeeper().Delegate(ctx, delAddr3, initialDelegatorBalance3, stakingtypes.Unbonded, validator, true)
	require.NoError(t, err)

	// handle missing signature event
	require.NoError(t, k.handleMissingSignature(ctx, valAddr.String(), slashAmount.String()))

	// tier module stake = 100_000, total stake = 200_000, tier share of slash = 5_000
	expectedSlashed := math.NewInt(5_000)
	require.Equal(t, totalTierStake.Sub(expectedSlashed), k.GetTotalLockupsAmount(ctx))

	// insurance lockup for delAddr1 should be 10_000 * 60_000 / 200_000 = 3_000
	expectedInsuredAmount1 := math.NewInt(3_000)
	insuredAmount1 := k.getInsuranceLockupAmount(ctx, delAddr1, valAddr)
	require.Equal(t, expectedInsuredAmount1, insuredAmount1)

	// insurance lockup for delAddr2 should be 10_000 * 40_000 / 200_000 = 2_000
	expectedInsuredAmount2 := math.NewInt(2_000)
	insuredAmount2 := k.getInsuranceLockupAmount(ctx, delAddr2, valAddr)
	require.Equal(t, expectedInsuredAmount2, insuredAmount2)

	// no insurance lockup for delAddr3
	insuredAmount3 := k.getInsuranceLockupAmount(ctx, delAddr3, valAddr)
	require.True(t, insuredAmount3.IsZero())

	// total insurance pool delegation should be equal to expectedSlashed
	insurancePoolAddr := authtypes.NewModuleAddress(types.InsurancePoolName)
	insuranceDelegation, err := k.GetStakingKeeper().GetDelegation(ctx, insurancePoolAddr, valAddr)
	require.NoError(t, err)
	require.Equal(t, expectedSlashed, insuranceDelegation.Shares.RoundInt())
}
