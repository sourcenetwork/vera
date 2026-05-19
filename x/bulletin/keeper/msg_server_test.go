package keeper

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	"github.com/sourcenetwork/sourcehub/x/bulletin/types"
)

func TestMsgCreatePost_DoesNotValidatePayloadShape(t *testing.T) {
	k, ctx := setupKeeper(t)
	require.NoError(t, k.SetParams(ctx, types.DefaultParams()))

	ownerKey := secp256k1.GenPrivKey().PubKey()
	owner := authtypes.NewBaseAccount(sdk.AccAddress(ownerKey.Address()), ownerKey, 1, 1)
	k.accountKeeper.SetAccount(ctx, owner)

	namespace := "orbis/rings/ring1"

	_, err := k.RegisterNamespace(ctx, &types.MsgRegisterNamespace{
		Creator:   owner.Address,
		Namespace: namespace,
	})
	require.NoError(t, err)

	_, err = k.CreatePost(ctx, &types.MsgCreatePost{
		Creator:   owner.Address,
		Namespace: namespace,
		Payload:   []byte(`{"anything":"goes"}`),
	})
	require.NoError(t, err)

	postID := types.GeneratePostId(getNamespaceId(namespace), []byte(`{"anything":"goes"}`))
	post := k.getPost(ctx, getNamespaceId(namespace), postID)
	require.NotNil(t, post)
	require.Equal(t, []byte(`{"anything":"goes"}`), post.Payload)
}

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
	require.True(t, allowed)
}
