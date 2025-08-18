package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/stretchr/testify/require"

	appparams "github.com/sourcenetwork/sourcehub/app/params"
	keepertest "github.com/sourcenetwork/sourcehub/testutil/keeper"
	"github.com/sourcenetwork/sourcehub/x/tier/types"
)

func TestMsgRemoveUserSubscription(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	p := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, p))

	validDeveloperAddr := "source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9"
	validUserAddr := "source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et"

	developerAddr := sdk.MustAccAddressFromBech32(validDeveloperAddr)
	userAddr := sdk.MustAccAddressFromBech32(validUserAddr)

	keepertest.CreateAccount(t, &k, sdkCtx, developerAddr)
	keepertest.CreateAccount(t, &k, sdkCtx, userAddr)

	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	keepertest.InitializeDelegator(t, &k, sdkCtx, developerAddr, math.NewInt(5000))
	keepertest.InitializeValidator(t, k.GetStakingKeeper().(*stakingkeeper.Keeper), sdkCtx, valAddr, math.NewInt(1_000_000))

	sdkCtx = sdkCtx.WithBlockHeight(1).WithBlockTime(time.Now())

	err = k.CreateDeveloper(sdkCtx, developerAddr, true)
	require.NoError(t, err)

	amount := sdk.NewCoin(appparams.MicroCreditDenom, math.NewInt(100))
	err = k.AddUserSubscription(sdkCtx, developerAddr, userAddr, &amount, 30)
	require.NoError(t, err)

	testCases := []struct {
		name      string
		input     *types.MsgRemoveUserSubscription
		expErr    bool
		expErrMsg string
	}{
		{
			name: "valid remove user subscription",
			input: &types.MsgRemoveUserSubscription{
				Developer: validDeveloperAddr,
				User:      validUserAddr,
			},
			expErr: false,
		},
		{
			name: "invalid developer address",
			input: &types.MsgRemoveUserSubscription{
				Developer: "invalid-developer-address",
				User:      validUserAddr,
			},
			expErr:    true,
			expErrMsg: "delegator address",
		},
		{
			name: "invalid user address",
			input: &types.MsgRemoveUserSubscription{
				Developer: validDeveloperAddr,
				User:      "invalid-user-address",
			},
			expErr:    true,
			expErrMsg: "delegator address",
		},
		{
			name: "empty developer address",
			input: &types.MsgRemoveUserSubscription{
				Developer: "",
				User:      validUserAddr,
			},
			expErr:    true,
			expErrMsg: "delegator address",
		},
		{
			name: "empty user address",
			input: &types.MsgRemoveUserSubscription{
				Developer: validDeveloperAddr,
				User:      "",
			},
			expErr:    true,
			expErrMsg: "delegator address",
		},
		{
			name: "non-existent subscription",
			input: &types.MsgRemoveUserSubscription{
				Developer: validDeveloperAddr,
				User:      "source1n34fvpteuanu2nx2a4hql4jvcrcnal3gsrjppy",
			},
			expErr:    true,
			expErrMsg: "remove user subscription",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.name == "valid remove user subscription" {
				subscription := k.GetUserSubscription(sdkCtx, developerAddr, userAddr)
				if subscription == nil {
					err := k.AddUserSubscription(ctx, developerAddr, userAddr, &amount, 30)
					require.NoError(t, err)
				}
			}

			err := tc.input.ValidateBasic()
			if err != nil {
				if tc.expErr {
					require.Contains(t, err.Error(), tc.expErrMsg)
					return
				}
				t.Fatalf("unexpected error in ValidateBasic: %v", err)
			}

			resp, err := ms.RemoveUserSubscription(sdkCtx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				developerAddr := sdk.MustAccAddressFromBech32(tc.input.Developer)
				userAddr := sdk.MustAccAddressFromBech32(tc.input.User)
				subscription := k.GetUserSubscription(sdkCtx, developerAddr, userAddr)
				require.Nil(t, subscription, "User subscription should not exist after removal")
			}
		})
	}
}

