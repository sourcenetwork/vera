package keeper

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	appparams "github.com/sourcenetwork/sourcehub/app/params"
	epochstypes "github.com/sourcenetwork/sourcehub/x/epochs/types"
	"github.com/sourcenetwork/sourcehub/x/tier/types"
	"github.com/stretchr/testify/require"
)

func TestBeginBlocker(t *testing.T) {
	k, ctx := setupKeeper(t)

	amount := math.NewInt(10_000_000_000_000)

	delAddr, err := sdk.AccAddressFromBech32("source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et")
	require.NoError(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	initialDelegatorBalance := math.NewInt(20_000_000_000_000)
	initializeDelegator(t, &k, ctx, delAddr, initialDelegatorBalance)
	initialValidatorBalance := math.NewInt(10_000_000_000_000)
	initializeValidator(t, k.GetStakingKeeper().(*stakingkeeper.Keeper), ctx, valAddr, initialValidatorBalance)

	// set initial block height and time
	ctx = ctx.WithBlockHeight(1).WithBlockTime(time.Now())

	// lock valid amount
	err = k.Lock(ctx, delAddr, valAddr, amount)
	require.NoError(t, err)

	// verify that lockup was added
	lockedAmt := k.GetLockupAmount(ctx, delAddr, valAddr)
	require.Equal(t, amount, lockedAmt)

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
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	initialDelegatorBalance := math.NewInt(200_000)
	initialValidatorBalance := math.NewInt(1_000_000)
	insurancePoolBalance := math.NewInt(500_000)

	initializeDelegator(t, &k, ctx, delAddr, initialDelegatorBalance)
	initializeValidator(t, k.GetStakingKeeper().(*stakingkeeper.Keeper), ctx, valAddr, initialValidatorBalance)
	mintCoinsToModule(t, &k, ctx, types.InsurancePoolName, insurancePoolBalance)

	epoch := epochstypes.EpochInfo{
		Identifier:            types.EpochIdentifier,
		CurrentEpoch:          1,
		CurrentEpochStartTime: ctx.BlockTime().Add(-5 * time.Minute),
		Duration:              5 * time.Minute,
	}
	k.GetEpochsKeeper().SetEpochInfo(ctx, epoch)

	err = k.Lock(ctx, delAddr, valAddr, initialDelegatorBalance)
	require.NoError(t, err)

	balance := k.GetBankKeeper().GetBalance(ctx, insurancePoolAddr, appparams.DefaultBondDenom)
	require.Equal(t, insurancePoolBalance, balance.Amount)

	// Emit missing_signature slashing event (must trigger recover)
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"slash",
		sdk.NewAttribute("address", valAddr.String()),
		sdk.NewAttribute("reason", slashingtypes.AttributeValueMissingSignature),
		sdk.NewAttribute("burned", "100000"),
	))

	err = k.handleSlashingEvents(ctx)
	require.NoError(t, err)

	// Burned tier module share is 100_000 / (1_200_000 / 200_000) = 16_667
	expectedRemainingBalance := insurancePoolBalance.Sub(math.NewInt(16_667))
	newBalance := k.GetBankKeeper().GetBalance(ctx, insurancePoolAddr, appparams.DefaultBondDenom)
	require.Equal(t, expectedRemainingBalance, newBalance.Amount)

	tierDelegation, err := k.GetStakingKeeper().GetDelegation(ctx, tierModuleAddr, valAddr)
	require.NoError(t, err)
	require.False(t, tierDelegation.Shares.IsZero())

	// Reset event manager and emit double_sign slashing event (no recover)
	ctx = ctx.WithBlockHeight(2).WithEventManager(sdk.NewEventManager())

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"slash",
		sdk.NewAttribute("address", valAddr.String()),
		sdk.NewAttribute("reason", slashingtypes.AttributeValueDoubleSign),
		sdk.NewAttribute("burned", "200000"),
	))

	err = k.handleSlashingEvents(ctx)
	require.NoError(t, err)

	newBalance = k.GetBankKeeper().GetBalance(ctx, insurancePoolAddr, appparams.DefaultBondDenom)
	require.Equal(t, expectedRemainingBalance, newBalance.Amount)

	tierDelegationAfter, err := k.GetStakingKeeper().GetDelegation(ctx, tierModuleAddr, valAddr)
	require.NoError(t, err)
	require.Equal(t, tierDelegation.Shares, tierDelegationAfter.Shares)
}
