package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/stretchr/testify/require"

	keepertest "github.com/sourcenetwork/sourcehub/testutil/keeper"
	"github.com/sourcenetwork/sourcehub/x/tier/types"
)

func TestParamsQuery(t *testing.T) {
	k, ctx := keepertest.TierKeeper(t)
	params := k.GetParams(ctx)
	require.NoError(t, k.SetParams(ctx, params))

	response, err := k.Params(ctx, &types.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, &types.QueryParamsResponse{Params: params}, response)
}

func TestParamsQuery_InvalidRequest(t *testing.T) {
	k, ctx := keepertest.TierKeeper(t)
	params := k.GetParams(ctx)
	require.NoError(t, k.SetParams(ctx, params))

	response, err := k.Params(ctx, nil)
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

	response, err := k.Lockup(ctx, &types.LockupRequest{
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

	response, err := k.Lockup(ctx, nil)
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

	response, err := k.Lockups(ctx, &types.LockupsRequest{
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

	response, err := k.Lockups(ctx, nil)
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

	response, err := k.UnlockingLockup(ctx, &types.UnlockingLockupRequest{
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

	response, err := k.UnlockingLockup(ctx, nil)
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

	response, err := k.UnlockingLockups(ctx, &types.UnlockingLockupsRequest{
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

	response, err := k.UnlockingLockups(ctx, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid request")
	require.Nil(t, response)
}

func TestDevelopersQuery(t *testing.T) {
	k, ctx := keepertest.TierKeeper(t)

	dev1, err := sdk.AccAddressFromBech32("source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et")
	require.NoError(t, err)
	dev2, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)

	require.NoError(t, k.CreateDeveloper(ctx, dev1, true))
	require.NoError(t, k.CreateDeveloper(ctx, dev2, false))

	resp, err := k.Developers(ctx, &types.DevelopersRequest{})
	require.NoError(t, err)
	require.Len(t, resp.Developers, 2)

	page1, err := k.Developers(ctx, &types.DevelopersRequest{Pagination: &query.PageRequest{Limit: 1}})
	require.NoError(t, err)
	require.Len(t, page1.Developers, 1)
	require.NotNil(t, page1.Pagination)
	require.NotEmpty(t, page1.Pagination.NextKey)

	page2, err := k.Developers(ctx, &types.DevelopersRequest{Pagination: &query.PageRequest{Key: page1.Pagination.NextKey, Limit: 1}})
	require.NoError(t, err)
	require.Len(t, page2.Developers, 1)
}

func TestDevelopersQuery_InvalidRequest(t *testing.T) {
	k, ctx := keepertest.TierKeeper(t)
	params := k.GetParams(ctx)
	require.NoError(t, k.SetParams(ctx, params))

	resp, err := k.Developers(ctx, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid request")
	require.Nil(t, resp)
}

func TestDeveloperQuery(t *testing.T) {
	k, ctx := keepertest.TierKeeper(t)

	address, err := sdk.AccAddressFromBech32("source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et")
	require.NoError(t, err)
	require.NoError(t, k.CreateDeveloper(ctx, address, true))

	response, err := k.Developer(ctx, &types.DeveloperRequest{Address: address.String()})
	require.NoError(t, err)
	require.Equal(t, address.String(), response.Developer.Address)
	require.True(t, response.Developer.AutoLockEnabled)
}

func TestDeveloperQueryErrors(t *testing.T) {
	k, ctx := keepertest.TierKeeper(t)

	response, err := k.Developer(ctx, nil)
	require.ErrorContains(t, err, "invalid request")
	require.Nil(t, response)

	response, err = k.Developer(ctx, &types.DeveloperRequest{Address: "invalid"})
	require.ErrorContains(t, err, "invalid developer address")
	require.Nil(t, response)

	address := "source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et"
	response, err = k.Developer(ctx, &types.DeveloperRequest{Address: address})
	require.ErrorContains(t, err, "developer does not exist")
	require.Nil(t, response)
}

func TestUserSubscriptionsQuery(t *testing.T) {
	k, ctx := keepertest.TierKeeper(t)

	developerAddr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	user1Did := "did:key:alice"
	user2Did := "did:key:bob"

	require.NoError(t, k.CreateDeveloper(ctx, developerAddr, true))

	initialValidatorBalance := math.NewInt(1000)
	keepertest.InitializeValidator(t, k.GetStakingKeeper().(*stakingkeeper.Keeper), ctx, valAddr, initialValidatorBalance)

	require.NoError(t, k.AddUserSubscription(ctx, developerAddr, user1Did, uint64(0), 0))
	require.NoError(t, k.AddUserSubscription(ctx, developerAddr, user2Did, uint64(0), 0))

	resp, err := k.UserSubscriptions(ctx, &types.UserSubscriptionsRequest{Developer: developerAddr.String()})
	require.NoError(t, err)
	require.Len(t, resp.UserSubscriptions, 2)

	page1, err := k.UserSubscriptions(ctx, &types.UserSubscriptionsRequest{
		Developer:  developerAddr.String(),
		Pagination: &query.PageRequest{Limit: 1},
	})
	require.NoError(t, err)
	require.Len(t, page1.UserSubscriptions, 1)
	require.NotNil(t, page1.Pagination)
	require.NotEmpty(t, page1.Pagination.NextKey)

	page2, err := k.UserSubscriptions(ctx, &types.UserSubscriptionsRequest{
		Developer:  developerAddr.String(),
		Pagination: &query.PageRequest{Key: page1.Pagination.NextKey, Limit: 1},
	})
	require.NoError(t, err)
	require.Len(t, page2.UserSubscriptions, 1)
}

func TestUserSubscriptionsQuery_InvalidRequest(t *testing.T) {
	k, ctx := keepertest.TierKeeper(t)
	params := k.GetParams(ctx)
	require.NoError(t, k.SetParams(ctx, params))

	resp, err := k.UserSubscriptions(ctx, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid request")
	require.Nil(t, resp)
}
