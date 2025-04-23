package keeper

import (
	"errors"
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sourcenetwork/sourcehub/x/tier/types"
	"github.com/stretchr/testify/require"
)

func TestSetAndGetLockup(t *testing.T) {
	k, ctx := setupKeeper(t)

	now := time.Now()
	creationHeight := int64(10)
	amount := math.NewInt(1000)

	delAddr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	ctx = ctx.WithBlockHeight(creationHeight).WithBlockTime(now)

	k.setLockup(ctx, delAddr, valAddr, amount)

	store := k.GetAllLockups(ctx)
	require.Len(t, store, 1)

	lockup := store[0]
	require.Equal(t, delAddr.String(), lockup.DelegatorAddress)
	require.Equal(t, valAddr.String(), lockup.ValidatorAddress)
	require.Equal(t, amount, lockup.Amount)
}

func TestAddLockup(t *testing.T) {
	k, ctx := setupKeeper(t)

	amount := math.NewInt(500)

	delAddr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	err = k.AddLockup(ctx, delAddr, valAddr, amount)
	require.NoError(t, err)

	lockedAmt := k.GetLockupAmount(ctx, delAddr, valAddr)
	require.Equal(t, amount, lockedAmt)
}

func TestSubtractLockup(t *testing.T) {
	k, ctx := setupKeeper(t)

	lockupAmount := math.NewInt(1000)
	partialSubtractAmount := math.NewInt(500)
	invalidSubtractAmount := math.NewInt(2000)

	delAddr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	err = k.AddLockup(ctx, delAddr, valAddr, lockupAmount)
	require.NoError(t, err)

	// subtract a partial amount
	err = k.subtractLockup(ctx, delAddr, valAddr, partialSubtractAmount)
	require.NoError(t, err)

	lockedAmt := k.GetLockupAmount(ctx, delAddr, valAddr)
	require.Equal(t, partialSubtractAmount, lockedAmt)

	// attempt to subtract more than the locked amount
	err = k.subtractLockup(ctx, delAddr, valAddr, invalidSubtractAmount)
	require.Error(t, err)

	// subtract the remaining amount
	err = k.subtractLockup(ctx, delAddr, valAddr, partialSubtractAmount)
	require.NoError(t, err)

	// verify that the lockup has been removed
	lockedAmt = k.GetLockupAmount(ctx, delAddr, valAddr)
	require.True(t, lockedAmt.IsZero(), "remaining lockup amount should be zero")
	require.Nil(t, k.GetLockup(ctx, delAddr, valAddr), "lockup should be removed")
}

func TestGetAllLockups(t *testing.T) {
	k, ctx := setupKeeper(t)

	amount1 := math.NewInt(1000)
	amount2 := math.NewInt(500)

	delAddr1, err := sdk.AccAddressFromBech32("source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et")
	require.NoError(t, err)
	valAddr1, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	delAddr2, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)
	valAddr2, err := sdk.ValAddressFromBech32("sourcevaloper13fj7t2yptf9k6ad6fv38434znzay4s4pjk0r4f")
	require.NoError(t, err)

	k.setLockup(ctx, delAddr1, valAddr1, amount1)
	k.setLockup(ctx, delAddr2, valAddr2, amount2)

	lockups := k.GetAllLockups(ctx)
	require.Len(t, lockups, 2)

	require.Equal(t, delAddr1.String(), lockups[0].DelegatorAddress)
	require.Equal(t, valAddr1.String(), lockups[0].ValidatorAddress)
	require.Equal(t, amount1, lockups[0].Amount)
	require.Equal(t, delAddr2.String(), lockups[1].DelegatorAddress)
	require.Equal(t, valAddr2.String(), lockups[1].ValidatorAddress)
	require.Equal(t, amount2, lockups[1].Amount)
}

