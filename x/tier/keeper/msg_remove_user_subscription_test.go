package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/stretchr/testify/require"

	keepertest "github.com/sourcenetwork/vera/testutil/keeper"
	"github.com/sourcenetwork/vera/x/tier/types"
)

func TestMsgRemoveUserSubscription(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	p := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, p))

	userDid := "did:key:alice"

	validDeveloperAddr := "source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9"
	developerAddr := sdk.MustAccAddressFromBech32(validDeveloperAddr)

	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	keepertest.InitializeDelegator(t, &k, sdkCtx, developerAddr, math.NewInt(5000))
	keepertest.InitializeValidator(t, k.GetStakingKeeper().(*stakingkeeper.Keeper), sdkCtx, valAddr, math.NewInt(1_000_000))

	sdkCtx = sdkCtx.WithBlockHeight(1).WithBlockTime(time.Now())

	err = k.CreateDeveloper(sdkCtx, developerAddr, true)
	require.NoError(t, err)

	amount := uint64(100)
	err = k.AddUserSubscription(sdkCtx, developerAddr, userDid, amount, 30)
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
				UserDid:   userDid,
			},
			expErr: false,
		},
		{
			name: "invalid developer address",
			input: &types.MsgRemoveUserSubscription{
				Developer: "invalid-developer-address",
				UserDid:   userDid,
			},
			expErr:    true,
			expErrMsg: "delegator address",
		},
		{
			name: "empty developer address",
			input: &types.MsgRemoveUserSubscription{
				Developer: "",
				UserDid:   userDid,
			},
			expErr:    true,
			expErrMsg: "delegator address",
		},
		{
			name: "empty user did",
			input: &types.MsgRemoveUserSubscription{
				Developer: validDeveloperAddr,
				UserDid:   "",
			},
			expErr:    true,
			expErrMsg: "invalid DID",
		},
		{
			name: "non-existent subscription",
			input: &types.MsgRemoveUserSubscription{
				Developer: validDeveloperAddr,
				UserDid:   userDid,
			},
			expErr:    true,
			expErrMsg: "remove user subscription",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.name == "valid remove user subscription" {
				subscription := k.GetUserSubscription(sdkCtx, developerAddr, userDid)
				if subscription == nil {
					err := k.AddUserSubscription(ctx, developerAddr, userDid, amount, 30)
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
				subscription := k.GetUserSubscription(sdkCtx, developerAddr, userDid)
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

	user1Did := "did:key:alice"
	user2Did := "did:key:bob"

	validDeveloperAddr := "source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9"
	developerAddr := sdk.MustAccAddressFromBech32(validDeveloperAddr)

	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	keepertest.InitializeDelegator(t, &k, sdkCtx, developerAddr, math.NewInt(5000))
	keepertest.InitializeValidator(t, k.GetStakingKeeper().(*stakingkeeper.Keeper), sdkCtx, valAddr, math.NewInt(1_000_000))

	sdkCtx = sdkCtx.WithBlockHeight(1).WithBlockTime(time.Now())

	err = k.CreateDeveloper(sdkCtx, developerAddr, true)
	require.NoError(t, err)

	amount1 := uint64(100)
	amount2 := uint64(200)
	err = k.AddUserSubscription(sdkCtx, developerAddr, user1Did, amount1, 30)
	require.NoError(t, err)
	err = k.AddUserSubscription(sdkCtx, developerAddr, user2Did, amount2, 60)
	require.NoError(t, err)

	subscription1 := k.GetUserSubscription(sdkCtx, developerAddr, user1Did)
	require.NotNil(t, subscription1)
	subscription2 := k.GetUserSubscription(sdkCtx, developerAddr, user2Did)
	require.NotNil(t, subscription2)

	msg := &types.MsgRemoveUserSubscription{
		Developer: validDeveloperAddr,
		UserDid:   user1Did,
	}

	resp, err := ms.RemoveUserSubscription(sdkCtx, msg)
	require.NoError(t, err)
	require.NotNil(t, resp)

	subscription1 = k.GetUserSubscription(sdkCtx, developerAddr, user1Did)
	require.Nil(t, subscription1, "User1's subscription should be removed")
	subscription2 = k.GetUserSubscription(sdkCtx, developerAddr, user2Did)
	require.NotNil(t, subscription2, "User2's subscription should still exist")

	msg.UserDid = user2Did
	resp, err = ms.RemoveUserSubscription(sdkCtx, msg)
	require.NoError(t, err)
	require.NotNil(t, resp)

	subscription1 = k.GetUserSubscription(sdkCtx, developerAddr, user1Did)
	require.Nil(t, subscription1, "User1's subscription should be removed")
	subscription2 = k.GetUserSubscription(sdkCtx, developerAddr, user2Did)
	require.Nil(t, subscription2, "User2's subscription should be removed")
}

func TestMsgRemoveUserSubscription_AlreadyRemoved(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	p := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, p))

	userDid := "did:key:alice"

	validDeveloperAddr := "source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9"
	developerAddr := sdk.MustAccAddressFromBech32(validDeveloperAddr)

	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	keepertest.InitializeDelegator(t, &k, sdkCtx, developerAddr, math.NewInt(5000))
	keepertest.InitializeValidator(t, k.GetStakingKeeper().(*stakingkeeper.Keeper), sdkCtx, valAddr, math.NewInt(1_000_000))

	sdkCtx = sdkCtx.WithBlockHeight(1).WithBlockTime(time.Now())

	err = k.CreateDeveloper(sdkCtx, developerAddr, true)
	require.NoError(t, err)

	amount := uint64(100)
	err = k.AddUserSubscription(sdkCtx, developerAddr, userDid, amount, 30)
	require.NoError(t, err)

	err = k.RemoveUserSubscription(sdkCtx, developerAddr, userDid)
	require.NoError(t, err)

	msg := &types.MsgRemoveUserSubscription{
		Developer: validDeveloperAddr,
		UserDid:   userDid,
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
	userDid := "did:key:alice"

	msg := &types.MsgRemoveUserSubscription{
		Developer: nonExistentDeveloperAddr,
		UserDid:   userDid,
	}

	resp, err := ms.RemoveUserSubscription(sdkCtx, msg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "remove user subscription")
	require.Nil(t, resp)
}
