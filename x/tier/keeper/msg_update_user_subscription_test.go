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

func TestMsgUpdateUserSubscription(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	p := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, p))

	userDid := "did:key:alice"

	validDeveloperAddr := "vera1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s2dq2wz"
	developerAddr := sdk.MustAccAddressFromBech32(validDeveloperAddr)

	valAddr, err := sdk.ValAddressFromBech32("veravaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnduc4vm0")
	require.NoError(t, err)

	keepertest.InitializeDelegator(t, &k, sdkCtx, developerAddr, math.NewInt(10000))
	keepertest.InitializeValidator(t, k.GetStakingKeeper().(*stakingkeeper.Keeper), sdkCtx, valAddr, math.NewInt(1_000_000))
	sdkCtx = sdkCtx.WithBlockHeight(1).WithBlockTime(time.Now())

	err = k.CreateDeveloper(sdkCtx, developerAddr, true)
	require.NoError(t, err)

	initialAmount := uint64(100)
	err = k.AddUserSubscription(sdkCtx, developerAddr, userDid, initialAmount, 30)
	require.NoError(t, err)

	validAmount := uint64(200)
	zeroAmount := uint64(0)

	testCases := []struct {
		name      string
		input     *types.MsgUpdateUserSubscription
		expErr    bool
		expErrMsg string
	}{
		{
			name: "valid update user subscription - change amount",
			input: &types.MsgUpdateUserSubscription{
				Developer: validDeveloperAddr,
				UserDid:   userDid,
				Amount:    validAmount,
				Period:    30,
			},
			expErr: false,
		},
		{
			name: "valid update user subscription - change period",
			input: &types.MsgUpdateUserSubscription{
				Developer: validDeveloperAddr,
				UserDid:   userDid,
				Amount:    validAmount,
				Period:    60,
			},
			expErr: false,
		},
		{
			name: "valid update user subscription - change both",
			input: &types.MsgUpdateUserSubscription{
				Developer: validDeveloperAddr,
				UserDid:   userDid,
				Amount:    uint64(300),
				Period:    90,
			},
			expErr: false,
		},
		{
			name: "invalid developer address",
			input: &types.MsgUpdateUserSubscription{
				Developer: "invalid-developer-address",
				UserDid:   userDid,
				Amount:    validAmount,
				Period:    30,
			},
			expErr:    true,
			expErrMsg: "delegator address",
		},
		{
			name: "invalid user did",
			input: &types.MsgUpdateUserSubscription{
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
			input: &types.MsgUpdateUserSubscription{
				Developer: validDeveloperAddr,
				UserDid:   userDid,
				Amount:    zeroAmount,
				Period:    30,
			},
			expErr:    true,
			expErrMsg: "invalid amount",
		},
		{
			name: "non-existent subscription",
			input: &types.MsgUpdateUserSubscription{
				Developer: validDeveloperAddr,
				UserDid:   "did:key:nonexistent",
				Amount:    validAmount,
				Period:    30,
			},
			expErr:    true,
			expErrMsg: "update user subscription",
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

			resp, err := ms.UpdateUserSubscription(sdkCtx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				developerAddr := sdk.MustAccAddressFromBech32(tc.input.Developer)
				subscription := k.GetUserSubscription(sdkCtx, developerAddr, userDid)
				require.NotNil(t, subscription, "User subscription should exist after update")
				require.Equal(t, tc.input.Amount, subscription.CreditAmount, "Amount should match updated value")
				require.Equal(t, tc.input.Period, subscription.Period, "Period should match updated value")
			}
		})
	}
}

func TestMsgUpdateUserSubscription_ProgressiveUpdates(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	p := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, p))

	userDid := "did:key:alice"

	validDeveloperAddr := "vera1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s2dq2wz"
	developerAddr := sdk.MustAccAddressFromBech32(validDeveloperAddr)

	valAddr, err := sdk.ValAddressFromBech32("veravaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnduc4vm0")
	require.NoError(t, err)

	keepertest.InitializeDelegator(t, &k, sdkCtx, developerAddr, math.NewInt(10000))
	keepertest.InitializeValidator(t, k.GetStakingKeeper().(*stakingkeeper.Keeper), sdkCtx, valAddr, math.NewInt(1_000_000))
	sdkCtx = sdkCtx.WithBlockHeight(1).WithBlockTime(time.Now())

	err = k.CreateDeveloper(sdkCtx, developerAddr, true)
	require.NoError(t, err)

	initialAmount := uint64(100)
	err = k.AddUserSubscription(sdkCtx, developerAddr, userDid, initialAmount, 30)
	require.NoError(t, err)

	subscription := k.GetUserSubscription(sdkCtx, developerAddr, userDid)
	require.NotNil(t, subscription)
	require.Equal(t, initialAmount, subscription.CreditAmount)
	require.Equal(t, uint64(30), subscription.Period)

	msg := &types.MsgUpdateUserSubscription{
		Developer: validDeveloperAddr,
		UserDid:   userDid,
		Amount:    uint64(200),
		Period:    30,
	}

	resp, err := ms.UpdateUserSubscription(sdkCtx, msg)
	require.NoError(t, err)
	require.NotNil(t, resp)

	subscription = k.GetUserSubscription(sdkCtx, developerAddr, userDid)
	require.NotNil(t, subscription)
	require.Equal(t, uint64(200), subscription.CreditAmount)
	require.Equal(t, uint64(30), subscription.Period)

	msg.Amount = uint64(200)
	msg.Period = 60

	resp, err = ms.UpdateUserSubscription(sdkCtx, msg)
	require.NoError(t, err)
	require.NotNil(t, resp)

	subscription = k.GetUserSubscription(sdkCtx, developerAddr, userDid)
	require.NotNil(t, subscription)
	require.Equal(t, uint64(200), subscription.CreditAmount)
	require.Equal(t, uint64(60), subscription.Period)

	msg.Amount = uint64(500)
	msg.Period = 90

	resp, err = ms.UpdateUserSubscription(sdkCtx, msg)
	require.NoError(t, err)
	require.NotNil(t, resp)

	subscription = k.GetUserSubscription(sdkCtx, developerAddr, userDid)
	require.NotNil(t, subscription)
	require.Equal(t, uint64(500), subscription.CreditAmount)
	require.Equal(t, uint64(90), subscription.Period)
}

func TestMsgUpdateUserSubscription_NonExistentDeveloper(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	p := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, p))

	nonExistentDeveloperAddr := "vera1n34fvpteuanu2nx2a4hql4jvcrcnal3gqfmnpr"
	userDid := "did:key:alice"

	validAmount := uint64(100)
	msg := &types.MsgUpdateUserSubscription{
		Developer: nonExistentDeveloperAddr,
		UserDid:   userDid,
		Amount:    validAmount,
		Period:    30,
	}

	resp, err := ms.UpdateUserSubscription(sdkCtx, msg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "update user subscription")
	require.Nil(t, resp)
}