func TestGetAllUnlockingLockups(t *testing.T) {
	k, ctx := setupKeeper(t)

	creationHeight1 := int64(1)
	creationHeight2 := int64(2)
	timestamp1 := time.Date(2006, time.January, 2, 15, 4, 5, 1, time.UTC)
	timestamp2 := time.Date(2006, time.January, 2, 15, 4, 5, 2, time.UTC)
	amount1 := math.NewInt(1000)
	amount2 := math.NewInt(500)

	delAddr1, err := sdk.AccAddressFromBech32("source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et")
	require.NoError(t, err)
	valAddr1, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	delAddr2, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)
	valAddr2, err := sdk.ValAddressFromBech32("sourcevaloper13fj7t2yptf9k6ad6fv38434znzay4s4pjk0r4f")
	require.NoError(t, err)

	k.SetUnlockingLockup(ctx, delAddr1, valAddr1, creationHeight1, amount1, timestamp1, timestamp1)
	k.SetUnlockingLockup(ctx, delAddr2, valAddr2, creationHeight2, amount2, timestamp2, timestamp2)

	unlockingLockups := k.GetAllUnlockingLockups(ctx)
	require.Len(t, unlockingLockups, 2)

	require.Equal(t, delAddr1.String(), unlockingLockups[0].DelegatorAddress)
	require.Equal(t, valAddr1.String(), unlockingLockups[0].ValidatorAddress)
	require.Equal(t, creationHeight1, unlockingLockups[0].CreationHeight)
	require.Equal(t, amount1, unlockingLockups[0].Amount)
	require.Equal(t, timestamp1, unlockingLockups[0].CompletionTime)
	require.Equal(t, timestamp1, unlockingLockups[0].UnlockTime)

	require.Equal(t, delAddr2.String(), unlockingLockups[1].DelegatorAddress)
	require.Equal(t, valAddr2.String(), unlockingLockups[1].ValidatorAddress)
	require.Equal(t, creationHeight2, unlockingLockups[1].CreationHeight)
	require.Equal(t, amount2, unlockingLockups[1].Amount)
	require.Equal(t, timestamp2, unlockingLockups[1].CompletionTime)
	require.Equal(t, timestamp2, unlockingLockups[1].UnlockTime)
}

func TestMustIterateLockups(t *testing.T) {
	k, ctx := setupKeeper(t)

	amount := math.NewInt(1000)

	delAddr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	err = k.AddLockup(ctx, delAddr, valAddr, amount)
	require.NoError(t, err)

	count := 0
	k.mustIterateLockups(ctx, func(delAddr sdk.AccAddress, valAddr sdk.ValAddress, lockup types.Lockup) {
		require.Equal(t, delAddr.String(), lockup.DelegatorAddress)
		require.Equal(t, valAddr.String(), lockup.ValidatorAddress)
		require.Equal(t, amount, lockup.Amount)
		count++
	})

	require.Equal(t, 1, count)
}

func TestMustIterateUnlockingLockups(t *testing.T) {
	k, ctx := setupKeeper(t)

	amount := math.NewInt(1000)

	delAddr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	k.SetUnlockingLockup(ctx, delAddr, valAddr, 1, amount, time.Time{}, time.Time{})

	count := 0
	k.mustIterateUnlockingLockups(ctx, func(delAddr sdk.AccAddress, valAddr sdk.ValAddress, creationHeight int64, lockup types.UnlockingLockup) {
		require.Equal(t, "source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9", delAddr.String())
		require.Equal(t, "sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm", valAddr.String())
		require.Equal(t, creationHeight, lockup.CreationHeight)
		require.Equal(t, amount, lockup.Amount)
		count++
	})

	require.Equal(t, 1, count)
}

