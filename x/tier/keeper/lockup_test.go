package keeper_test

import (
	"errors"
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

	now := time.Now()
	params := k.GetParams(ctx)
	epochDuration := *params.EpochDuration
	creationHeight := int64(10)
	amount := math.NewInt(1000)

	delAddr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.Nil(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.Nil(t, err)

	ctx = ctx.WithBlockHeight(creationHeight).WithBlockTime(now)

	unbondTime := ctx.BlockTime().Add(epochDuration * time.Duration(params.UnlockingEpochs))
	unlockTime := unbondTime

	k.SetLockup(ctx, false, delAddr, valAddr, amount, nil)

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

	lockupAmount := math.NewInt(1000)
	partialSubtractAmount := math.NewInt(500)
	invalidSubtractAmount := math.NewInt(2000)

	delAddr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.Nil(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.Nil(t, err)

	k.AddLockup(ctx, delAddr, valAddr, lockupAmount)

	// subtract a partial amount
	err = k.SubtractLockup(ctx, delAddr, valAddr, partialSubtractAmount)
	require.NoError(t, err)

	lockedAmt := k.GetLockupAmount(ctx, delAddr, valAddr)
	require.Equal(t, partialSubtractAmount, lockedAmt)

	// attempt to subtract more than the locked amount
	err = k.SubtractLockup(ctx, delAddr, valAddr, invalidSubtractAmount)
	require.Error(t, err)

	// subtract the remaining amount
	err = k.SubtractLockup(ctx, delAddr, valAddr, partialSubtractAmount)
	require.NoError(t, err)

	// verify that the lockup has been removed
	lockedAmt = k.GetLockupAmount(ctx, delAddr, valAddr)
	require.True(t, lockedAmt.IsZero(), "remaining lockup amount should be zero")
	require.False(t, k.HasLockup(ctx, delAddr, valAddr), "lockup should be removed")
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
	valAddr2, err := sdk.ValAddressFromBech32("sourcevaloper13fj7t2yptf9k6ad6fv38434znzay4s4pjk0r4f")
	require.Nil(t, err)

	k.SetLockup(ctx, false, delAddr1, valAddr1, amount1, nil)
	k.SetLockup(ctx, false, delAddr2, valAddr2, amount2, nil)

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

	k.SetLockup(ctx, true, delAddr, valAddr, amount, nil)

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
	valAddr2, err := sdk.ValAddressFromBech32("sourcevaloper13fj7t2yptf9k6ad6fv38434znzay4s4pjk0r4f")
	require.Nil(t, err)

	ctx = ctx.WithBlockHeight(1)
	k.SetLockup(ctx, false, delAddr1, valAddr1, math.NewInt(1000), nil)
	k.SetLockup(ctx, false, delAddr2, valAddr2, math.NewInt(500), nil)

	ctx = ctx.WithBlockHeight(2)
	k.SetLockup(ctx, true, delAddr1, valAddr1, math.NewInt(200), nil)

	ctx = ctx.WithBlockHeight(3)
	k.SetLockup(ctx, true, delAddr1, valAddr1, math.NewInt(200), nil)

	ctx = ctx.WithBlockHeight(4)
	k.SetLockup(ctx, true, delAddr1, valAddr1, math.NewInt(200), nil)

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

	err = k.IterateLockups(ctx, false, func(delAddr sdk.AccAddress, valAddr sdk.ValAddress, creationHeight int64, lockup types.Lockup) error {
		return errors.New("not found")
	})
	require.Error(t, err)
}

func TestTotalAmountByAddr(t *testing.T) {
	k, ctx := testutil.SetupKeeper(t)

	delAddr1, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)
	valAddr1, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	delAddr2, err := sdk.AccAddressFromBech32("source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et")
	require.NoError(t, err)
	valAddr2, err := sdk.ValAddressFromBech32("sourcevaloper13fj7t2yptf9k6ad6fv38434znzay4s4pjk0r4f")
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

func TestHasLockup(t *testing.T) {
	k, ctx := testutil.SetupKeeper(t)

	delAddr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	require.False(t, k.HasLockup(ctx, delAddr, valAddr))

	k.AddLockup(ctx, delAddr, valAddr, math.NewInt(100))
	require.True(t, k.HasLockup(ctx, delAddr, valAddr))

	err = k.SubtractLockup(ctx, delAddr, valAddr, math.NewInt(100))
	require.NoError(t, err)
	require.False(t, k.HasLockup(ctx, delAddr, valAddr), "lockup should no longer exist after removing the entire amount")
}

func TestGetUnlockingLockup(t *testing.T) {
	k, ctx := testutil.SetupKeeper(t)

	now := time.Now()
	params := k.GetParams(ctx)
	epochDuration := *params.EpochDuration
	creationHeight := int64(10)
	amount := math.NewInt(300)

	delAddr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	ctx = ctx.WithBlockHeight(creationHeight).WithBlockTime(now)

	unbondTime := ctx.BlockTime().Add(epochDuration * time.Duration(params.UnlockingEpochs))
	unlockTime := unbondTime

	k.SetLockup(ctx, true, delAddr, valAddr, amount, nil)

	found, amt, gotUnbondTime, gotUnlockTime := k.GetUnlockingLockup(ctx, delAddr, valAddr, creationHeight)
	require.True(t, found, "unlocking lockup should be found")
	require.Equal(t, amount, amt, "amount should match the one set")
	require.Equal(t, unbondTime, gotUnbondTime, "unbondTime should match the one set")
	require.Equal(t, unlockTime, gotUnlockTime, "unlockTime should match the one set")

	found, amt, gotUnbondTime, gotUnlockTime = k.GetUnlockingLockup(ctx, delAddr, valAddr, creationHeight+1)
	require.False(t, found, "this unlocking lockup does not exist")
	require.True(t, amt.IsZero(), "amount should be zero")
	require.True(t, gotUnbondTime.IsZero(), "unbond time should be zero")
	require.True(t, gotUnlockTime.IsZero(), "unlock time should be zero")
}

func TestGetLockup(t *testing.T) {
	k, ctx := testutil.SetupKeeper(t)

	now := time.Now()
	creationHeight := int64(10)
	amount := math.NewInt(1000)

	delAddr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	ctx = ctx.WithBlockHeight(creationHeight).WithBlockTime(now)

	params := k.GetParams(ctx)
	unbondTime := ctx.BlockTime().Add(*params.EpochDuration * time.Duration(params.UnlockingEpochs))
	unlockTime := unbondTime

	k.SetLockup(ctx, false, delAddr, valAddr, amount, nil)

	lockup := k.GetLockup(ctx, delAddr, valAddr)

	require.NotNil(t, lockup, "lockup should exist")
	require.Equal(t, delAddr.String(), lockup.DelegatorAddress, "delegator address should match")
	require.Equal(t, valAddr.String(), lockup.ValidatorAddress, "validator address should match")
	require.Equal(t, amount, lockup.Amount, "amount should match")
	require.Equal(t, creationHeight, lockup.CreationHeight, "creation height should match")
	require.Equal(t, unbondTime.UTC(), *lockup.UnbondTime, "unbond time should match")
	require.Equal(t, unlockTime.UTC(), *lockup.UnlockTime, "unlock time should match")

	nonExistentValAddr, err := sdk.ValAddressFromBech32("sourcevaloper13fj7t2yptf9k6ad6fv38434znzay4s4pjk0r4f")
	require.NoError(t, err)

	nonExistentLockup := k.GetLockup(ctx, delAddr, nonExistentValAddr)
	require.Nil(t, nonExistentLockup, "lockup should not exist for the given validator")
}

func TestGetLockups(t *testing.T) {
	k, ctx := testutil.SetupKeeper(t)

	amount1 := math.NewInt(1000)
	amount2 := math.NewInt(500)

	delAddr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)
	valAddr1, err := sdk.ValAddressFromBech32("sourcevaloper13fj7t2yptf9k6ad6fv38434znzay4s4pjk0r4f")
	require.NoError(t, err)
	valAddr2, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	ctx = ctx.WithBlockHeight(10).WithBlockTime(time.Now())
	k.SetLockup(ctx, false, delAddr, valAddr1, amount1, nil)

	ctx = ctx.WithBlockHeight(11).WithBlockTime(time.Now().Add(time.Minute))
	k.SetLockup(ctx, false, delAddr, valAddr2, amount2, nil)

	lockups := k.GetLockups(ctx, delAddr)

	require.Len(t, lockups, 2, "delegator should have 2 lockups")

	require.Equal(t, delAddr.String(), lockups[0].DelegatorAddress)
	require.Equal(t, valAddr1.String(), lockups[0].ValidatorAddress)
	require.Equal(t, amount1, lockups[0].Amount)

	require.Equal(t, delAddr.String(), lockups[1].DelegatorAddress)
	require.Equal(t, valAddr2.String(), lockups[1].ValidatorAddress)
	require.Equal(t, amount2, lockups[1].Amount)
}

func TestSubtractUnlockingLockup(t *testing.T) {
	k, ctx := testutil.SetupKeeper(t)

	unlockingLockupAmount := math.NewInt(1000)
	cancelUnlockAmount := math.NewInt(500)
	cancelUnlockAmount2 := math.NewInt(2000)
	creationHeight := int64(10)

	delAddr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	ctx = ctx.WithBlockHeight(creationHeight)
	k.SetLockup(ctx, true, delAddr, valAddr, unlockingLockupAmount, nil)

	// subtract partial amount
	err = k.SubtractUnlockingLockup(ctx, delAddr, valAddr, creationHeight, cancelUnlockAmount)
	require.NoError(t, err)

	found, lockedAmt, _, _ := k.GetUnlockingLockup(ctx, delAddr, valAddr, creationHeight)
	require.True(t, found)
	require.Equal(t, cancelUnlockAmount, lockedAmt)

	// try to subtract more than the locked amount
	err = k.SubtractUnlockingLockup(ctx, delAddr, valAddr, creationHeight, cancelUnlockAmount2)
	require.Error(t, err)

	// subtract remaining amount
	err = k.SubtractUnlockingLockup(ctx, delAddr, valAddr, creationHeight, cancelUnlockAmount)
	require.NoError(t, err)

	found, lockedAmt, _, _ = k.GetUnlockingLockup(ctx, delAddr, valAddr, creationHeight)
	require.False(t, found)
	require.True(t, lockedAmt.IsZero())
}
