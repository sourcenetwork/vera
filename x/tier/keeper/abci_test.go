package keeper

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	appparams "github.com/sourcenetwork/sourcehub/app/params"
	"github.com/sourcenetwork/sourcehub/x/tier/types"
	"github.com/stretchr/testify/require"
)

func TestBeginBlocker(t *testing.T) {
	k, ctx := setupKeeper(t)

	tierModuleAddr := authtypes.NewModuleAddress(types.ModuleName)
	insurancePoolAddr := authtypes.NewModuleAddress(types.InsurancePoolName)

	insurancePoolBalance := math.NewInt(500_000)

	delAddr, err := sdk.AccAddressFromBech32("source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et")
	require.NoError(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	initialDelegatorBalance := math.NewInt(20_000_000_000_000)
	initializeDelegator(t, &k, ctx, delAddr, initialDelegatorBalance)
	initialValidatorBalance := math.NewInt(10_000_000_000_000)
	initializeValidator(t, k.GetStakingKeeper().(*stakingkeeper.Keeper), ctx, valAddr, initialValidatorBalance)
	mintCoinsToModule(t, &k, ctx, types.InsurancePoolName, insurancePoolBalance)

	// set initial block height and time
	ctx = ctx.WithBlockHeight(1).WithBlockTime(time.Now())

	tierDelegation, err := k.GetStakingKeeper().GetDelegation(ctx, tierModuleAddr, valAddr)
	require.Error(t, err)

	// lock valid amount
	err = k.Lock(ctx, delAddr, valAddr, initialDelegatorBalance)
	require.NoError(t, err)

	tierDelegation, err = k.GetStakingKeeper().GetDelegation(ctx, tierModuleAddr, valAddr)
	require.NoError(t, err)
	require.Equal(t, math.LegacyNewDecFromInt(initialDelegatorBalance), tierDelegation.Shares)

	balance := k.GetBankKeeper().GetBalance(ctx, insurancePoolAddr, appparams.DefaultBondDenom)
	require.Equal(t, insurancePoolBalance, balance.Amount)

	// verify that lockup was added
	lockedAmt := k.GetLockupAmount(ctx, delAddr, valAddr)
	require.Equal(t, initialDelegatorBalance, lockedAmt)

	// advance to block at height 1000
	ctx = ctx.WithBlockHeight(1000).WithBlockTime(time.Now().Add(time.Hour))

	err = k.BeginBlocker(ctx)
	require.NoError(t, err)
}

func TestHandleSlashingEvents(t *testing.T) {
	k, ctx := setupKeeper(t)

	tierModuleAddr := authtypes.NewModuleAddress(types.ModuleName)
	insurancePoolAddr := authtypes.NewModuleAddress(types.InsurancePoolName)

	delAddr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)
	delAddr2, err := sdk.AccAddressFromBech32("source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et")
	require.NoError(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	initialDelegatorBalance := math.NewInt(200_000)
	initialDelegatorBalance2 := math.NewInt(800_000)
	initialValidatorBalance := math.NewInt(1_000_000)
	insurancePoolBalance := math.NewInt(500_000)
	missingSignatureSlashAmount := math.NewInt(100_000)
	doubleSignSlashAmount := math.NewInt(200_000)

	initializeDelegator(t, &k, ctx, delAddr, initialDelegatorBalance)
	initializeDelegator(t, &k, ctx, delAddr2, initialDelegatorBalance2)
	initializeValidator(t, k.GetStakingKeeper().(*stakingkeeper.Keeper), ctx, valAddr, initialValidatorBalance)
	mintCoinsToModule(t, &k, ctx, types.InsurancePoolName, insurancePoolBalance)

	validator, err := k.GetStakingKeeper().GetValidator(ctx, valAddr)
	require.NoError(t, err)
	require.Equal(t, initialValidatorBalance, validator.Tokens)

	_, err = k.stakingKeeper.Delegate(ctx, delAddr2, initialDelegatorBalance2, stakingtypes.Unbonded, validator, true)

	err = k.Lock(ctx, delAddr, valAddr, initialDelegatorBalance)
	require.NoError(t, err)

	balance := k.GetBankKeeper().GetBalance(ctx, insurancePoolAddr, appparams.DefaultBondDenom)
	require.Equal(t, insurancePoolBalance, balance.Amount)

	ctx = ctx.WithBlockHeight(10).WithBlockTime(time.Now().Add(time.Minute))

	expectedTotalStake := initialValidatorBalance.Add(initialDelegatorBalance).Add(initialDelegatorBalance2)
	validator, err = k.GetStakingKeeper().GetValidator(ctx, valAddr)
	require.NoError(t, err)
	require.Equal(t, expectedTotalStake, validator.Tokens)

	// emit missing_signature slashing event
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"slash",
		sdk.NewAttribute("address", valAddr.String()),
		sdk.NewAttribute("reason", slashingtypes.AttributeValueMissingSignature),
		sdk.NewAttribute("burned", missingSignatureSlashAmount.String()),
	))

	// total validator stake remains unchanged since we just emit the event without burning tokens
	validator, err = k.GetStakingKeeper().GetValidator(ctx, valAddr)
	require.NoError(t, err)
	require.Equal(t, expectedTotalStake, validator.Tokens)

	// tier stake should be equal to initial delegator balance
	tierDelegation, err := k.GetStakingKeeper().GetDelegation(ctx, tierModuleAddr, valAddr)
	require.NoError(t, err)
	tierStake := validator.TokensFromSharesTruncated(tierDelegation.Shares)
	require.Equal(t, initialDelegatorBalance, tierStake.RoundInt())

	// no insurance pool delegation at this point
	_, err = k.GetStakingKeeper().GetDelegation(ctx, insurancePoolAddr, valAddr)
	require.Error(t, err)

	// total lockups amount should be equal to initial delegator balance at this point
	totalLockupsAmount := k.GetTotalLockupsAmount(ctx)
	require.Equal(t, initialDelegatorBalance, totalLockupsAmount)

	// handle missing_signature event (cover slashed tier module stake)
	k.handleSlashingEvents(ctx)

	// slashed tier module amount is 200_000 / (1_000_000 + 800_000 + 200_000) * 100_000 = 10_000
	missingSigTierSlashedAmt := math.NewInt(10_000)

	// verify insurance pool balance
	expectedRemainingInsurancePoolBalance := insurancePoolBalance.Sub(missingSigTierSlashedAmt)
	newInsurancePoolBalance := k.GetBankKeeper().GetBalance(ctx, insurancePoolAddr, appparams.DefaultBondDenom)
	require.Equal(t, expectedRemainingInsurancePoolBalance, newInsurancePoolBalance.Amount)

	// slashed tier module amount is delegated back to the slashed validator
	validator, err = k.GetStakingKeeper().GetValidator(ctx, valAddr)
	require.NoError(t, err)
	require.Equal(t, expectedTotalStake.Add(missingSigTierSlashedAmt), validator.Tokens)

	// tier stake should be equal to initial delegator balance
	tierDelegation, err = k.GetStakingKeeper().GetDelegation(ctx, tierModuleAddr, valAddr)
	require.NoError(t, err)
	tierStake = validator.TokensFromSharesTruncated(tierDelegation.Shares)
	require.Equal(t, initialDelegatorBalance, tierStake.RoundInt())

	// insurance pool should have delegation equal to the slashed amount
	insurancePoolDelegation, err := k.GetStakingKeeper().GetDelegation(ctx, insurancePoolAddr, valAddr)
	require.NoError(t, err)
	insurancePoolStake := validator.TokensFromSharesTruncated(insurancePoolDelegation.Shares)
	require.Equal(t, missingSigTierSlashedAmt, insurancePoolStake.RoundInt())

	// total lockups amount should be reduced by the slashed amount
	totalLockupsAmount = k.GetTotalLockupsAmount(ctx)
	require.Equal(t, initialDelegatorBalance.Sub(missingSigTierSlashedAmt), totalLockupsAmount)

	// insured amount should be equal to the slashed amount since it had enough balance
	insuredAmount := k.getInsuranceLockupAmount(ctx, delAddr, valAddr)
	require.Equal(t, missingSigTierSlashedAmt, insuredAmount)

	// reset event manager
	ctx = ctx.WithBlockHeight(2).WithEventManager(sdk.NewEventManager())

	// and emit double_sign slashing event
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"slash",
		sdk.NewAttribute("address", valAddr.String()),
		sdk.NewAttribute("reason", slashingtypes.AttributeValueDoubleSign),
		sdk.NewAttribute("burned", doubleSignSlashAmount.String()),
	))

	// handle double_sign event (no insurance)
	k.handleSlashingEvents(ctx)

	// slashed tier module amount is 190_000 / (1_000_000 + 800_000 + 200_000 + 10_000) * 200_000 = 18906
	doubleSignSlashedAmount := math.NewInt(18_906)

	// verify insurance pool balance
	newInsurancePoolBalance = k.GetBankKeeper().GetBalance(ctx, insurancePoolAddr, appparams.DefaultBondDenom)
	require.Equal(t, expectedRemainingInsurancePoolBalance, newInsurancePoolBalance.Amount)

	// tier stake should be equal to initial delegator balance
	tierDelegation, err = k.GetStakingKeeper().GetDelegation(ctx, tierModuleAddr, valAddr)
	require.NoError(t, err)
	tierStake = validator.TokensFromSharesTruncated(tierDelegation.Shares)
	require.Equal(t, initialDelegatorBalance, tierStake.RoundInt())

	// total validator stake remains the same as it was after missing_signature event
	validator, err = k.GetStakingKeeper().GetValidator(ctx, valAddr)
	require.NoError(t, err)
	require.Equal(t, expectedTotalStake.Add(missingSigTierSlashedAmt), validator.Tokens)

	// total lockups amount should be reduced by the slashed amount
	totalLockupsAmount = k.GetTotalLockupsAmount(ctx)
	require.Equal(t, initialDelegatorBalance.Sub(missingSigTierSlashedAmt).Sub(doubleSignSlashedAmount), totalLockupsAmount)

	// insured amount should not change because double_sign events are not covered
	insuredAmount = k.getInsuranceLockupAmount(ctx, delAddr, valAddr)
	require.Equal(t, missingSigTierSlashedAmt, insuredAmount)
}
