package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	appparams "github.com/sourcenetwork/vera/app/params"
	"github.com/sourcenetwork/vera/x/core/types"
)

func TestMsgInvalidateJWS(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	issuerDID := "did:example:alice"
	issuerDID2 := "did:example:bob"
	authorizedAccount := "source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9"
	unauthorizedAccount := "source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et"
	bearerToken1 := "test.bearer.token1"
	bearerToken2 := "test.bearer.token2"
	bearerToken3 := "test.bearer.token3"
	bearerToken4 := "test.bearer.token4"
	tokenHash1 := types.HashJWSToken(bearerToken1)
	tokenHash2 := types.HashJWSToken(bearerToken2)
	tokenHash3 := types.HashJWSToken(bearerToken3)
	tokenHash4 := types.HashJWSToken(bearerToken4)

	// Store valid tokens
	err := k.StoreOrUpdateJWSToken(
		sdkCtx,
		bearerToken1,
		issuerDID,
		authorizedAccount,
		time.Now(),
		time.Now().Add(1*time.Hour),
	)
	require.NoError(t, err)

	err = k.StoreOrUpdateJWSToken(
		sdkCtx,
		bearerToken2,
		issuerDID2,
		authorizedAccount,
		time.Now(),
		time.Now().Add(1*time.Hour),
	)
	require.NoError(t, err)

	err = k.StoreOrUpdateJWSToken(
		sdkCtx,
		bearerToken3,
		issuerDID,
		authorizedAccount,
		time.Now(),
		time.Now().Add(1*time.Hour),
	)
	require.NoError(t, err)

	err = k.StoreOrUpdateJWSToken(
		sdkCtx,
		bearerToken4,
		issuerDID,
		authorizedAccount,
		time.Now(),
		time.Now().Add(1*time.Hour),
	)
	require.NoError(t, err)

	// Mark token3 as invalid
	err = k.UpdateJWSTokenStatus(sdkCtx, tokenHash3, types.JWSTokenStatus_STATUS_INVALID, "someone")
	require.NoError(t, err)

	testCases := []struct {
		name       string
		setupCtx   func(sdk.Context) sdk.Context
		input      *types.MsgInvalidateJWS
		expErr     bool
		expErrMsg  string
		verifyFunc func(*testing.T, *types.MsgInvalidateJWSResponse)
	}{
		{
			name: "valid - authorized account invalidates token",
			setupCtx: func(ctx sdk.Context) sdk.Context {
				return ctx
			},
			input: &types.MsgInvalidateJWS{
				Creator:   authorizedAccount,
				TokenHash: tokenHash1,
			},
			expErr: false,
			verifyFunc: func(t *testing.T, resp *types.MsgInvalidateJWSResponse) {
				require.True(t, resp.Success)
				record, found, err := k.GetJWSToken(sdkCtx, tokenHash1)
				require.NoError(t, err)
				require.True(t, found)
				require.Equal(t, types.JWSTokenStatus_STATUS_INVALID, record.Status)
				require.Equal(t, authorizedAccount, record.InvalidatedBy)
				require.NotNil(t, record.InvalidatedAt)
			},
		},
		{
			name: "valid - matching DID in context invalidates token",
			setupCtx: func(ctx sdk.Context) sdk.Context {
				return ctx.WithValue(appparams.ExtractedDIDContextKey, issuerDID2)
			},
			input: &types.MsgInvalidateJWS{
				Creator:   unauthorizedAccount,
				TokenHash: tokenHash2,
			},
			expErr: false,
			verifyFunc: func(t *testing.T, resp *types.MsgInvalidateJWSResponse) {
				require.True(t, resp.Success)
				record, found, err := k.GetJWSToken(sdkCtx, tokenHash2)
				require.NoError(t, err)
				require.True(t, found)
				require.Equal(t, types.JWSTokenStatus_STATUS_INVALID, record.Status)
			},
		},
		{
			name: "invalid - token not found",
			setupCtx: func(ctx sdk.Context) sdk.Context {
				return ctx
			},
			input: &types.MsgInvalidateJWS{
				Creator:   authorizedAccount,
				TokenHash: "nonexistenthash",
			},
			expErr:    true,
			expErrMsg: "token not found",
		},
		{
			name: "invalid - token already invalid",
			setupCtx: func(ctx sdk.Context) sdk.Context {
				return ctx
			},
			input: &types.MsgInvalidateJWS{
				Creator:   authorizedAccount,
				TokenHash: tokenHash3,
			},
			expErr:    true,
			expErrMsg: "already invalid",
		},
		{
			name: "invalid - unauthorized (neither DID nor account matches)",
			setupCtx: func(ctx sdk.Context) sdk.Context {
				return ctx.WithValue(appparams.ExtractedDIDContextKey, "did:example:different")
			},
			input: &types.MsgInvalidateJWS{
				Creator:   unauthorizedAccount,
				TokenHash: tokenHash4,
			},
			expErr:    true,
			expErrMsg: "not authorized to invalidate",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testCtx := tc.setupCtx(sdkCtx)

			resp, err := ms.InvalidateJWS(testCtx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				if tc.verifyFunc != nil {
					tc.verifyFunc(t, resp)
				}
			}
		})
	}
}

func TestMsgInvalidateJWS_Idempotency(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	issuerDID := "did:example:charlie"
	authorizedAccount := "source18jtkvj0995fy7lggqayg2f5syna92ndq5mkuv4"
	bearerToken := "test.bearer.charlie"
	tokenHash := types.HashJWSToken(bearerToken)

	err := k.StoreOrUpdateJWSToken(
		sdkCtx,
		bearerToken,
		issuerDID,
		authorizedAccount,
		time.Now(),
		time.Now().Add(1*time.Hour),
	)
	require.NoError(t, err)

	msg := &types.MsgInvalidateJWS{
		Creator:   authorizedAccount,
		TokenHash: tokenHash,
	}

	// First invalidation should succeed
	resp, err := ms.InvalidateJWS(sdkCtx, msg)
	require.NoError(t, err)
	require.True(t, resp.Success)

	// Second attempt should fail cause it is already invalid
	resp, err = ms.InvalidateJWS(sdkCtx, msg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "already invalid")
	require.Nil(t, resp)
}
