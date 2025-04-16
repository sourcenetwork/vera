package keeper

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	"github.com/sourcenetwork/sourcehub/x/bulletin/types"
)

func TestMsgServer(t *testing.T) {
	k, ctx := setupKeeper(t)
	require.NotNil(t, ctx)
	require.NotEmpty(t, k)
}

func TestMsgUpdateParams(t *testing.T) {
	k, ctx := setupKeeper(t)
	params := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, params))
	wctx := sdk.UnwrapSDKContext(ctx)

	// default params
	testCases := []struct {
		name      string
		input     *types.MsgUpdateParams
		expErr    bool
		expErrMsg string
	}{
		{
			name: "invalid authority",
			input: &types.MsgUpdateParams{
				Authority: "invalid",
				Params:    params,
			},
			expErr:    true,
			expErrMsg: "invalid authority",
		},
		{
			name: "send enabled param",
			input: &types.MsgUpdateParams{
				Authority: k.GetAuthority(),
				Params:    types.Params{},
			},
			expErr: false,
		},
		{
			name: "all good",
			input: &types.MsgUpdateParams{
				Authority: k.GetAuthority(),
				Params:    params,
			},
			expErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := k.UpdateParams(wctx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMsgRegisterNamespace(t *testing.T) {
	k, ctx := setupKeeper(t)

	p := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, p))

	pubKey := secp256k1.GenPrivKey().PubKey()
	addr := sdk.AccAddress(pubKey.Address())
	baseAcc := authtypes.NewBaseAccount(addr, pubKey, 1, 1)
	k.accountKeeper.SetAccount(ctx, baseAcc)

	namespace := "ns1"

	testCases := []struct {
		name      string
		input     *types.MsgRegisterNamespace
		setup     func()
		expErr    bool
		expErrMsg string
	}{
		{
			name:      "register namespace (error: invalid creator address)",
			input:     &types.MsgRegisterNamespace{},
			setup:     func() {},
			expErr:    true,
			expErrMsg: "invalid creator address",
		},
		{
			name: "register namespace (error: invalid namespace id)",
			input: &types.MsgRegisterNamespace{
				Creator: baseAcc.Address,
			},
			setup:     func() {},
			expErr:    true,
			expErrMsg: "invalid namespace id",
		},
		{
			name: "register namespace (error: invalid policy id)",
			input: &types.MsgRegisterNamespace{
				Creator:   baseAcc.Address,
				Namespace: namespace,
			},
			setup:     func() {},
			expErr:    true,
			expErrMsg: "invalid policy id",
		},
		{
			name: "register namespace (error: fetching capability for policy)",
			input: &types.MsgRegisterNamespace{
				Creator:   baseAcc.Address,
				Namespace: namespace,
			},
			setup: func() {
				k.SetPolicyId(ctx, "policy1")
			},
			expErr:    true,
			expErrMsg: "fetching capability for policy",
		},
		{
			name: "register namespace (error: invalid policy id)",
			input: &types.MsgRegisterNamespace{
				Creator:   baseAcc.Address,
				Namespace: namespace,
			},
			setup: func() {
				k.SetPolicyId(ctx, "")
			},
			expErr:    true,
			expErrMsg: "invalid policy id",
		},
		{
			name: "register namespace (no error)",
			input: &types.MsgRegisterNamespace{
				Creator:   baseAcc.Address,
				Namespace: namespace,
			},
			setup: func() {
				setupTestPolicy(t, ctx, k)
			},
			expErr: false,
		},
		{
			name: "register namespace (error: namespace exists)",
			input: &types.MsgRegisterNamespace{
				Creator:   baseAcc.Address,
				Namespace: namespace,
			},
			setup:     func() {},
			expErr:    true,
			expErrMsg: "namespace already exists",
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

			tc.setup()

			_, err = k.RegisterNamespace(ctx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMsgCreatePost(t *testing.T) {
	k, ctx := setupKeeper(t)

	p := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, p))

	pubKey := secp256k1.GenPrivKey().PubKey()
	addr := sdk.AccAddress(pubKey.Address())
	baseAcc := authtypes.NewBaseAccount(addr, pubKey, 1, 1)
	k.accountKeeper.SetAccount(ctx, baseAcc)

	pubKey2 := secp256k1.GenPrivKey().PubKey()
	addr2 := sdk.AccAddress(pubKey2.Address())
	baseAcc2 := authtypes.NewBaseAccount(addr2, pubKey2, 2, 1)
	k.accountKeeper.SetAccount(ctx, baseAcc2)

	namespace := "ns1"

	testCases := []struct {
		name      string
		input     *types.MsgCreatePost
		setup     func()
		expErr    bool
		expErrMsg string
	}{
		{
			name:      "create post (error: nvalid creator address)",
			input:     &types.MsgCreatePost{},
			setup:     func() {},
			expErr:    true,
			expErrMsg: "nvalid creator address",
		},
		{
			name: "create post (error: invalid namespace id)",
			input: &types.MsgCreatePost{
				Creator: baseAcc.Address,
				Payload: []byte("post123"),
				Proof:   []byte("proof456"),
			},
			setup:     func() {},
			expErr:    true,
			expErrMsg: "invalid namespace id",
		},
		{
			name: "create post (error: no payload)",
			input: &types.MsgCreatePost{
				Creator:   baseAcc.Address,
				Namespace: namespace,
				Proof:     []byte("proof456"),
			},
			setup:     func() {},
			expErr:    true,
			expErrMsg: "invalid post payload",
		},
		{
			name: "create post (error: no proof)",
			input: &types.MsgCreatePost{
				Creator:   baseAcc.Address,
				Namespace: namespace,
				Payload:   []byte("post123"),
			},
			setup:     func() {},
			expErr:    true,
			expErrMsg: "invalid post proof",
		},
		{
			name: "create post (error: no policy)",
			input: &types.MsgCreatePost{
				Creator:   baseAcc.Address,
				Namespace: namespace,
				Payload:   []byte("post123"),
				Proof:     []byte("proof456"),
			},
			setup:     func() {},
			expErr:    true,
			expErrMsg: "invalid policy id",
		},
		{
			name: "create post (error: no namespace)",
			input: &types.MsgCreatePost{
				Creator:   baseAcc.Address,
				Namespace: namespace,
				Payload:   []byte("post123"),
				Proof:     []byte("proof456"),
			},
			setup: func() {
				k.SetPolicyId(ctx, "policy1")
			},
			expErr:    true,
			expErrMsg: "namespace not found",
		},
		{
			name: "create post (no error)",
			input: &types.MsgCreatePost{
				Creator:   baseAcc.Address,
				Namespace: namespace,
				Payload:   []byte("post123"),
				Proof:     []byte("proof456"),
			},
			setup: func() {
				setupTestPolicy(t, ctx, k)

				_, err := k.RegisterNamespace(ctx, &types.MsgRegisterNamespace{
					Creator:   baseAcc.Address,
					Namespace: namespace,
				})
				require.NoError(t, err)
			},
			expErr: false,
		},
		{
			name: "create post (error: post already exists)",
			input: &types.MsgCreatePost{
				Creator:   baseAcc.Address,
				Namespace: namespace,
				Payload:   []byte("post123"),
				Proof:     []byte("proof456"),
			},
			setup:     func() {},
			expErr:    true,
			expErrMsg: "post already exists",
		},
		{
			name: "create post (error: unauthorized)",
			input: &types.MsgCreatePost{
				Creator:   baseAcc2.Address,
				Namespace: namespace,
				Payload:   []byte("post1234"),
				Proof:     []byte("proof4567"),
			},
			setup:     func() {},
			expErr:    true,
			expErrMsg: "expected authorized account as a post creator",
		},
		{
			name: "create post from collaborator (no error)",
			input: &types.MsgCreatePost{
				Creator:   baseAcc2.Address,
				Namespace: namespace,
				Payload:   []byte("post1234"),
				Proof:     []byte("proof4567"),
			},
			setup: func() {
				_, err := k.AddCollaborator(ctx, &types.MsgAddCollaborator{
					Creator:      baseAcc.Address,
					Collaborator: baseAcc2.Address,
					Namespace:    namespace,
				})
				require.NoError(t, err)
			},
			expErr: false,
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

			tc.setup()

			_, err = k.CreatePost(ctx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMsgAddCollaborator(t *testing.T) {
	k, ctx := setupKeeper(t)

	p := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, p))

	pubKey := secp256k1.GenPrivKey().PubKey()
	addr := sdk.AccAddress(pubKey.Address())
	baseAcc := authtypes.NewBaseAccount(addr, pubKey, 1, 1)
	k.accountKeeper.SetAccount(ctx, baseAcc)

	pubKey2 := secp256k1.GenPrivKey().PubKey()
	addr2 := sdk.AccAddress(pubKey2.Address())
	baseAcc2 := authtypes.NewBaseAccount(addr2, pubKey2, 2, 1)
	k.accountKeeper.SetAccount(ctx, baseAcc2)

	pubKey3 := secp256k1.GenPrivKey().PubKey()
	addr3 := sdk.AccAddress(pubKey3.Address())
	baseAcc3 := authtypes.NewBaseAccount(addr3, pubKey3, 3, 1)
	k.accountKeeper.SetAccount(ctx, baseAcc3)

	namespace := "ns1"

	testCases := []struct {
		name      string
		input     *types.MsgAddCollaborator
		setup     func()
		expErr    bool
		expErrMsg string
	}{
		{
			name:      "add collaborator (error: invalid creator address)",
			input:     &types.MsgAddCollaborator{},
			setup:     func() {},
			expErr:    true,
			expErrMsg: "invalid creator address",
		},
		{
			name: "add collaborator (error: invalid namespace id)",
			input: &types.MsgAddCollaborator{
				Creator: baseAcc.Address,
			},
			setup:     func() {},
			expErr:    true,
			expErrMsg: "invalid namespace id",
		},
		{
			name: "add collaborator (error: invalid collaborator address)",
			input: &types.MsgAddCollaborator{
				Creator:   baseAcc.Address,
				Namespace: namespace,
			},
			setup:     func() {},
			expErr:    true,
			expErrMsg: "invalid collaborator address",
		},
		{
			name: "add collaborator (error: invalid policy id)",
			input: &types.MsgAddCollaborator{
				Creator:      baseAcc.Address,
				Collaborator: baseAcc2.Address,
				Namespace:    namespace,
			},
			setup:     func() {},
			expErr:    true,
			expErrMsg: "invalid policy id",
		},
		{
			name: "add collaborator (error: namespace not found)",
			input: &types.MsgAddCollaborator{
				Creator:      baseAcc.Address,
				Collaborator: baseAcc2.Address,
				Namespace:    namespace,
			},
			setup: func() {
				k.SetPolicyId(ctx, "policy1")
			},
			expErr:    true,
			expErrMsg: "namespace not found",
		},
		{
			name: "add collaborator (no error)",
			input: &types.MsgAddCollaborator{
				Creator:      baseAcc.Address,
				Collaborator: baseAcc2.Address,
				Namespace:    namespace,
			},
			setup: func() {
				setupTestPolicy(t, ctx, k)

				_, err := k.RegisterNamespace(ctx, &types.MsgRegisterNamespace{
					Creator:   baseAcc.Address,
					Namespace: namespace,
				})
				require.NoError(t, err)
			},
			expErr: false,
		},
		{
			name: "add collaborator (error: collaborator already exists)",
			input: &types.MsgAddCollaborator{
				Creator:      baseAcc.Address,
				Collaborator: baseAcc2.Address,
				Namespace:    namespace,
			},
			setup:     func() {},
			expErr:    true,
			expErrMsg: "collaborator already exists",
		},
		{
			name: "add collaborator (error: unauthorized)",
			input: &types.MsgAddCollaborator{
				Creator:      baseAcc2.Address,
				Collaborator: baseAcc3.Address,
				Namespace:    namespace,
			},
			setup:     func() {},
			expErr:    true,
			expErrMsg: "actor is not a manager of relation",
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

			tc.setup()

			_, err = k.AddCollaborator(ctx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMsgRemoveCollaborator(t *testing.T) {
	k, ctx := setupKeeper(t)

	p := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, p))

	pubKey := secp256k1.GenPrivKey().PubKey()
	addr := sdk.AccAddress(pubKey.Address())
	baseAcc := authtypes.NewBaseAccount(addr, pubKey, 1, 1)
	k.accountKeeper.SetAccount(ctx, baseAcc)

	pubKey2 := secp256k1.GenPrivKey().PubKey()
	addr2 := sdk.AccAddress(pubKey2.Address())
	baseAcc2 := authtypes.NewBaseAccount(addr2, pubKey2, 2, 1)
	k.accountKeeper.SetAccount(ctx, baseAcc2)

	pubKey3 := secp256k1.GenPrivKey().PubKey()
	addr3 := sdk.AccAddress(pubKey3.Address())
	baseAcc3 := authtypes.NewBaseAccount(addr3, pubKey3, 3, 1)
	k.accountKeeper.SetAccount(ctx, baseAcc3)

	namespace := "ns1"

	testCases := []struct {
		name      string
		input     *types.MsgRemoveCollaborator
		setup     func()
		expErr    bool
		expErrMsg string
	}{
		{
			name:      "remove collaborator (error: invalid creator address)",
			input:     &types.MsgRemoveCollaborator{},
			setup:     func() {},
			expErr:    true,
			expErrMsg: "invalid creator address",
		},
		{
			name: "remove collaborator (error: invalid namespace id)",
			input: &types.MsgRemoveCollaborator{
				Creator: baseAcc.Address,
			},
			setup:     func() {},
			expErr:    true,
			expErrMsg: "invalid namespace id",
		},
		{
			name: "remove collaborator (error: invalid collaborator address)",
			input: &types.MsgRemoveCollaborator{
				Creator:   baseAcc.Address,
				Namespace: namespace,
			},
			setup:     func() {},
			expErr:    true,
			expErrMsg: "invalid collaborator address",
		},
		{
			name: "remove collaborator (error: invalid policy id)",
			input: &types.MsgRemoveCollaborator{
				Creator:      baseAcc.Address,
				Collaborator: baseAcc2.Address,
				Namespace:    namespace,
			},
			setup:     func() {},
			expErr:    true,
			expErrMsg: "invalid policy id",
		},
		{
			name: "remove collaborator (error: namespace not found)",
			input: &types.MsgRemoveCollaborator{
				Creator:      baseAcc.Address,
				Collaborator: baseAcc2.Address,
				Namespace:    namespace,
			},
			setup: func() {
				k.SetPolicyId(ctx, "policy1")
			},
			expErr:    true,
			expErrMsg: "namespace not found",
		},
		{
			name: "remove collaborator (no error)",
			input: &types.MsgRemoveCollaborator{
				Creator:      baseAcc.Address,
				Collaborator: baseAcc2.Address,
				Namespace:    namespace,
			},
			setup: func() {
				setupTestPolicy(t, ctx, k)

				_, err := k.RegisterNamespace(ctx, &types.MsgRegisterNamespace{
					Creator:   baseAcc.Address,
					Namespace: namespace,
				})
				require.NoError(t, err)

				_, err = k.AddCollaborator(ctx, &types.MsgAddCollaborator{
					Creator:      baseAcc.Address,
					Collaborator: baseAcc2.Address,
					Namespace:    namespace,
				})
				require.NoError(t, err)
			},
			expErr: false,
		},
		{
			name: "remove collaborator (error: collaborator not found)",
			input: &types.MsgRemoveCollaborator{
				Creator:      baseAcc.Address,
				Collaborator: baseAcc2.Address,
				Namespace:    namespace,
			},
			setup:     func() {},
			expErr:    true,
			expErrMsg: "collaborator not found",
		},
		{
			name: "remove collaborator (error: unauthorized)",
			input: &types.MsgRemoveCollaborator{
				Creator:      baseAcc2.Address,
				Collaborator: baseAcc3.Address,
				Namespace:    namespace,
			},
			setup: func() {
				_, err := k.AddCollaborator(ctx, &types.MsgAddCollaborator{
					Creator:      baseAcc.Address,
					Collaborator: baseAcc2.Address,
					Namespace:    namespace,
				})
				require.NoError(t, err)

				_, err = k.AddCollaborator(ctx, &types.MsgAddCollaborator{
					Creator:      baseAcc.Address,
					Collaborator: baseAcc3.Address,
					Namespace:    namespace,
				})
				require.NoError(t, err)
			},
			expErr:    true,
			expErrMsg: "actor is not a manager of relation",
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

			tc.setup()

			_, err = k.RemoveCollaborator(ctx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
