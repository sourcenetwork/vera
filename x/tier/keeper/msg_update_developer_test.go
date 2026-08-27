package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/sourcenetwork/vera/x/tier/types"
)

func TestMsgUpdateDeveloper(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	p := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, p))

	validDeveloperAddr := "vera1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s2dq2wz"

	developerAddr := sdk.MustAccAddressFromBech32(validDeveloperAddr)
	err := k.CreateDeveloper(ctx, developerAddr, false)
	require.NoError(t, err)

	testCases := []struct {
		name      string
		input     *types.MsgUpdateDeveloper
		expErr    bool
		expErrMsg string
	}{
		{
			name: "valid update developer - enable auto lock",
			input: &types.MsgUpdateDeveloper{
				Developer:       validDeveloperAddr,
				AutoLockEnabled: true,
			},
			expErr: false,
		},
		{
			name: "valid update developer - disable auto lock",
			input: &types.MsgUpdateDeveloper{
				Developer:       validDeveloperAddr,
				AutoLockEnabled: false,
			},
			expErr: false,
		},
		{
			name: "invalid developer address",
			input: &types.MsgUpdateDeveloper{
				Developer:       "invalid-developer-address",
				AutoLockEnabled: true,
			},
			expErr:    true,
			expErrMsg: "delegator address",
		},
		{
			name: "empty developer address",
			input: &types.MsgUpdateDeveloper{
				Developer:       "",
				AutoLockEnabled: true,
			},
			expErr:    true,
			expErrMsg: "delegator address",
		},
		{
			name: "non-existent developer (should fail)",
			input: &types.MsgUpdateDeveloper{
				Developer:       "vera1wjj5v5rlf57kayyeskncpu4hwev25ty697gcev",
				AutoLockEnabled: true,
			},
			expErr:    true,
			expErrMsg: "does not exist",
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

			resp, err := ms.UpdateDeveloper(sdkCtx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				developerAddr := sdk.MustAccAddressFromBech32(tc.input.Developer)
				developer := k.GetDeveloper(sdkCtx, developerAddr)
				require.NotNil(t, developer, "Developer should exist after update")
				require.Equal(t, tc.input.AutoLockEnabled, developer.AutoLockEnabled, "AutoLockEnabled should match updated value")
			}
		})
	}
}

func TestMsgUpdateDeveloper_ToggleAutoLock(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	p := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, p))

	validDeveloperAddr := "vera1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s2dq2wz"

	developerAddr := sdk.MustAccAddressFromBech32(validDeveloperAddr)
	err := k.CreateDeveloper(ctx, developerAddr, false)
	require.NoError(t, err)

	developer := k.GetDeveloper(sdkCtx, developerAddr)
	require.NotNil(t, developer)
	require.False(t, developer.AutoLockEnabled)

	msg := &types.MsgUpdateDeveloper{
		Developer:       validDeveloperAddr,
		AutoLockEnabled: true,
	}

	resp, err := ms.UpdateDeveloper(sdkCtx, msg)
	require.NoError(t, err)
	require.NotNil(t, resp)

	developer = k.GetDeveloper(sdkCtx, developerAddr)
	require.NotNil(t, developer)
	require.True(t, developer.AutoLockEnabled)

	msg.AutoLockEnabled = false
	resp, err = ms.UpdateDeveloper(sdkCtx, msg)
	require.NoError(t, err)
	require.NotNil(t, resp)

	developer = k.GetDeveloper(sdkCtx, developerAddr)
	require.NotNil(t, developer)
	require.False(t, developer.AutoLockEnabled)
}
