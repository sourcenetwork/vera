package keeper

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	query "github.com/cosmos/cosmos-sdk/types/query"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	"github.com/sourcenetwork/sourcehub/x/bulletin/types"
)

func TestParamsQuery(t *testing.T) {
	k, ctx := setupKeeper(t)
	params := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, params))

	response, err := k.Params(ctx, &types.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, &types.QueryParamsResponse{Params: params}, response)
}

func TestParamsQuery_InvalidRequest(t *testing.T) {
	k, ctx := setupKeeper(t)
	params := k.GetParams(ctx)
	require.NoError(t, k.SetParams(ctx, params))

	response, err := k.Params(ctx, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid request")
	require.Nil(t, response)
}

func TestNamespacesQuery(t *testing.T) {
	k, ctx := setupKeeper(t)

	setupTestPolicy(t, ctx, k)

	pubKey := secp256k1.GenPrivKey().PubKey()
	addr := sdk.AccAddress(pubKey.Address())
	baseAcc := authtypes.NewBaseAccount(addr, pubKey, 1, 1)
	k.accountKeeper.SetAccount(ctx, baseAcc)

	namespace1 := "ns1"
	namespace2 := "ns2"

	ownerDID, err := k.GetAcpKeeper().IssueDIDFromAccountAddr(ctx, baseAcc.Address)
	require.NoError(t, err)

	_, err = k.RegisterNamespace(ctx, &types.MsgRegisterNamespace{
		Creator:   baseAcc.Address,
		Namespace: namespace1,
	})
	require.NoError(t, err)

	_, err = k.RegisterNamespace(ctx, &types.MsgRegisterNamespace{
		Creator:   baseAcc.Address,
		Namespace: namespace2,
	})
	require.NoError(t, err)

	response, err := k.Namespaces(ctx, &types.QueryNamespacesRequest{})
	require.NoError(t, err)
	require.Equal(t, &types.QueryNamespacesResponse{
		Namespaces: []types.Namespace{
			{
				Id:       getNamespaceId(namespace1),
				Creator:  baseAcc.Address,
				OwnerDid: ownerDID,
			},
			{
				Id:       getNamespaceId(namespace2),
				Creator:  baseAcc.Address,
				OwnerDid: ownerDID,
			},
		},
		Pagination: &query.PageResponse{
			NextKey: nil,
			Total:   2,
		},
	}, response)
}

func TestNamespaceQuery(t *testing.T) {
	k, ctx := setupKeeper(t)

	setupTestPolicy(t, ctx, k)

	pubKey := secp256k1.GenPrivKey().PubKey()
	addr := sdk.AccAddress(pubKey.Address())
	baseAcc := authtypes.NewBaseAccount(addr, pubKey, 1, 1)
	k.accountKeeper.SetAccount(ctx, baseAcc)

	namespace := "ns1"

	ownerDID, err := k.GetAcpKeeper().IssueDIDFromAccountAddr(ctx, baseAcc.Address)
	require.NoError(t, err)

	_, err = k.RegisterNamespace(ctx, &types.MsgRegisterNamespace{
		Creator:   baseAcc.Address,
		Namespace: namespace,
	})
	require.NoError(t, err)

	response, err := k.Namespace(ctx, &types.QueryNamespaceRequest{
		Namespace: namespace,
	})
	require.NoError(t, err)
	require.Equal(t, &types.QueryNamespaceResponse{
		Namespace: &types.Namespace{
			Id:       getNamespaceId(namespace),
			Creator:  baseAcc.Address,
			OwnerDid: ownerDID,
		},
	}, response)
}

func TestNamespaceCollaboratorsQuery(t *testing.T) {
	k, ctx := setupKeeper(t)

	setupTestPolicy(t, ctx, k)

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

	firstCollaboratorDID, err := k.GetAcpKeeper().IssueDIDFromAccountAddr(ctx, baseAcc2.Address)
	require.NoError(t, err)

	secondCollaboratorDID, err := k.GetAcpKeeper().IssueDIDFromAccountAddr(ctx, baseAcc3.Address)
	require.NoError(t, err)

	_, err = k.RegisterNamespace(ctx, &types.MsgRegisterNamespace{
		Creator:   baseAcc.Address,
		Namespace: namespace,
	})
	require.NoError(t, err)

	_, err = k.AddCollaborator(ctx, &types.MsgAddCollaborator{
		Creator:      baseAcc.Address,
		Namespace:    namespace,
		Collaborator: baseAcc2.Address,
	})
	require.NoError(t, err)

	_, err = k.AddCollaborator(ctx, &types.MsgAddCollaborator{
		Creator:      baseAcc.Address,
		Namespace:    namespace,
		Collaborator: baseAcc3.Address,
	})
	require.NoError(t, err)

	response, err := k.NamespaceCollaborators(ctx, &types.QueryNamespaceCollaboratorsRequest{
		Namespace: namespace,
	})
	require.NoError(t, err)
	require.ElementsMatch(t, []types.Collaborator{
		{
			Address:   baseAcc2.Address,
			Did:       firstCollaboratorDID,
			Namespace: getNamespaceId(namespace),
		},
		{
			Address:   baseAcc3.Address,
			Did:       secondCollaboratorDID,
			Namespace: getNamespaceId(namespace),
		},
	}, response.Collaborators)
}

func TestNamespacePostsQuery(t *testing.T) {
	k, ctx := setupKeeper(t)

	setupTestPolicy(t, ctx, k)

	pubKey := secp256k1.GenPrivKey().PubKey()
	addr := sdk.AccAddress(pubKey.Address())
	baseAcc := authtypes.NewBaseAccount(addr, pubKey, 1, 1)
	k.accountKeeper.SetAccount(ctx, baseAcc)

	namespace := "ns1"
	payload1 := []byte("post123")
	proof1 := []byte("proof456")
	payload2 := []byte("post321")
	proof2 := []byte("proof654")

	postId1 := types.GeneratePostId(getNamespaceId(namespace), payload1)
	postId2 := types.GeneratePostId(getNamespaceId(namespace), payload2)

	ownerDID, err := k.GetAcpKeeper().IssueDIDFromAccountAddr(ctx, baseAcc.Address)
	require.NoError(t, err)

	_, err = k.RegisterNamespace(ctx, &types.MsgRegisterNamespace{
		Creator:   baseAcc.Address,
		Namespace: namespace,
	})
	require.NoError(t, err)

	_, err = k.CreatePost(ctx, &types.MsgCreatePost{
		Creator:   baseAcc.Address,
		Namespace: namespace,
		Payload:   payload1,
		Proof:     proof1,
	})
	require.NoError(t, err)

	_, err = k.CreatePost(ctx, &types.MsgCreatePost{
		Creator:   baseAcc.Address,
		Namespace: namespace,
		Payload:   payload2,
		Proof:     proof2,
	})
	require.NoError(t, err)

	response, err := k.NamespacePosts(ctx, &types.QueryNamespacePostsRequest{
		Namespace: namespace,
	})
	require.NoError(t, err)
	require.Equal(t, &types.QueryNamespacePostsResponse{
		Posts: []types.Post{
			{
				Id:         postId1,
				Namespace:  getNamespaceId(namespace),
				CreatorDid: ownerDID,
				Payload:    payload1,
				Proof:      proof1,
			},
			{
				Id:         postId2,
				Namespace:  getNamespaceId(namespace),
				CreatorDid: ownerDID,
				Payload:    payload2,
				Proof:      proof2,
			},
		},
		Pagination: &query.PageResponse{
			NextKey: nil,
			Total:   2,
		},
	}, response)
}

func TestPostsQuery(t *testing.T) {
	k, ctx := setupKeeper(t)

	setupTestPolicy(t, ctx, k)

	pubKey := secp256k1.GenPrivKey().PubKey()
	addr := sdk.AccAddress(pubKey.Address())
	baseAcc := authtypes.NewBaseAccount(addr, pubKey, 1, 1)
	k.accountKeeper.SetAccount(ctx, baseAcc)

	namespace1 := "ns1"
	namespace2 := "ns2"
	payload1 := []byte("post123")
	proof1 := []byte("proof456")
	payload2 := []byte("post321")
	proof2 := []byte("proof654")

	postId1 := types.GeneratePostId(getNamespaceId(namespace1), payload1)
	postId2 := types.GeneratePostId(getNamespaceId(namespace2), payload2)

	ownerDID, err := k.GetAcpKeeper().IssueDIDFromAccountAddr(ctx, baseAcc.Address)
	require.NoError(t, err)

	_, err = k.RegisterNamespace(ctx, &types.MsgRegisterNamespace{
		Creator:   baseAcc.Address,
		Namespace: namespace1,
	})
	require.NoError(t, err)

	_, err = k.RegisterNamespace(ctx, &types.MsgRegisterNamespace{
		Creator:   baseAcc.Address,
		Namespace: namespace2,
	})
	require.NoError(t, err)

	_, err = k.CreatePost(ctx, &types.MsgCreatePost{
		Creator:   baseAcc.Address,
		Namespace: namespace1,
		Payload:   payload1,
		Proof:     proof1,
	})
	require.NoError(t, err)

	_, err = k.CreatePost(ctx, &types.MsgCreatePost{
		Creator:   baseAcc.Address,
		Namespace: namespace2,
		Payload:   payload2,
		Proof:     proof2,
	})
	require.NoError(t, err)

	response, err := k.Posts(ctx, &types.QueryPostsRequest{})
	require.NoError(t, err)
	require.Equal(t, &types.QueryPostsResponse{
		Posts: []types.Post{
			{
				Id:         postId1,
				Namespace:  getNamespaceId(namespace1),
				CreatorDid: ownerDID,
				Payload:    payload1,
				Proof:      proof1,
			},
			{
				Id:         postId2,
				Namespace:  getNamespaceId(namespace2),
				CreatorDid: ownerDID,
				Payload:    payload2,
				Proof:      proof2,
			},
		},
		Pagination: &query.PageResponse{
			NextKey: nil,
			Total:   2,
		},
	}, response)
}

func TestPostQuery(t *testing.T) {
	k, ctx := setupKeeper(t)

	setupTestPolicy(t, ctx, k)

	pubKey := secp256k1.GenPrivKey().PubKey()
	addr := sdk.AccAddress(pubKey.Address())
	baseAcc := authtypes.NewBaseAccount(addr, pubKey, 1, 1)
	k.accountKeeper.SetAccount(ctx, baseAcc)

	namespace := "ns1"
	payload1 := []byte("post123")
	proof1 := []byte("proof456")

	postId1 := types.GeneratePostId(getNamespaceId(namespace), payload1)

	ownerDID, err := k.GetAcpKeeper().IssueDIDFromAccountAddr(ctx, baseAcc.Address)
	require.NoError(t, err)

	_, err = k.RegisterNamespace(ctx, &types.MsgRegisterNamespace{
		Creator:   baseAcc.Address,
		Namespace: namespace,
	})
	require.NoError(t, err)

	_, err = k.CreatePost(ctx, &types.MsgCreatePost{
		Creator:   baseAcc.Address,
		Namespace: namespace,
		Payload:   payload1,
		Proof:     proof1,
	})
	require.NoError(t, err)

	response, err := k.Post(ctx, &types.QueryPostRequest{
		Namespace: namespace,
		Id:        postId1,
	})
	require.NoError(t, err)
	require.Equal(t, &types.QueryPostResponse{
		Post: &types.Post{
			Id:         postId1,
			Namespace:  getNamespaceId(namespace),
			CreatorDid: ownerDID,
			Payload:    payload1,
			Proof:      proof1,
		},
	}, response)
}
