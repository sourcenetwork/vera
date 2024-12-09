package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sourcenetwork/sourcehub/app"
	testutil "github.com/sourcenetwork/sourcehub/testutil"
	"github.com/sourcenetwork/sourcehub/x/tier/types"
	"github.com/stretchr/testify/require"
)

func init() {
	app.SetConfig(true)
}

func TestSetAndGetLockup(t *testing.T) {
	k, ctx := testutil.SetupKeeper(t)

	amount := math.NewInt(1000)
	creationHeight := int64(10)
	unbondTime := time.Now().Add(1 * time.Hour)
	unlockTime := time.Now().Add(2 * time.Hour)

	delAddr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.Nil(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.Nil(t, err)

	k.SetLockup(ctx, false, delAddr, valAddr, amount, creationHeight, &unbondTime, &unlockTime)

	store := k.GetAllLockups(ctx)
	require.Len(t, store, 1)

	lockup := store[0]
	require.Equal(t, delAddr.String(), lockup.DelegatorAddress)
	require.Equal(t, valAddr.String(), lockup.ValidatorAddress)
	require.Equal(t, amount, lockup.Amount)
	require.Equal(t, creationHeight, lockup.CreationHeight)
	require.Equal(t, unbondTime.UTC(), *lockup.UnbondTime)
	require.Equal(t, unlockTime.UTC(), *lockup.UnlockTime)
}

func TestAddLockup(t *testing.T) {
	k, ctx := testutil.SetupKeeper(t)

	amount := math.NewInt(500)

	delAddr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.Nil(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.Nil(t, err)

	k.AddLockup(ctx, delAddr, valAddr, amount)

	lockup := k.GetLockupAmount(ctx, delAddr, valAddr)
	require.Equal(t, amount, lockup)
}

func TestSubtractLockup(t *testing.T) {
	k, ctx := testutil.SetupKeeper(t)

	amount := math.NewInt(1000)

	delAddr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.Nil(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.Nil(t, err)

	k.AddLockup(ctx, delAddr, valAddr, amount)

	err = k.SubtractLockup(ctx, delAddr, valAddr, math.NewInt(500))
	require.NoError(t, err)

	lockup := k.GetLockupAmount(ctx, delAddr, valAddr)
	require.Equal(t, math.NewInt(500), lockup)
}

func TestGetAllLockups(t *testing.T) {
	k, ctx := testutil.SetupKeeper(t)

	amount1 := math.NewInt(1000)
	amount2 := math.NewInt(500)

	delAddr1, err := sdk.AccAddressFromBech32("source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et")
	require.Nil(t, err)
	valAddr1, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.Nil(t, err)

	delAddr2, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.Nil(t, err)
	valAddr2, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.Nil(t, err)

	k.SetLockup(ctx, false, delAddr1, valAddr1, amount1, 1, nil, nil)
	k.SetLockup(ctx, false, delAddr2, valAddr2, amount2, 2, nil, nil)

	lockups := k.GetAllLockups(ctx)
	require.Len(t, lockups, 2)

	require.Equal(t, delAddr1.String(), lockups[0].DelegatorAddress)
	require.Equal(t, valAddr1.String(), lockups[0].ValidatorAddress)
	require.Equal(t, delAddr2.String(), lockups[1].DelegatorAddress)
	require.Equal(t, valAddr2.String(), lockups[1].ValidatorAddress)
}

func TestMustIterateLockups(t *testing.T) {
	k, ctx := testutil.SetupKeeper(t)

	amount := math.NewInt(1000)

	delAddr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.Nil(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.Nil(t, err)

	k.AddLockup(ctx, delAddr, valAddr, amount)

	count := 0
	k.MustIterateLockups(ctx, func(delAddr sdk.AccAddress, valAddr sdk.ValAddress, lockup types.Lockup) {
		require.Equal(t, "source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9", delAddr.String())
		require.Equal(t, "sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm", valAddr.String())
		require.Equal(t, amount, lockup.Amount)
		count++
	})

	require.Equal(t, 1, count)
}

func TestMustIterateUnlockingLockups(t *testing.T) {
	k, ctx := testutil.SetupKeeper(t)

	amount := math.NewInt(1000)

	delAddr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.Nil(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.Nil(t, err)

	unbondTime := time.Now().Add(24 * time.Hour)
	unlockTime := time.Now().Add(48 * time.Hour)

	k.SetLockup(ctx, true, delAddr, valAddr, amount, 1, &unbondTime, &unlockTime)

	count := 0
	k.MustIterateUnlockingLockups(ctx, func(delAddr sdk.AccAddress, valAddr sdk.ValAddress, creationHeight int64, lockup types.Lockup) {
		require.Equal(t, "source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9", delAddr.String())
		require.Equal(t, "sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm", valAddr.String())
		require.Equal(t, creationHeight, lockup.CreationHeight)
		require.Equal(t, amount, lockup.Amount)
		count++
	})

	require.Equal(t, 1, count)
}

func TestIterateLockups(t *testing.T) {
	k, ctx := testutil.SetupKeeper(t)

	delAddr1, err := sdk.AccAddressFromBech32("source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et")
	require.Nil(t, err)
	valAddr1, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.Nil(t, err)

	delAddr2, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.Nil(t, err)
	valAddr2, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.Nil(t, err)

	k.SetLockup(ctx, false, delAddr1, valAddr1, math.NewInt(1000), 1, nil, nil)
	k.SetLockup(ctx, false, delAddr2, valAddr2, math.NewInt(500), 2, nil, nil)

	unbondTime := time.Now().Add(24 * time.Hour)
	unlockTime := time.Now().Add(48 * time.Hour)
	k.SetLockup(ctx, true, delAddr1, valAddr1, math.NewInt(200), 3, &unbondTime, &unlockTime)
	k.SetLockup(ctx, true, delAddr1, valAddr1, math.NewInt(200), 4, &unbondTime, &unbondTime)
	k.SetLockup(ctx, true, delAddr1, valAddr1, math.NewInt(200), 5, &unbondTime, &unbondTime)

	lockupsCount := 0
	err = k.IterateLockups(ctx, false, func(delAddr sdk.AccAddress, valAddr sdk.ValAddress, creationHeight int64, lockup types.Lockup) error {
		require.NotNil(t, delAddr)
		require.NotNil(t, valAddr)
		require.True(t, lockup.Amount.IsPositive())
		lockupsCount++
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, 2, lockupsCount)

	unlockingLockupsCount := 0
	err = k.IterateLockups(ctx, true, func(delAddr sdk.AccAddress, valAddr sdk.ValAddress, creationHeight int64, lockup types.Lockup) error {
		require.NotNil(t, delAddr)
		require.NotNil(t, valAddr)
		require.True(t, lockup.Amount.IsPositive())
		require.NotNil(t, lockup.UnbondTime)
		require.NotNil(t, lockup.UnlockTime)
		unlockingLockupsCount++
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, 3, unlockingLockupsCount)
}

func TestTotalAmountByAddr(t *testing.T) {
	k, ctx := testutil.SetupKeeper(t)

	delAddr1, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)
	valAddr1, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	delAddr2, err := sdk.AccAddressFromBech32("source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et")
	require.NoError(t, err)
	valAddr2, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	k.AddLockup(ctx, delAddr1, valAddr1, math.NewInt(1000))
	k.AddLockup(ctx, delAddr1, valAddr1, math.NewInt(500))
	k.AddLockup(ctx, delAddr2, valAddr2, math.NewInt(700))

	totalDel1 := k.TotalAmountByAddr(ctx, delAddr1)
	require.Equal(t, math.NewInt(1500), totalDel1, "delAddr1 should have a total of 1500")

	totalDel2 := k.TotalAmountByAddr(ctx, delAddr2)
	require.Equal(t, math.NewInt(700), totalDel2, "delAddr2 should have a total of 700")

	delAddr3, err := sdk.AccAddressFromBech32("source1n34fvpteuanu2nx2a4hql4jvcrcnal3gsrjppy")
	require.NoError(t, err)
	totalDel3 := k.TotalAmountByAddr(ctx, delAddr3)
	require.True(t, totalDel3.IsZero(), "delAddr3 should have no lockups")
}
