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

func TestMsgAddUserSubscription(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	p := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, p))

	validDeveloperAddr := "source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9"
	validUserAddr := "source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et"
	validUserAddr2 := "source1n34fvpteuanu2nx2a4hql4jvcrcnal3gsrjppy"

	developerAddr := sdk.MustAccAddressFromBech32(validDeveloperAddr)
	userAddr := sdk.MustAccAddressFromBech32(validUserAddr)
	userAddr2 := sdk.MustAccAddressFromBech32(validUserAddr2)

	keepertest.CreateAccount(t, &k, sdkCtx, developerAddr)
	keepertest.CreateAccount(t, &k, sdkCtx, userAddr)
	keepertest.CreateAccount(t, &k, sdkCtx, userAddr2)

	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	keepertest.InitializeDelegator(t, &k, sdkCtx, developerAddr, math.NewInt(5000))
	keepertest.InitializeValidator(t, k.GetStakingKeeper().(*stakingkeeper.Keeper), sdkCtx, valAddr, math.NewInt(1_000_000))

	sdkCtx = sdkCtx.WithBlockHeight(1).WithBlockTime(time.Now())

	err = k.CreateDeveloper(ctx, developerAddr, true)
	require.NoError(t, err)

	validAmount := sdk.NewCoin(appparams.MicroCreditDenom, math.NewInt(100))
	zeroAmount := sdk.NewCoin(appparams.MicroCreditDenom, math.ZeroInt())
	negativeAmount := sdk.Coin{
		Denom:  appparams.MicroCreditDenom,
		Amount: math.NewInt(-100),
	}

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
				User:      validUserAddr,
				Amount:    validAmount,
				Period:    30,
			},
			expErr: false,
		},
		{
			name: "valid add user subscription with zero period",
			input: &types.MsgAddUserSubscription{
				Developer: validDeveloperAddr,
				User:      "source1n34fvpteuanu2nx2a4hql4jvcrcnal3gsrjppy",
				Amount:    validAmount,
				Period:    0,
			},
			expErr: false,
		},
		{
			name: "invalid developer address",
			input: &types.MsgAddUserSubscription{
				Developer: "invalid-developer-address",
				User:      validUserAddr,
				Amount:    validAmount,
				Period:    30,
			},
			expErr:    true,
			expErrMsg: "delegator address",
		},
		{
			name: "invalid user address",
			input: &types.MsgAddUserSubscription{
				Developer: validDeveloperAddr,
				User:      "invalid-user-address",
				Amount:    validAmount,
				Period:    30,
			},
			expErr:    true,
			expErrMsg: "delegator address",
		},
		{
			name: "developer and user are the same",
			input: &types.MsgAddUserSubscription{
				Developer: validDeveloperAddr,
				User:      validDeveloperAddr,
				Amount:    validAmount,
				Period:    30,
			},
			expErr:    true,
			expErrMsg: "developer and user cannot be the same address",
		},
		{
			name: "zero amount",
			input: &types.MsgAddUserSubscription{
				Developer: validDeveloperAddr,
				User:      validUserAddr,
				Amount:    zeroAmount,
				Period:    30,
			},
			expErr:    true,
			expErrMsg: "invalid amount",
		},
		{
			name: "negative amount",
			input: &types.MsgAddUserSubscription{
				Developer: validDeveloperAddr,
				User:      validUserAddr,
				Amount:    negativeAmount,
				Period:    30,
			},
			expErr:    true,
			expErrMsg: "invalid amount",
		},
		{
			name: "non-existent developer",
			input: &types.MsgAddUserSubscription{
				Developer: "source1n34fvpteuanu2nx2a4hql4jvcrcnal3gsrjppy",
				User:      validUserAddr,
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
				userAddr := sdk.MustAccAddressFromBech32(tc.input.User)
				subscription := k.GetUserSubscription(sdkCtx, developerAddr, userAddr)
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

	err = k.CreateDeveloper(ctx, developerAddr, true)
	require.NoError(t, err)

	validAmount := sdk.NewCoin(appparams.MicroCreditDenom, math.NewInt(100))

	msg := &types.MsgAddUserSubscription{
		Developer: validDeveloperAddr,
		User:      validUserAddr,
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

	err = k.CreateDeveloper(ctx, developerAddr, true)
	require.NoError(t, err)

	validAmount := sdk.NewCoin(appparams.MicroCreditDenom, math.NewInt(100))

	msg1 := &types.MsgAddUserSubscription{
		Developer: validDeveloperAddr,
		User:      user1Addr,
		Amount:    validAmount,
		Period:    30,
	}

	resp, err := ms.AddUserSubscription(sdkCtx, msg1)
	require.NoError(t, err)
	require.NotNil(t, resp)

	msg2 := &types.MsgAddUserSubscription{
		Developer: validDeveloperAddr,
		User:      user2Addr,
		Amount:    sdk.NewCoin(appparams.MicroCreditDenom, math.NewInt(200)),
		Period:    60,
	}

	resp, err = ms.AddUserSubscription(sdkCtx, msg2)
	require.NoError(t, err)
	require.NotNil(t, resp)

	subscription1 := k.GetUserSubscription(sdkCtx, developerAddr, user1Address)
	require.NotNil(t, subscription1)
	require.Equal(t, validAmount, subscription1.CreditAmount)
	require.Equal(t, uint64(30), subscription1.Period)

	subscription2 := k.GetUserSubscription(sdkCtx, developerAddr, user2Address)
	require.NotNil(t, subscription2)
	require.Equal(t, sdk.NewCoin(appparams.MicroCreditDenom, math.NewInt(200)), subscription2.CreditAmount)
	require.Equal(t, uint64(60), subscription2.Period)
}
