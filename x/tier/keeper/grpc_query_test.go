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
	keeper, ctx := keepertest.TierKeeper(t)
	params := types.DefaultParams()
	require.NoError(t, keeper.SetParams(ctx, params))

	querier := tierkeeper.NewQuerier(keeper)
	response, err := querier.Params(ctx, &types.QueryParamsRequest{})

	require.NoError(t, err)
	require.Equal(t, &types.QueryParamsResponse{Params: params}, response)
}

func TestLockupQuery(t *testing.T) {
	keeper, ctx := keepertest.TierKeeper(t)
	amount := math.NewInt(1000)

	delAddr, err := sdk.AccAddressFromBech32("source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et")
	require.NoError(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	keeper.AddLockup(ctx, delAddr, valAddr, amount)

	querier := tierkeeper.NewQuerier(keeper)
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

func TestLockupsQuery(t *testing.T) {
	keeper, ctx := keepertest.TierKeeper(t)
	amount1 := math.NewInt(1000)
	amount2 := math.NewInt(500)

	delAddr, err := sdk.AccAddressFromBech32("source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et")
	require.NoError(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	keeper.AddLockup(ctx, delAddr, valAddr, amount1)
	keeper.AddLockup(ctx, delAddr, valAddr, amount2)

	querier := tierkeeper.NewQuerier(keeper)
	response, err := querier.Lockups(ctx, &types.LockupsRequest{
		DelegatorAddress: delAddr.String(),
	})

	require.NoError(t, err)
	require.Len(t, response.Lockup, 1)
	require.Equal(t, amount1.Add(amount2), response.Lockup[0].Amount)
}

func TestUnlockingLockupQuery(t *testing.T) {
	keeper, ctx := keepertest.TierKeeper(t)
	params := keeper.GetParams(ctx)
	epochDuration := *params.EpochDuration
	amount := math.NewInt(1000)

	delAddr, err := sdk.AccAddressFromBech32("source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et")
	require.NoError(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	ctx = ctx.WithBlockHeight(1).WithBlockTime(time.Now())

	keeper.SetLockup(ctx, true, delAddr, valAddr, amount, nil)

	unbondTime := ctx.BlockTime().Add(epochDuration * time.Duration(params.UnlockingEpochs))
	unlockTime := unbondTime

	querier := tierkeeper.NewQuerier(keeper)
	response, err := querier.UnlockingLockup(ctx, &types.UnlockingLockupRequest{
		DelegatorAddress: delAddr.String(),
		ValidatorAddress: valAddr.String(),
		CreationHeight:   1,
	})

	// use normalized time to confirm SetLockup() logic
	unbondTimeUTC := unbondTime.UTC()
	unlockTimeUTC := unlockTime.UTC()

	require.NoError(t, err)
	require.Equal(t, &types.UnlockingLockupResponse{
		Lockup: types.Lockup{
			DelegatorAddress: delAddr.String(),
			ValidatorAddress: valAddr.String(),
			Amount:           amount,
			UnbondTime:       &unbondTimeUTC,
			UnlockTime:       &unlockTimeUTC,
		},
	}, response)
}

func TestUnlockingLockupsQuery(t *testing.T) {
	keeper, ctx := keepertest.TierKeeper(t)
	params := keeper.GetParams(ctx)
	epochDuration := *params.EpochDuration
	amount1 := math.NewInt(1000)
	amount2 := math.NewInt(500)

	delAddr, err := sdk.AccAddressFromBech32("source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et")
	require.NoError(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	ctx = ctx.WithBlockHeight(1).WithBlockTime(time.Now())
	keeper.SetLockup(ctx, true, delAddr, valAddr, amount1, nil)

	unbondTime1 := ctx.BlockTime().Add(epochDuration * time.Duration(params.UnlockingEpochs))
	unlockTime1 := unbondTime1

	ctx = ctx.WithBlockHeight(2).WithBlockTime(unbondTime1)
	keeper.SetLockup(ctx, true, delAddr, valAddr, amount2, nil)

	unbondTime2 := ctx.BlockTime().Add(epochDuration * time.Duration(params.UnlockingEpochs))
	unlockTime2 := unbondTime2

	querier := tierkeeper.NewQuerier(keeper)
	response, err := querier.UnlockingLockups(ctx, &types.UnlockingLockupsRequest{
		DelegatorAddress: delAddr.String(),
	})

	require.NoError(t, err)
	require.Len(t, response.Lockup, 2)

	require.Equal(t, amount1, response.Lockup[0].Amount)
	require.Equal(t, int64(1), response.Lockup[0].CreationHeight)
	require.Equal(t, &unbondTime1, response.Lockup[0].UnbondTime)
	require.Equal(t, &unlockTime1, response.Lockup[0].UnlockTime)

	require.Equal(t, amount2, response.Lockup[1].Amount)
	require.Equal(t, int64(2), response.Lockup[1].CreationHeight)
	require.Equal(t, &unbondTime2, response.Lockup[1].UnbondTime)
	require.Equal(t, &unlockTime2, response.Lockup[1].UnlockTime)
}
