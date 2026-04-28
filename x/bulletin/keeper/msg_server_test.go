package keeper

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	"github.com/sourcenetwork/sourcehub/x/bulletin/types"
)

func TestMsgUpdatePost(t *testing.T) {
	k, ctx := setupKeeper(t)
	require.NoError(t, k.SetParams(ctx, types.DefaultParams()))

	ownerKey := secp256k1.GenPrivKey().PubKey()
	owner := authtypes.NewBaseAccount(sdk.AccAddress(ownerKey.Address()), ownerKey, 1, 1)
	k.accountKeeper.SetAccount(ctx, owner)

	outsiderKey := secp256k1.GenPrivKey().PubKey()
	outsider := authtypes.NewBaseAccount(sdk.AccAddress(outsiderKey.Address()), outsiderKey, 2, 1)
	k.accountKeeper.SetAccount(ctx, outsider)

	collabKey := secp256k1.GenPrivKey().PubKey()
	collab := authtypes.NewBaseAccount(sdk.AccAddress(collabKey.Address()), collabKey, 3, 1)
	k.accountKeeper.SetAccount(ctx, collab)

	setupTestPolicy(t, ctx, k)

	namespace := "ns1"
	namespaceId := getNamespaceId(namespace)
	originalPayload := []byte("hello world")

	_, err := k.RegisterNamespace(ctx, &types.MsgRegisterNamespace{
		Creator:   owner.Address,
		Namespace: namespace,
	})
	require.NoError(t, err)

	_, err = k.CreatePost(ctx, &types.MsgCreatePost{
		Creator:   owner.Address,
		Namespace: namespace,
		Payload:   originalPayload,
	})
	require.NoError(t, err)

	postId := types.GeneratePostId(namespaceId, originalPayload)

	t.Run("validate basic rejects empty fields", func(t *testing.T) {
		cases := []struct {
			msg    *types.MsgUpdatePost
			errMsg string
		}{
			{&types.MsgUpdatePost{}, "invalid creator address"},
			{&types.MsgUpdatePost{Creator: owner.Address}, "invalid namespace id"},
			{&types.MsgUpdatePost{Creator: owner.Address, Namespace: namespace}, "post not found"},
			{&types.MsgUpdatePost{Creator: owner.Address, Namespace: namespace, PostId: postId}, "invalid post payload"},
		}
		for _, c := range cases {
			err := c.msg.ValidateBasic()
			require.Error(t, err)
			require.Contains(t, err.Error(), c.errMsg)
		}
	})

	t.Run("namespace not found", func(t *testing.T) {
		_, err := k.UpdatePost(ctx, &types.MsgUpdatePost{
			Creator:   owner.Address,
			Namespace: "does-not-exist",
			PostId:    postId,
			Payload:   []byte("new"),
		})
		require.ErrorIs(t, err, types.ErrNamespaceNotFound)
	})

	t.Run("post not found", func(t *testing.T) {
		_, err := k.UpdatePost(ctx, &types.MsgUpdatePost{
			Creator:   owner.Address,
			Namespace: namespace,
			PostId:    "nonexistent-post-id",
			Payload:   []byte("new"),
		})
		require.ErrorIs(t, err, types.ErrPostNotFound)
	})

	t.Run("outsider without collaborator relation is rejected", func(t *testing.T) {
		_, err := k.UpdatePost(ctx, &types.MsgUpdatePost{
			Creator:   outsider.Address,
			Namespace: namespace,
			PostId:    postId,
			Payload:   []byte("nope"),
		})
		require.ErrorIs(t, err, types.ErrInvalidPostUpdater)

		stored := k.getPost(ctx, namespaceId, postId)
		require.NotNil(t, stored)
		require.Equal(t, originalPayload, stored.Payload)
	})

	t.Run("creator can update and post_id is preserved", func(t *testing.T) {
		newPayload := []byte("hello mars")
		_, err := k.UpdatePost(ctx, &types.MsgUpdatePost{
			Creator:   owner.Address,
			Namespace: namespace,
			PostId:    postId,
			Payload:   newPayload,
		})
		require.NoError(t, err)

		stored := k.getPost(ctx, namespaceId, postId)
		require.NotNil(t, stored)
		require.Equal(t, postId, stored.Id)
		require.Equal(t, newPayload, stored.Payload)
	})

	t.Run("collaborator can update via ACP policy", func(t *testing.T) {
		_, err := k.AddCollaborator(ctx, &types.MsgAddCollaborator{
			Creator:      owner.Address,
			Namespace:    namespace,
			Collaborator: collab.Address,
		})
		require.NoError(t, err)

		newPayload := []byte("hello venus")
		_, err = k.UpdatePost(ctx, &types.MsgUpdatePost{
			Creator:   collab.Address,
			Namespace: namespace,
			PostId:    postId,
			Payload:   newPayload,
		})
		require.NoError(t, err)

		stored := k.getPost(ctx, namespaceId, postId)
		require.NotNil(t, stored)
		require.Equal(t, postId, stored.Id)
		require.Equal(t, newPayload, stored.Payload)
	})

	t.Run("revoked collaborator can no longer update", func(t *testing.T) {
		// collab was granted in the previous subtest; revoke it now
		_, err := k.RemoveCollaborator(ctx, &types.MsgRemoveCollaborator{
			Creator:      owner.Address,
			Namespace:    namespace,
			Collaborator: collab.Address,
		})
		require.NoError(t, err)

		beforePayload := k.getPost(ctx, namespaceId, postId).Payload

		_, err = k.UpdatePost(ctx, &types.MsgUpdatePost{
			Creator:   collab.Address,
			Namespace: namespace,
			PostId:    postId,
			Payload:   []byte("should be rejected"),
		})
		require.ErrorIs(t, err, types.ErrInvalidPostUpdater)

		stored := k.getPost(ctx, namespaceId, postId)
		require.NotNil(t, stored)
		require.Equal(t, beforePayload, stored.Payload)
	})
}

