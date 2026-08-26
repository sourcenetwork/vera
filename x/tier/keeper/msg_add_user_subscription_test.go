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

func TestMsgAddUserSubscription(t *testing.T) {
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

	err = k.CreateDeveloper(ctx, developerAddr, true)
	require.NoError(t, err)

	validAmount := uint64(100)
	zeroAmount := uint64(0)

	testCases := []struct {
		name      string
		input     *types.MsgAddUserSubscription
		expErr    bool
		expErrMsg string
	}{
		{
			name: "valid add user subscription",
			input: &types.MsgAddUserSubscription{
				Developer: validDeveloperAddr,
				UserDid:   user1Did,
				Amount:    validAmount,
				Period:    30,
			},
			expErr: false,
		},
		{
			name: "valid add user subscription with zero period",
			input: &types.MsgAddUserSubscription{
				Developer: validDeveloperAddr,
				UserDid:   user2Did,
				Amount:    validAmount,
				Period:    0,
			},
			expErr:    true,
			expErrMsg: "invalid subscription period",
		},
		{
			name: "invalid developer address",
			input: &types.MsgAddUserSubscription{
				Developer: "invalid-developer-address",
				UserDid:   user1Did,
				Amount:    validAmount,
				Period:    30,
			},
			expErr:    true,
			expErrMsg: "delegator address",
		},
		{
			name: "invalid user did",
			input: &types.MsgAddUserSubscription{
				Developer: validDeveloperAddr,
				UserDid:   "",
				Amount:    validAmount,
				Period:    30,
			},
			expErr:    true,
			expErrMsg: "invalid DID",
		},
		{
			name: "zero amount",
			input: &types.MsgAddUserSubscription{
				Developer: validDeveloperAddr,
				UserDid:   user1Did,
				Amount:    zeroAmount,
				Period:    30,
			},
			expErr:    true,
			expErrMsg: "invalid amount",
		},
		{
			name: "non-existent developer",
			input: &types.MsgAddUserSubscription{
				Developer: "source1n34fvpteuanu2nx2a4hql4jvcrcnal3gsrjppy",
				UserDid:   user1Did,
				Amount:    validAmount,
				Period:    30,
			},
			expErr:    true,
			expErrMsg: "add subscription",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.input.ValidateBasic()
			if err != nil {
				if tc.expErr {
					require.Contains(t, err.Error(), tc.expErrMsg)
					return
				}
				t.Fatalf("unexpected error in ValidateBasic: %v", err)
			}

			resp, err := ms.AddUserSubscription(sdkCtx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				developerAddr := sdk.MustAccAddressFromBech32(tc.input.Developer)
				subscription := k.GetUserSubscription(sdkCtx, developerAddr, tc.input.UserDid)
				require.NotNil(t, subscription, "User subscription should exist after creation")
				require.Equal(t, tc.input.Amount, subscription.CreditAmount, "Amount should match input")
				require.Equal(t, tc.input.Period, subscription.Period, "Period should match input")
			}
		})
	}
}

func TestMsgAddUserSubscription_DuplicateSubscription(t *testing.T) {
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

	err = k.CreateDeveloper(ctx, developerAddr, true)
	require.NoError(t, err)

	validAmount := uint64(100)
	msg := &types.MsgAddUserSubscription{
		Developer: validDeveloperAddr,
		UserDid:   userDid,
		Amount:    validAmount,
		Period:    30,
	}

	resp, err := ms.AddUserSubscription(sdkCtx, msg)
	require.NoError(t, err)
	require.NotNil(t, resp)

	resp, err = ms.AddUserSubscription(sdkCtx, msg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "add subscription")
}

func TestMsgAddUserSubscription_MultipleDifferentUsers(t *testing.T) {
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

	err = k.CreateDeveloper(ctx, developerAddr, true)
	require.NoError(t, err)

	validAmount := uint64(100)
	msg1 := &types.MsgAddUserSubscription{
		Developer: validDeveloperAddr,
		UserDid:   user1Did,
		Amount:    validAmount,
		Period:    30,
	}

	resp, err := ms.AddUserSubscription(sdkCtx, msg1)
	require.NoError(t, err)
	require.NotNil(t, resp)

	msg2 := &types.MsgAddUserSubscription{
		Developer: validDeveloperAddr,
		UserDid:   user2Did,
		Amount:    uint64(200),
		Period:    60,
	}

	resp, err = ms.AddUserSubscription(sdkCtx, msg2)
	require.NoError(t, err)
	require.NotNil(t, resp)

	subscription1 := k.GetUserSubscription(sdkCtx, developerAddr, user1Did)
	require.NotNil(t, subscription1)
	require.Equal(t, validAmount, subscription1.CreditAmount)
	require.Equal(t, uint64(30), subscription1.Period)

	subscription2 := k.GetUserSubscription(sdkCtx, developerAddr, user2Did)
	require.NotNil(t, subscription2)
	require.Equal(t, uint64(200), subscription2.CreditAmount)
	require.Equal(t, uint64(60), subscription2.Period)
}
