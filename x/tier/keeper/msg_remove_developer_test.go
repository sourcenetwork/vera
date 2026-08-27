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

func TestMsgRemoveDeveloper(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	p := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, p))

	validDeveloperAddr := "vera1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s2dq2wz"

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
				Developer: "vera1wjj5v5rlf57kayyeskncpu4hwev25ty697gcev",
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

	userDid := "did:key:alice"

	validDeveloperAddr := "vera1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s2dq2wz"
	developerAddr := sdk.MustAccAddressFromBech32(validDeveloperAddr)

	valAddr, err := sdk.ValAddressFromBech32("veravaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnduc4vm0")
	require.NoError(t, err)

	keepertest.InitializeDelegator(t, &k, sdkCtx, developerAddr, math.NewInt(5000))
	keepertest.InitializeValidator(t, k.GetStakingKeeper().(*stakingkeeper.Keeper), sdkCtx, valAddr, math.NewInt(1_000_000))

	sdkCtx = sdkCtx.WithBlockHeight(1).WithBlockTime(time.Now())

	err = k.CreateDeveloper(sdkCtx, developerAddr, true)
	require.NoError(t, err)

	amount := uint64(100)
	err = k.AddUserSubscription(sdkCtx, developerAddr, userDid, amount, 30)
	require.NoError(t, err)

	subscription := k.GetUserSubscription(sdkCtx, developerAddr, userDid)
	require.NotNil(t, subscription)

	msg := &types.MsgRemoveDeveloper{
		Developer: validDeveloperAddr,
	}

	resp, err := ms.RemoveDeveloper(sdkCtx, msg)
	require.NoError(t, err)
	require.NotNil(t, resp)

	developer := k.GetDeveloper(sdkCtx, developerAddr)
	require.Nil(t, developer, "Developer should not exist after removal")

	subscription = k.GetUserSubscription(sdkCtx, developerAddr, userDid)
	require.Nil(t, subscription, "User subscriptions should be removed when developer is removed")
}

func TestMsgRemoveDeveloper_AlreadyRemoved(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	p := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, p))

	validDeveloperAddr := "vera1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s2dq2wz"

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
