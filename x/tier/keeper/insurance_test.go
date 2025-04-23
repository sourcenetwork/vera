package keeper

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sourcenetwork/sourcehub/x/tier/types"
	"github.com/stretchr/testify/require"
)

func TestGetAllInsuranceLockups(t *testing.T) {
	k, ctx := setupKeeper(t)

	amount := math.NewInt(1000)

	delAddr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	k.setInsuranceLockup(ctx, delAddr, valAddr, amount)

	lockups := k.GetAllInsuranceLockups(ctx)
	require.Len(t, lockups, 1)
	require.Equal(t, delAddr.String(), lockups[0].DelegatorAddress)
	require.Equal(t, valAddr.String(), lockups[0].ValidatorAddress)
	require.Equal(t, amount, lockups[0].Amount)
}

func TestAddInsuranceLockup_New(t *testing.T) {
	k, ctx := setupKeeper(t)

	amount := math.NewInt(500)

	delAddr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	k.AddInsuranceLockup(ctx, delAddr, valAddr, amount)

	insuredAmt := k.getInsuranceLockupAmount(ctx, delAddr, valAddr)
	require.Equal(t, amount, insuredAmt)
}

func TestAddInsuranceLockup_Append(t *testing.T) {
	k, ctx := setupKeeper(t)

	amount := math.NewInt(800)

	delAddr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	k.setInsuranceLockup(ctx, delAddr, valAddr, math.NewInt(500))
	k.AddInsuranceLockup(ctx, delAddr, valAddr, math.NewInt(300))

	insuredAmt := k.getInsuranceLockupAmount(ctx, delAddr, valAddr)
	require.Equal(t, amount, insuredAmt)
}

func TestGetInsuranceLockup(t *testing.T) {
	k, ctx := setupKeeper(t)

	amount := math.NewInt(1000)

	delAddr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	k.setInsuranceLockup(ctx, delAddr, valAddr, amount)

	lockup := k.getInsuranceLockup(ctx, delAddr, valAddr)
	require.NotNil(t, lockup)
	require.Equal(t, amount, lockup.Amount)
}

func TestGetInsuranceLockup_NotFound(t *testing.T) {
	k, ctx := setupKeeper(t)

	delAddr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	lockup := k.getInsuranceLockup(ctx, delAddr, valAddr)
	require.Nil(t, lockup)
}

func TestGetInsuranceLockupAmount_ZeroIfNotFound(t *testing.T) {
	k, ctx := setupKeeper(t)

	delAddr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	insuredAmt := k.getInsuranceLockupAmount(ctx, delAddr, valAddr)
	require.True(t, insuredAmt.IsZero())
}

func TestSetInsuranceLockup(t *testing.T) {
	k, ctx := setupKeeper(t)

	amount := math.NewInt(1500)

	delAddr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	k.setInsuranceLockup(ctx, delAddr, valAddr, amount)

	insuredAmt := k.getInsuranceLockupAmount(ctx, delAddr, valAddr)
	require.Equal(t, amount, insuredAmt)
}

func TestRemoveInsuranceLockup(t *testing.T) {
	k, ctx := setupKeeper(t)

	amount := math.NewInt(1000)

	delAddr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	k.setInsuranceLockup(ctx, delAddr, valAddr, amount)
	k.removeInsuranceLockup(ctx, delAddr, valAddr)

	lockup := k.getInsuranceLockup(ctx, delAddr, valAddr)
	require.Nil(t, lockup)
}

func TestTotalInsuredAmountByAddr(t *testing.T) {
	k, ctx := setupKeeper(t)

	amount1 := math.NewInt(300)
	amount2 := math.NewInt(500)
	amount3 := math.NewInt(1000)

	delAddr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)
	valAddr1, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)
	valAddr2, err := sdk.ValAddressFromBech32("sourcevaloper13fj7t2yptf9k6ad6fv38434znzay4s4pjk0r4f")
	require.NoError(t, err)

	k.setInsuranceLockup(ctx, delAddr, valAddr1, amount1)
	k.setInsuranceLockup(ctx, delAddr, valAddr2, amount2)
	k.AddInsuranceLockup(ctx, delAddr, valAddr2, amount3)

	totalAmount := k.totalInsuredAmountByAddr(ctx, delAddr)
	require.Equal(t, amount1.Add(amount2).Add(amount3), totalAmount)
}

func TestMustIterateInsuranceLockups(t *testing.T) {
	k, ctx := setupKeeper(t)

	amount := math.NewInt(5000)

	delAddr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	k.setInsuranceLockup(ctx, delAddr, valAddr, amount)

	count := 0
	k.mustIterateInsuranceLockups(ctx, func(d sdk.AccAddress, v sdk.ValAddress, lockup types.Lockup) {
		count++
		require.Equal(t, delAddr.String(), lockup.DelegatorAddress)
		require.Equal(t, valAddr.String(), lockup.ValidatorAddress)
		require.Equal(t, amount, lockup.Amount)
	})
	require.Equal(t, 1, count)
}
