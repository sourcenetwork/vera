package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/sourcenetwork/sourcehub/testutil/keeper"
	tierkeeper "github.com/sourcenetwork/sourcehub/x/tier/keeper"
	"github.com/sourcenetwork/sourcehub/x/tier/types"
)

func TestParamsQuery(t *testing.T) {
	k, ctx := keepertest.TierKeeper(t)
	params := k.GetParams(ctx)
	require.NoError(t, k.SetParams(ctx, params))

	querier := tierkeeper.NewQuerier(k)
	response, err := querier.Params(ctx, &types.QueryParamsRequest{})

	require.NoError(t, err)
	require.Equal(t, &types.QueryParamsResponse{Params: params}, response)
}

func TestParamsQuery_InvalidRequest(t *testing.T) {
	k, ctx := keepertest.TierKeeper(t)
	params := k.GetParams(ctx)
	require.NoError(t, k.SetParams(ctx, params))

	querier := tierkeeper.NewQuerier(k)
	response, err := querier.Params(ctx, nil)

	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid request")
	require.Nil(t, response)
}

func TestLockupQuery(t *testing.T) {
	k, ctx := keepertest.TierKeeper(t)
	amount := math.NewInt(1000)

	delAddr, err := sdk.AccAddressFromBech32("source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et")
	require.NoError(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	err = k.AddLockup(ctx, delAddr, valAddr, amount)
	require.NoError(t, err)

	querier := tierkeeper.NewQuerier(k)
	response, err := querier.Lockup(ctx, &types.LockupRequest{
		DelegatorAddress: delAddr.String(),
		ValidatorAddress: valAddr.String(),
	})

	require.NoError(t, err)
	require.Equal(t, &types.LockupResponse{
		Lockup: types.Lockup{
			DelegatorAddress: delAddr.String(),
			ValidatorAddress: valAddr.String(),
			Amount:           amount,
		},
	}, response)
}

func TestLockupQuery_InvalidRequest(t *testing.T) {
	k, ctx := keepertest.TierKeeper(t)
	params := k.GetParams(ctx)
	require.NoError(t, k.SetParams(ctx, params))

	querier := tierkeeper.NewQuerier(k)
	response, err := querier.Lockup(ctx, nil)

	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid request")
	require.Nil(t, response)
}

func TestLockupsQuery(t *testing.T) {
	k, ctx := keepertest.TierKeeper(t)
	amount1 := math.NewInt(1000)
	amount2 := math.NewInt(500)

	delAddr, err := sdk.AccAddressFromBech32("source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et")
	require.NoError(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	err = k.AddLockup(ctx, delAddr, valAddr, amount1)
	require.NoError(t, err)
	err = k.AddLockup(ctx, delAddr, valAddr, amount2)
	require.NoError(t, err)

	querier := tierkeeper.NewQuerier(k)
	response, err := querier.Lockups(ctx, &types.LockupsRequest{
		DelegatorAddress: delAddr.String(),
	})

	require.NoError(t, err)
	require.Len(t, response.Lockups, 1)
	require.Equal(t, amount1.Add(amount2), response.Lockups[0].Amount)
}

func TestLockupsQuery_InvalidRequest(t *testing.T) {
	k, ctx := keepertest.TierKeeper(t)
	params := k.GetParams(ctx)
	require.NoError(t, k.SetParams(ctx, params))

	querier := tierkeeper.NewQuerier(k)
	response, err := querier.Lockups(ctx, nil)

	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid request")
	require.Nil(t, response)
}

func TestUnlockingLockupQuery(t *testing.T) {
	k, ctx := keepertest.TierKeeper(t)
	params := k.GetParams(ctx)
	epochDuration := *params.EpochDuration
	amount := math.NewInt(1000)

	delAddr, err := sdk.AccAddressFromBech32("source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et")
	require.NoError(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	ctx = ctx.WithBlockHeight(1).WithBlockTime(time.Now())

	completionTime := ctx.BlockTime().Add(epochDuration * time.Duration(params.UnlockingEpochs))
	unlockTime := ctx.BlockTime().Add(epochDuration * time.Duration(params.UnlockingEpochs))

	k.SetUnlockingLockup(ctx, delAddr, valAddr, int64(1), amount, completionTime, unlockTime)

	querier := tierkeeper.NewQuerier(k)
	response, err := querier.UnlockingLockup(ctx, &types.UnlockingLockupRequest{
		DelegatorAddress: delAddr.String(),
		ValidatorAddress: valAddr.String(),
		CreationHeight:   1,
	})

	require.NoError(t, err)
	require.Equal(t, &types.UnlockingLockupResponse{
		UnlockingLockup: types.UnlockingLockup{
			DelegatorAddress: delAddr.String(),
			ValidatorAddress: valAddr.String(),
			CreationHeight:   1,
			Amount:           amount,
			CompletionTime:   completionTime,
			UnlockTime:       unlockTime,
		},
	}, response)
}

func TestUnlockingLockupQuery_InvalidRequest(t *testing.T) {
	k, ctx := keepertest.TierKeeper(t)
	params := k.GetParams(ctx)
	require.NoError(t, k.SetParams(ctx, params))

	querier := tierkeeper.NewQuerier(k)
	response, err := querier.UnlockingLockup(ctx, nil)

	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid request")
	require.Nil(t, response)
}

func TestUnlockingLockupsQuery(t *testing.T) {
	k, ctx := keepertest.TierKeeper(t)
	params := k.GetParams(ctx)
	epochDuration := *params.EpochDuration
	amount1 := math.NewInt(1000)
	amount2 := math.NewInt(500)

	delAddr, err := sdk.AccAddressFromBech32("source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et")
	require.NoError(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	ctx = ctx.WithBlockHeight(1).WithBlockTime(time.Now())

	completionTime1 := ctx.BlockTime().Add(time.Hour * 24 * 21)
	unlockTime1 := ctx.BlockTime().Add(epochDuration * time.Duration(params.UnlockingEpochs))
	k.SetUnlockingLockup(ctx, delAddr, valAddr, int64(1), amount1, completionTime1, unlockTime1)

	ctx = ctx.WithBlockHeight(2).WithBlockTime(ctx.BlockTime().Add(time.Second))

	completionTime2 := ctx.BlockTime().Add(time.Hour * 24 * 21)
	unlockTime2 := ctx.BlockTime().Add(epochDuration * time.Duration(params.UnlockingEpochs))
	k.SetUnlockingLockup(ctx, delAddr, valAddr, int64(2), amount2, completionTime2, unlockTime2)

	querier := tierkeeper.NewQuerier(k)
	response, err := querier.UnlockingLockups(ctx, &types.UnlockingLockupsRequest{
		DelegatorAddress: delAddr.String(),
	})

	require.NoError(t, err)
	require.Len(t, response.UnlockingLockups, 2)

	require.Equal(t, amount1, response.UnlockingLockups[0].Amount)
	require.Equal(t, int64(1), response.UnlockingLockups[0].CreationHeight)
	require.Equal(t, completionTime1, response.UnlockingLockups[0].CompletionTime)
	require.Equal(t, unlockTime1, response.UnlockingLockups[0].UnlockTime)

	require.Equal(t, amount2, response.UnlockingLockups[1].Amount)
	require.Equal(t, int64(2), response.UnlockingLockups[1].CreationHeight)
	require.Equal(t, completionTime2, response.UnlockingLockups[1].CompletionTime)
	require.Equal(t, unlockTime2, response.UnlockingLockups[1].UnlockTime)
}

func TestUnlockingLockupsQuery_InvalidRequest(t *testing.T) {
	k, ctx := keepertest.TierKeeper(t)
	params := k.GetParams(ctx)
	require.NoError(t, k.SetParams(ctx, params))

	querier := tierkeeper.NewQuerier(k)
	response, err := querier.UnlockingLockups(ctx, nil)

	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid request")
	require.Nil(t, response)
}