func TestIterateUnlockingLockups(t *testing.T) {
	k, ctx := setupKeeper(t)

	timestamp1 := time.Date(2006, time.January, 2, 15, 4, 5, 1, time.UTC)
	timestamp2 := time.Date(2006, time.January, 2, 15, 4, 5, 2, time.UTC)
	timestamp3 := time.Date(2006, time.January, 2, 15, 4, 5, 3, time.UTC)
	timestamp4 := time.Date(2006, time.January, 2, 15, 4, 5, 4, time.UTC)
	timestamp5 := time.Date(2006, time.January, 2, 15, 4, 5, 5, time.UTC)

	delAddr1, err := sdk.AccAddressFromBech32("source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et")
	require.NoError(t, err)
	valAddr1, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	delAddr2, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)
	valAddr2, err := sdk.ValAddressFromBech32("sourcevaloper13fj7t2yptf9k6ad6fv38434znzay4s4pjk0r4f")
	require.NoError(t, err)

	ctx = ctx.WithBlockHeight(1)
	k.SetUnlockingLockup(ctx, delAddr1, valAddr1, ctx.BlockHeight(), math.NewInt(1000), timestamp1, timestamp1)
	k.SetUnlockingLockup(ctx, delAddr2, valAddr2, ctx.BlockHeight(), math.NewInt(500), timestamp2, timestamp2)

	ctx = ctx.WithBlockHeight(2)
	k.SetUnlockingLockup(ctx, delAddr1, valAddr1, ctx.BlockHeight(), math.NewInt(200), timestamp3, timestamp3)

	ctx = ctx.WithBlockHeight(3)
	k.SetUnlockingLockup(ctx, delAddr1, valAddr1, ctx.BlockHeight(), math.NewInt(200), timestamp4, timestamp4)

	ctx = ctx.WithBlockHeight(4)
	k.SetUnlockingLockup(ctx, delAddr1, valAddr1, ctx.BlockHeight(), math.NewInt(200), timestamp5, timestamp5)

	unlockingLockupsCount := 0
	err = k.iterateUnlockingLockups(ctx, func(delAddr sdk.AccAddress, valAddr sdk.ValAddress,
		creationHeight int64, unlockingLockup types.UnlockingLockup) error {
		require.NotNil(t, delAddr)
		require.NotNil(t, valAddr)
		require.True(t, unlockingLockup.Amount.IsPositive())
		require.NotZero(t, unlockingLockup.CompletionTime)
		require.NotZero(t, unlockingLockup.UnlockTime)
		unlockingLockupsCount++
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, 5, unlockingLockupsCount)

	err = k.iterateUnlockingLockups(ctx, func(delAddr sdk.AccAddress, valAddr sdk.ValAddress,
		creationHeight int64, lockup types.UnlockingLockup) error {
		return errors.New("not found")
	})
	require.Error(t, err)
}

func TestTotalAmountByAddr(t *testing.T) {
	k, ctx := setupKeeper(t)

	delAddr1, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)
	valAddr1, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	delAddr2, err := sdk.AccAddressFromBech32("source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et")
	require.NoError(t, err)
	valAddr2, err := sdk.ValAddressFromBech32("sourcevaloper13fj7t2yptf9k6ad6fv38434znzay4s4pjk0r4f")
	require.NoError(t, err)

	err = k.AddLockup(ctx, delAddr1, valAddr1, math.NewInt(1000))
	require.NoError(t, err)
	err = k.AddLockup(ctx, delAddr1, valAddr1, math.NewInt(500))
	require.NoError(t, err)
	err = k.AddLockup(ctx, delAddr2, valAddr2, math.NewInt(700))
	require.NoError(t, err)

	totalDel1 := k.totalLockedAmountByAddr(ctx, delAddr1)
	require.Equal(t, math.NewInt(1500), totalDel1, "delAddr1 should have a total of 1500")

	totalDel2 := k.totalLockedAmountByAddr(ctx, delAddr2)
	require.Equal(t, math.NewInt(700), totalDel2, "delAddr2 should have a total of 700")

	delAddr3, err := sdk.AccAddressFromBech32("source1n34fvpteuanu2nx2a4hql4jvcrcnal3gsrjppy")
	require.NoError(t, err)
	totalDel3 := k.totalLockedAmountByAddr(ctx, delAddr3)
	require.True(t, totalDel3.IsZero(), "delAddr3 should have no lockups")
}

func TestHasLockup(t *testing.T) {
	k, ctx := setupKeeper(t)

	delAddr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	require.False(t, k.hasLockup(ctx, delAddr, valAddr))

	k.AddLockup(ctx, delAddr, valAddr, math.NewInt(100))
	require.True(t, k.hasLockup(ctx, delAddr, valAddr))

	err = k.subtractLockup(ctx, delAddr, valAddr, math.NewInt(100))
	require.NoError(t, err)
	require.False(t, k.hasLockup(ctx, delAddr, valAddr), "lockup should no longer exist after removing the entire amount")
}

