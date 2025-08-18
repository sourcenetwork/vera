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

func TestMsgRemoveDeveloper(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	p := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, p))

	validDeveloperAddr := "source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9"

	developerAddr := sdk.MustAccAddressFromBech32(validDeveloperAddr)
	err := k.CreateDeveloper(ctx, developerAddr, true)
	require.NoError(t, err)

	testCases := []struct {
		name      string
		input     *types.MsgRemoveDeveloper
		expErr    bool
		expErrMsg string
	}{
		{
			name: "valid remove developer",
			input: &types.MsgRemoveDeveloper{
				Developer: validDeveloperAddr,
			},
			expErr: false,
		},
		{
			name: "invalid developer address",
			input: &types.MsgRemoveDeveloper{
				Developer: "invalid-developer-address",
			},
			expErr:    true,
			expErrMsg: "delegator address",
		},
		{
			name: "empty developer address",
			input: &types.MsgRemoveDeveloper{
				Developer: "",
			},
			expErr:    true,
			expErrMsg: "delegator address",
		},
		{
			name: "non-existent developer",
			input: &types.MsgRemoveDeveloper{
				Developer: "source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et",
			},
			expErr:    true,
			expErrMsg: "remove developer",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.name == "valid remove developer" {
				developer := k.GetDeveloper(sdkCtx, developerAddr)
				if developer == nil {
					err := k.CreateDeveloper(ctx, developerAddr, true)
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

			resp, err := ms.RemoveDeveloper(sdkCtx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				developerAddr := sdk.MustAccAddressFromBech32(tc.input.Developer)
				developer := k.GetDeveloper(sdkCtx, developerAddr)
				require.Nil(t, developer, "Developer should not exist after removal")
			}
		})
	}
}

func TestMsgRemoveDeveloper_WithSubscriptions(t *testing.T) {
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

	subscription := k.GetUserSubscription(sdkCtx, developerAddr, userAddr)
	require.NotNil(t, subscription)

	msg := &types.MsgRemoveDeveloper{
		Developer: validDeveloperAddr,
	}

	resp, err := ms.RemoveDeveloper(sdkCtx, msg)
	require.NoError(t, err)
	require.NotNil(t, resp)

	developer := k.GetDeveloper(sdkCtx, developerAddr)
	require.Nil(t, developer, "Developer should not exist after removal")

	subscription = k.GetUserSubscription(sdkCtx, developerAddr, userAddr)
	require.Nil(t, subscription, "User subscriptions should be removed when developer is removed")
}

func TestMsgRemoveDeveloper_AlreadyRemoved(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	p := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, p))

	validDeveloperAddr := "source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9"

	developerAddr := sdk.MustAccAddressFromBech32(validDeveloperAddr)
	err := k.CreateDeveloper(ctx, developerAddr, true)
	require.NoError(t, err)

	err = k.RemoveDeveloper(ctx, developerAddr)
	require.NoError(t, err)

	msg := &types.MsgRemoveDeveloper{
		Developer: validDeveloperAddr,
	}

	resp, err := ms.RemoveDeveloper(sdkCtx, msg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "remove developer")
	require.Nil(t, resp)
}