func TestMsgRemoveUserSubscription_MultipleSubscriptions(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	p := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, p))

	validDeveloperAddr := "source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9"
	user1Addr := "source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et"
	user2Addr := "source18jtkvj0995fy7lggqayg2f5syna92ndq5mkuv4"

	developerAddr := sdk.MustAccAddressFromBech32(validDeveloperAddr)
	user1Address := sdk.MustAccAddressFromBech32(user1Addr)
	user2Address := sdk.MustAccAddressFromBech32(user2Addr)

	keepertest.CreateAccount(t, &k, sdkCtx, developerAddr)
	keepertest.CreateAccount(t, &k, sdkCtx, user1Address)
	keepertest.CreateAccount(t, &k, sdkCtx, user2Address)

	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	keepertest.InitializeDelegator(t, &k, sdkCtx, developerAddr, math.NewInt(5000))
	keepertest.InitializeValidator(t, k.GetStakingKeeper().(*stakingkeeper.Keeper), sdkCtx, valAddr, math.NewInt(1_000_000))

	sdkCtx = sdkCtx.WithBlockHeight(1).WithBlockTime(time.Now())

	err = k.CreateDeveloper(sdkCtx, developerAddr, true)
	require.NoError(t, err)

	amount1 := sdk.NewCoin(appparams.MicroCreditDenom, math.NewInt(100))
	amount2 := sdk.NewCoin(appparams.MicroCreditDenom, math.NewInt(200))
	err = k.AddUserSubscription(sdkCtx, developerAddr, user1Address, &amount1, 30)
	require.NoError(t, err)
	err = k.AddUserSubscription(sdkCtx, developerAddr, user2Address, &amount2, 60)
	require.NoError(t, err)

	subscription1 := k.GetUserSubscription(sdkCtx, developerAddr, user1Address)
	require.NotNil(t, subscription1)
	subscription2 := k.GetUserSubscription(sdkCtx, developerAddr, user2Address)
	require.NotNil(t, subscription2)

	msg := &types.MsgRemoveUserSubscription{
		Developer: validDeveloperAddr,
		User:      user1Addr,
	}

	resp, err := ms.RemoveUserSubscription(sdkCtx, msg)
	require.NoError(t, err)
	require.NotNil(t, resp)

	subscription1 = k.GetUserSubscription(sdkCtx, developerAddr, user1Address)
	require.Nil(t, subscription1, "User1's subscription should be removed")
	subscription2 = k.GetUserSubscription(sdkCtx, developerAddr, user2Address)
	require.NotNil(t, subscription2, "User2's subscription should still exist")

	msg.User = user2Addr
	resp, err = ms.RemoveUserSubscription(sdkCtx, msg)
	require.NoError(t, err)
	require.NotNil(t, resp)

	subscription1 = k.GetUserSubscription(sdkCtx, developerAddr, user1Address)
	require.Nil(t, subscription1, "User1's subscription should be removed")
	subscription2 = k.GetUserSubscription(sdkCtx, developerAddr, user2Address)
	require.Nil(t, subscription2, "User2's subscription should be removed")
}

func TestMsgRemoveUserSubscription_AlreadyRemoved(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	p := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, p))

	validDeveloperAddr := "source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9"
	validUserAddr := "source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et"

	developerAddr := sdk.MustAccAddressFromBech32(validDeveloperAddr)
	userAddr := sdk.MustAccAddressFromBech32(validUserAddr)

	keepertest.CreateAccount(t, &k, sdkCtx, developerAddr)
	keepertest.CreateAccount(t, &k, sdkCtx, userAddr)

	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	keepertest.InitializeDelegator(t, &k, sdkCtx, developerAddr, math.NewInt(5000))
	keepertest.InitializeValidator(t, k.GetStakingKeeper().(*stakingkeeper.Keeper), sdkCtx, valAddr, math.NewInt(1_000_000))

	sdkCtx = sdkCtx.WithBlockHeight(1).WithBlockTime(time.Now())

	err = k.CreateDeveloper(sdkCtx, developerAddr, true)
	require.NoError(t, err)

	amount := sdk.NewCoin(appparams.MicroCreditDenom, math.NewInt(100))
	err = k.AddUserSubscription(sdkCtx, developerAddr, userAddr, &amount, 30)
	require.NoError(t, err)

	err = k.RemoveUserSubscription(sdkCtx, developerAddr, userAddr)
	require.NoError(t, err)

	msg := &types.MsgRemoveUserSubscription{
		Developer: validDeveloperAddr,
		User:      validUserAddr,
	}

	resp, err := ms.RemoveUserSubscription(sdkCtx, msg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "remove user subscription")
	require.Nil(t, resp)
}

func TestMsgRemoveUserSubscription_NonExistentDeveloper(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	p := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, p))

	nonExistentDeveloperAddr := "source1n34fvpteuanu2nx2a4hql4jvcrcnal3gsrjppy"
	validUserAddr := "source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et"

	msg := &types.MsgRemoveUserSubscription{
		Developer: nonExistentDeveloperAddr,
		User:      validUserAddr,
	}

	resp, err := ms.RemoveUserSubscription(sdkCtx, msg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "remove user subscription")
	require.Nil(t, resp)
}