func TestHasUnlockingLockup(t *testing.T) {
	k, ctx := setupKeeper(t)

	creationHeight := int64(1)
	timestamp := time.Date(2006, time.January, 2, 15, 4, 5, 1, time.UTC)

	delAddr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	require.False(t, k.HasUnlockingLockup(ctx, delAddr, valAddr, int64(1)))

	k.SetUnlockingLockup(ctx, delAddr, valAddr, creationHeight, math.NewInt(100), timestamp, timestamp)
	require.True(t, k.HasUnlockingLockup(ctx, delAddr, valAddr, creationHeight))

	err = k.subtractUnlockingLockup(ctx, delAddr, valAddr, creationHeight, math.NewInt(100))
	require.NoError(t, err)
	require.False(t, k.HasUnlockingLockup(ctx, delAddr, valAddr, creationHeight))
}

func TestGetUnlockingLockup(t *testing.T) {
	k, ctx := setupKeeper(t)

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

	expectedCompletionTime := ctx.BlockTime().Add(epochDuration * time.Duration(params.UnlockingEpochs))
	expectedUnlockTime := ctx.BlockTime().Add(epochDuration * time.Duration(params.UnlockingEpochs))

	k.SetUnlockingLockup(ctx, delAddr, valAddr, creationHeight, amount, expectedCompletionTime, expectedUnlockTime)

	unlockingLockup := k.GetUnlockingLockup(ctx, delAddr, valAddr, creationHeight)
	require.NotNil(t, unlockingLockup, "unlocking lockup should not be nil")
	require.Equal(t, amount, unlockingLockup.Amount, "amount should match the one set")
	require.Equal(t, expectedUnlockTime, unlockingLockup.CompletionTime, "completionTime should match the one set")
	require.Equal(t, expectedUnlockTime, unlockingLockup.UnlockTime, "unlockTime should match the one set")

	invalidUnlockingLockup := k.GetUnlockingLockup(ctx, delAddr, valAddr, creationHeight+1)
	require.Nil(t, invalidUnlockingLockup, "this unlocking lockup does not exist")
}

func TestGetLockup(t *testing.T) {
	k, ctx := setupKeeper(t)

	now := time.Now()
	creationHeight := int64(10)
	amount := math.NewInt(1000)

	delAddr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	ctx = ctx.WithBlockHeight(creationHeight).WithBlockTime(now)

	k.setLockup(ctx, delAddr, valAddr, amount)

	lockup := k.GetLockup(ctx, delAddr, valAddr)

	require.NotNil(t, lockup, "lockup should exist")
	require.Equal(t, delAddr.String(), lockup.DelegatorAddress, "delegator address should match")
	require.Equal(t, valAddr.String(), lockup.ValidatorAddress, "validator address should match")
	require.Equal(t, amount, lockup.Amount, "amount should match")

	nonExistentValAddr, err := sdk.ValAddressFromBech32("sourcevaloper13fj7t2yptf9k6ad6fv38434znzay4s4pjk0r4f")
	require.NoError(t, err)

	nonExistentLockup := k.GetLockup(ctx, delAddr, nonExistentValAddr)
	require.Nil(t, nonExistentLockup, "lockup should not exist for the given validator")
}