// TestPolicy_CreatePostPermissionSurvives locks in that the `create_post`
// permission remains queryable via ACP with the `collaborator` relation.
// The keeper no longer gates CreatePost with this permission, but external
// consumers may still query it — this guards against accidental removal
// from the policy YAML.
func TestPolicy_CreatePostPermissionSurvives(t *testing.T) {
	k, ctx := setupKeeper(t)
	require.NoError(t, k.SetParams(ctx, types.DefaultParams()))

	ownerKey := secp256k1.GenPrivKey().PubKey()
	owner := authtypes.NewBaseAccount(sdk.AccAddress(ownerKey.Address()), ownerKey, 1, 1)
	k.accountKeeper.SetAccount(ctx, owner)

	collabKey := secp256k1.GenPrivKey().PubKey()
	collab := authtypes.NewBaseAccount(sdk.AccAddress(collabKey.Address()), collabKey, 2, 1)
	k.accountKeeper.SetAccount(ctx, collab)

	setupTestPolicy(t, ctx, k)

	namespace := "ns-create-perm"
	namespaceId := getNamespaceId(namespace)

	_, err := k.RegisterNamespace(ctx, &types.MsgRegisterNamespace{
		Creator:   owner.Address,
		Namespace: namespace,
	})
	require.NoError(t, err)

	_, err = k.AddCollaborator(ctx, &types.MsgAddCollaborator{
		Creator:      owner.Address,
		Namespace:    namespace,
		Collaborator: collab.Address,
	})
	require.NoError(t, err)

	collabDID, err := k.GetAcpKeeper().IssueDIDFromAccountAddr(ctx, collab.Address)
	require.NoError(t, err)

	allowed, err := hasPermission(ctx, &k, k.GetPolicyId(ctx), namespaceId, types.CreatePostPermission, collabDID, collab.Address)
	require.NoError(t, err)
	require.True(t, allowed, "create_post permission should resolve for a collaborator")

	outsiderKey := secp256k1.GenPrivKey().PubKey()
	outsider := authtypes.NewBaseAccount(sdk.AccAddress(outsiderKey.Address()), outsiderKey, 3, 1)
	k.accountKeeper.SetAccount(ctx, outsider)

	outsiderDID, err := k.GetAcpKeeper().IssueDIDFromAccountAddr(ctx, outsider.Address)
	require.NoError(t, err)

	allowed, err = hasPermission(ctx, &k, k.GetPolicyId(ctx), namespaceId, types.CreatePostPermission, outsiderDID, outsider.Address)
	require.NoError(t, err)
	require.False(t, allowed, "create_post permission should deny a non-collaborator")
}

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
			name: "register namespace (error: fetching capability for invalid policy)",
			input: &types.MsgRegisterNamespace{
				Creator:   baseAcc.Address,
				Namespace: namespace,
			},
			setup: func() {
				k.SetPolicyId(ctx, "invalidPolicy1")
			},
			expErr:    true,
			expErrMsg: "fetching capability for policy invalidPolicy1",
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
			name:      "create post (error: invalid creator address)",
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
			},
			setup:     func() {},
			expErr:    true,
			expErrMsg: "invalid post payload",
		},
		{
			name: "create post (error: namespace not found)",
			input: &types.MsgCreatePost{
				Creator:   baseAcc.Address,
				Namespace: namespace,
				Payload:   []byte("post123"),
			},
			setup:     func() {},
			expErr:    true,
			expErrMsg: "namespace not found",
		},
		{
			name: "create post (no error)",
			input: &types.MsgCreatePost{
				Creator:   baseAcc.Address,
				Namespace: namespace,
				Payload:   []byte("post123"),
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
			},
			setup:     func() {},
			expErr:    true,
			expErrMsg: "post already exists",
		},
		{
			name: "create post from unrelated signer (no error: posting is open)",
			input: &types.MsgCreatePost{
				Creator:   baseAcc2.Address,
				Namespace: namespace,
				Payload:   []byte("post-from-other"),
			},
			setup:  func() {},
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

func TestMsgCreatePost_EmitsArtifactInEvent(t *testing.T) {
	k, ctx := setupKeeper(t)
	// given test policy and namespace
	pubKey := secp256k1.GenPrivKey().PubKey()
	addr := sdk.AccAddress(pubKey.Address())
	baseAcc := authtypes.NewBaseAccount(addr, pubKey, 1, 1)
	k.accountKeeper.SetAccount(ctx, baseAcc)
	setupTestPolicy(t, ctx, k)
	_, err := k.RegisterNamespace(ctx, &types.MsgRegisterNamespace{
		Creator:   baseAcc.Address,
		Namespace: "ns1",
	})
	require.NoError(t, err)

	// reset event manager
	ctx = ctx.WithEventManager(sdk.NewEventManager())

	// when i create post
	post := types.MsgCreatePost{
		Creator:   baseAcc.Address,
		Namespace: "ns1",
		Payload:   []byte("some payload"),
		Artifact:  "session-id",
	}
	_, err = k.CreatePost(ctx, &post)

	// then post emit event with artifact
	require.NoError(t, err)

	evs := ctx.EventManager().Events()
	require.Len(t, evs, 1)

	creatorDid, err := k.acpKeeper.GetActorDID(ctx, baseAcc.Address)
	require.NoError(t, err)
	eventDid := "\"" + creatorDid + "\""
	ev := evs[0]
	require.Equal(t, `"session-id"`, ev.Attributes[0].Value)
	require.Equal(t, eventDid, ev.Attributes[1].Value)
	require.Equal(t, `"bulletin/ns1"`, ev.Attributes[2].Value)
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
