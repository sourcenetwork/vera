package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/sourcenetwork/vera/x/tier/types"
)

func TestMsgCreateDeveloper(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	p := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, p))

	validDeveloperAddr := "source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9"

	testCases := []struct {
		name      string
		input     *types.MsgCreateDeveloper
		expErr    bool
		expErrMsg string
	}{
		{
			name: "valid create developer with auto lock enabled",
			input: &types.MsgCreateDeveloper{
				Developer:       validDeveloperAddr,
				AutoLockEnabled: true,
			},
			expErr: false,
		},
		{
			name: "valid create developer with auto lock disabled",
			input: &types.MsgCreateDeveloper{
				Developer:       validDeveloperAddr,
				AutoLockEnabled: false,
			},
			expErr: false,
		},
		{
			name: "invalid developer address",
			input: &types.MsgCreateDeveloper{
				Developer:       "invalid-developer-address",
				AutoLockEnabled: true,
			},
			expErr:    true,
			expErrMsg: "delegator address",
		},
		{
			name: "empty developer address",
			input: &types.MsgCreateDeveloper{
				Developer:       "",
				AutoLockEnabled: true,
			},
			expErr:    true,
			expErrMsg: "delegator address",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			k, ms, ctx = setupMsgServer(t)
			sdkCtx = sdk.UnwrapSDKContext(ctx)
			require.NoError(t, k.SetParams(ctx, p))

			err := tc.input.ValidateBasic()
			if err != nil {
				if tc.expErr {
					require.Contains(t, err.Error(), tc.expErrMsg)
					return
				}
				t.Fatalf("unexpected error in ValidateBasic: %v", err)
			}

			resp, err := ms.CreateDeveloper(sdkCtx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				developerAddr := sdk.MustAccAddressFromBech32(tc.input.Developer)
				developer := k.GetDeveloper(sdkCtx, developerAddr)
				require.NotNil(t, developer, "Developer should exist after creation")
				require.Equal(t, tc.input.AutoLockEnabled, developer.AutoLockEnabled, "AutoLockEnabled should match input")
			}
		})
	}
}

func TestMsgCreateDeveloper_DuplicateDeveloper(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	p := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, p))

	validDeveloperAddr := "source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9"

	msg := &types.MsgCreateDeveloper{
		Developer:       validDeveloperAddr,
		AutoLockEnabled: true,
	}

	resp, err := ms.CreateDeveloper(sdkCtx, msg)
	require.NoError(t, err)
	require.NotNil(t, resp)

	resp, err = ms.CreateDeveloper(sdkCtx, msg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "create developer")
}