func TestSubtractUnlockingLockup(t *testing.T) {
	k, ctx := setupKeeper(t)

	params := k.GetParams(ctx)
	epochDuration := *params.EpochDuration
	unlockingLockupAmount := math.NewInt(1000)
	cancelUnlockAmount := math.NewInt(500)
	cancelUnlockAmount2 := math.NewInt(2000)
	creationHeight := int64(10)

	delAddr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	ctx = ctx.WithBlockHeight(creationHeight).WithBlockTime(time.Now())

	expectedCompletionTime := ctx.BlockTime().Add(epochDuration * time.Duration(params.UnlockingEpochs))
	expectedUnlockTime := ctx.BlockTime().Add(epochDuration * time.Duration(params.UnlockingEpochs))

	k.SetUnlockingLockup(ctx, delAddr, valAddr, creationHeight, unlockingLockupAmount, expectedCompletionTime, expectedUnlockTime)

	// subtract partial amount
	err = k.subtractUnlockingLockup(ctx, delAddr, valAddr, creationHeight, cancelUnlockAmount)
	require.NoError(t, err)

	unlockingLockup := k.GetUnlockingLockup(ctx, delAddr, valAddr, creationHeight)
	require.NotNil(t, unlockingLockup)
	require.Equal(t, cancelUnlockAmount, unlockingLockup.Amount)
	require.Equal(t, expectedCompletionTime, unlockingLockup.CompletionTime)
	require.Equal(t, expectedUnlockTime, unlockingLockup.UnlockTime)

	// try to subtract more than the locked amount
	err = k.subtractUnlockingLockup(ctx, delAddr, valAddr, creationHeight, cancelUnlockAmount2)
	require.Error(t, err)

	// subtract remaining amount
	err = k.subtractUnlockingLockup(ctx, delAddr, valAddr, creationHeight, cancelUnlockAmount)
	require.NoError(t, err)

	invalidUnlockingLockup := k.GetUnlockingLockup(ctx, delAddr, valAddr, creationHeight)
	require.Nil(t, invalidUnlockingLockup)
}

func TestSetAndGetTotalLockupsAmount(t *testing.T) {
	k, ctx := setupKeeper(t)

	total := k.GetTotalLockupsAmount(ctx)
	require.True(t, total.IsZero(), "Initial total lockups amount should be zero")

	expectedTotal := math.NewInt(1000)
	err := k.setTotalLockupsAmount(ctx, expectedTotal)
	require.NoError(t, err, "Setting total lockups amount should not fail")

	total = k.GetTotalLockupsAmount(ctx)
	require.Equal(t, expectedTotal, total, "Total lockups amount should be 1000")
}

func TestAddLockupUpdatesTotalLockups(t *testing.T) {
	k, ctx := setupKeeper(t)

	amount1 := math.NewInt(500)
	amount2 := math.NewInt(1000)

	delAddr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	err = k.AddLockup(ctx, delAddr, valAddr, amount1)
	require.NoError(t, err, "Adding first lockup should not fail")

	total := k.GetTotalLockupsAmount(ctx)
	require.Equal(t, amount1, total, "Total lockups amount should be 500")

	err = k.AddLockup(ctx, delAddr, valAddr, amount2)
	require.NoError(t, err, "Adding another lockup should not fail")

	total = k.GetTotalLockupsAmount(ctx)
	require.Equal(t, amount1.Add(amount2), total, "Total lockups amount should be 1500")
}

func TestSubtractLockupUpdatesTotalLockups(t *testing.T) {
	k, ctx := setupKeeper(t)

	initialAmount := math.NewInt(5000)
	subtractAmount := math.NewInt(1500)

	delAddr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	err = k.AddLockup(ctx, delAddr, valAddr, initialAmount)
	require.NoError(t, err, "Adding lockup should not fail")

	total := k.GetTotalLockupsAmount(ctx)
	require.Equal(t, initialAmount, total, "Total lockups amount should be 5000")

	err = k.subtractLockup(ctx, delAddr, valAddr, subtractAmount)
	require.NoError(t, err, "Subtracting lockup should not fail")

	total = k.GetTotalLockupsAmount(ctx)
	require.Equal(t, initialAmount.Sub(subtractAmount), total, "Total lockups amount should be 3500")
}

func TestRemoveLockupUpdatesTotalLockups(t *testing.T) {
	k, ctx := setupKeeper(t)

	initialAmount := math.NewInt(5000)
	subtractAmount := math.NewInt(5000)

	delAddr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	err = k.AddLockup(ctx, delAddr, valAddr, initialAmount)
	require.NoError(t, err, "Adding lockup should not fail")

	total := k.GetTotalLockupsAmount(ctx)
	require.Equal(t, initialAmount, total, "Total lockups amount should be 5000")

	err = k.subtractLockup(ctx, delAddr, valAddr, subtractAmount)
	require.NoError(t, err, "Subtracting lockup should not fail")

	total = k.GetTotalLockupsAmount(ctx)
	require.Equal(t, math.ZeroInt(), total, "Total lockups amount should be 0")
}
