package keeper

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	query "github.com/cosmos/cosmos-sdk/types/query"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/sourcenetwork/vera/x/bulletin/types"
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
	payload2 := []byte("post321")

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
	})
	require.NoError(t, err)

	_, err = k.CreatePost(ctx, &types.MsgCreatePost{
		Creator:   baseAcc.Address,
		Namespace: namespace,
		Payload:   payload2,
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
			},
			{
				Id:         postId2,
				Namespace:  getNamespaceId(namespace),
				CreatorDid: ownerDID,
				Payload:    payload2,
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
	payload2 := []byte("post321")

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
	})
	require.NoError(t, err)

	_, err = k.CreatePost(ctx, &types.MsgCreatePost{
		Creator:   baseAcc.Address,
		Namespace: namespace2,
		Payload:   payload2,
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
			},
			{
				Id:         postId2,
				Namespace:  getNamespaceId(namespace2),
				CreatorDid: ownerDID,
				Payload:    payload2,
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
	artifact := "artifact"

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
		Artifact:  artifact,
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
		},
	}, response)
}

func TestBulletinPolicyIdQuery(t *testing.T) {
	k, ctx := setupKeeper(t)

	setupTestPolicy(t, ctx, k)

	response, err := k.BulletinPolicyId(ctx, &types.QueryBulletinPolicyIdRequest{})
	require.NoError(t, err)
	require.Equal(t, k.GetPolicyId(ctx), response.PolicyId)
}

func TestBulletinPolicyIdQuery_InvalidRequest(t *testing.T) {
	k, ctx := setupKeeper(t)

	response, err := k.BulletinPolicyId(ctx, nil)
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	require.Nil(t, response)
}

func TestBulletinPolicyIdQuery_PolicyNotSet(t *testing.T) {
	k, ctx := setupKeeper(t)

	response, err := k.BulletinPolicyId(ctx, &types.QueryBulletinPolicyIdRequest{})
	require.Error(t, err)
	require.Equal(t, codes.NotFound, status.Code(err))
	require.Nil(t, response)
}

func TestIterateGlob(t *testing.T) {
	keeper, ctx := setupKeeper(t)

	posts := []*types.Post{
		// sub1 set
		{
			Namespace: "bulletin/test1/foo/bar/key1",
			Payload:   []byte("val1"),
		},
		{
			Namespace: "bulletin/test1/foo/bar/key2",
			Payload:   []byte("val2"),
		},
		{
			Namespace: "bulletin/test1/foo/baz/key1",
			Payload:   []byte("val3"),
		},

		// base set
		{
			Namespace: "bulletin/test1/key1",
			Payload:   []byte("val1"),
		},
		{
			Namespace: "bulletin/test1/key2",
			Payload:   []byte("val2"),
		},
		{
			Namespace: "bulletin/test1/key3",
			Payload:   []byte("val3"),
		},

		// sub1 set
		{
			Namespace: "bulletin/test1/sub1/key1",
			Payload:   []byte("val1"),
		},
		{
			Namespace: "bulletin/test1/sub1/key2",
			Payload:   []byte("val2"),
		},
		{
			Namespace: "bulletin/test1/sub1/key3",
			Payload:   []byte("val3"),
		},
	}

	// add posts
	for _, post := range posts {
		keeper.SetPost(ctx, *post)
	}

	// iterate all
	resp1, err := keeper.IterateGlob(ctx, &types.QueryIterateGlobRequest{
		Namespace: "bulletin/test1",
		Glob:      "*",
	})
	require.NoError(t, err)
	require.Equal(t, posts, resp1.Posts)

	// iterate sub1
	resp2, err := keeper.IterateGlob(ctx, &types.QueryIterateGlobRequest{
		Namespace: "bulletin/test1",
		Glob:      "sub1*",
	})
	require.NoError(t, err)
	require.Equal(t, posts[6:9], resp2.Posts)

	// iterate no subset
	resp3, err := keeper.IterateGlob(ctx, &types.QueryIterateGlobRequest{
		Namespace: "bulletin/test1",
		Glob:      "k*",
	})
	require.NoError(t, err)
	require.Equal(t, posts[3:6], resp3.Posts)

	// empty glob will return the empty set
	resp4, err := keeper.IterateGlob(ctx, &types.QueryIterateGlobRequest{
		Namespace: "bulletin/test1",
		Glob:      "",
	})
	require.NoError(t, err)
	require.Zero(t, resp4.Posts)

	// mid selector
	resp5, err := keeper.IterateGlob(ctx, &types.QueryIterateGlobRequest{
		Namespace: "bulletin/test1",
		Glob:      "foo/*/key1",
	})
	require.NoError(t, err)
	require.Equal(t,
		[]*types.Post{
			{
				Namespace: "bulletin/test1/foo/bar/key1",
				Payload:   []byte("val1"),
			},
			{
				Namespace: "bulletin/test1/foo/baz/key1",
				Payload:   []byte("val3"),
			},
		}, resp5.Posts)

	// pagination
	resp6, err := keeper.IterateGlob(ctx, &types.QueryIterateGlobRequest{
		Namespace: "bulletin/test1",
		Glob:      "*",
		Pagination: &query.PageRequest{
			Limit: 3,
		},
	})
	require.NoError(t, err)
	require.Equal(t, posts[:3], resp6.Posts)

	// continuation pagination
	resp7, err := keeper.IterateGlob(ctx, &types.QueryIterateGlobRequest{
		Namespace: "bulletin/test1",
		Glob:      "*",
		Pagination: &query.PageRequest{
			Key: resp6.Pagination.NextKey,
		},
	})
	require.NoError(t, err)
	require.Equal(t, posts[3:], resp7.Posts)
}

func TestIterateGlobWithPostIds(t *testing.T) {
	keeper, ctx := setupKeeper(t)

	posts := []*types.Post{
		// posts with exact namespace match and non-empty postIds
		{
			Id:        "post1",
			Namespace: "bulletin/test2",
			Payload:   []byte("val1"),
		},
		{
			Id:        "post2",
			Namespace: "bulletin/test2",
			Payload:   []byte("val2"),
		},
		{
			Id:        "post3",
			Namespace: "bulletin/test2",
			Payload:   []byte("val3"),
		},
		// posts with sub-paths in namespace
		{
			Id:        "post1",
			Namespace: "bulletin/test2/sub1",
			Payload:   []byte("val4"),
		},
		{
			Id:        "post2",
			Namespace: "bulletin/test2/sub1",
			Payload:   []byte("val5"),
		},
	}

	// add posts
	for _, post := range posts {
		keeper.SetPost(ctx, *post)
	}

	// match all posts (both exact namespace and sub-paths)
	resp1, err := keeper.IterateGlob(ctx, &types.QueryIterateGlobRequest{
		Namespace: "bulletin/test2",
		Glob:      "*",
	})
	require.NoError(t, err)
	require.Equal(t, posts, resp1.Posts)

	// match only posts with exact namespace (non-empty postIds, "/" separator case)
	resp2, err := keeper.IterateGlob(ctx, &types.QueryIterateGlobRequest{
		Namespace: "bulletin/test2",
		Glob:      "post*",
	})
	require.NoError(t, err)
	require.Equal(t, posts[0:3], resp2.Posts)

	// match only posts in sub1 namespace
	resp3, err := keeper.IterateGlob(ctx, &types.QueryIterateGlobRequest{
		Namespace: "bulletin/test2",
		Glob:      "sub1/*",
	})
	require.NoError(t, err)
	require.Equal(t, posts[3:5], resp3.Posts)

	// match specific postId at exact namespace boundary
	resp4, err := keeper.IterateGlob(ctx, &types.QueryIterateGlobRequest{
		Namespace: "bulletin/test2",
		Glob:      "post1",
	})
	require.NoError(t, err)
	require.Equal(t, []*types.Post{posts[0]}, resp4.Posts)
}

func TestIterateGlobNamespaceBoundaryPagination(t *testing.T) {
	keeper, ctx := setupKeeper(t)

	expected := []*types.Post{
		{Id: "post1", Namespace: "bulletin/ns1"},
		{Id: "post2", Namespace: "bulletin/ns1/sub"},
	}
	for _, post := range append(expected, &types.Post{Id: "other", Namespace: "bulletin/ns10"}) {
		keeper.SetPost(ctx, *post)
	}

	first, err := keeper.IterateGlob(ctx, &types.QueryIterateGlobRequest{
		Namespace:  "ns1",
		Glob:       "*",
		Pagination: &query.PageRequest{Limit: 1},
	})
	require.NoError(t, err)
	require.Equal(t, expected[:1], first.Posts)
	require.NotEmpty(t, first.Pagination.NextKey)

	second, err := keeper.IterateGlob(ctx, &types.QueryIterateGlobRequest{
		Namespace: "ns1",
		Glob:      "*",
		Pagination: &query.PageRequest{
			Key:   first.Pagination.NextKey,
			Limit: 1,
		},
	})
	require.NoError(t, err)
	require.Equal(t, expected[1:], second.Posts)
	require.Empty(t, second.Pagination.NextKey)
}

func TestIterateGlobEdgeCases(t *testing.T) {
	keeper, ctx := setupKeeper(t)

	posts := []*types.Post{
		{
			Id:        "",
			Namespace: "bulletin/test3/mixed/post1",
			Payload:   []byte("val1"),
		},
		{
			Id:        "actualId",
			Namespace: "bulletin/test3/mixed",
			Payload:   []byte("val2"),
		},
		{
			Id:        "my/nested/id",
			Namespace: "bulletin/test3",
			Payload:   []byte("val3"),
		},
		{
			Id:        "",
			Namespace: "bulletin/test3/a/b/c/d/key",
			Payload:   []byte("val4"),
		},
	}

	// add posts
	for _, post := range posts {
		keeper.SetPost(ctx, *post)
	}

	// match all posts in test3 namespace
	resp1, err := keeper.IterateGlob(ctx, &types.QueryIterateGlobRequest{
		Namespace: "bulletin/test3",
		Glob:      "*",
	})
	require.NoError(t, err)
	require.ElementsMatch(t, posts, resp1.Posts)

	// postId with "/" characters should match after unsanitization
	resp2, err := keeper.IterateGlob(ctx, &types.QueryIterateGlobRequest{
		Namespace: "bulletin/test3",
		Glob:      "my/nested/id",
	})
	require.NoError(t, err)
	require.Equal(t, []*types.Post{posts[2]}, resp2.Posts)

	// match with wildcard in postId path
	resp3, err := keeper.IterateGlob(ctx, &types.QueryIterateGlobRequest{
		Namespace: "bulletin/test3",
		Glob:      "my/*/id",
	})
	require.NoError(t, err)
	require.Equal(t, []*types.Post{posts[2]}, resp3.Posts)

	// deep nesting with multiple wildcards
	resp4, err := keeper.IterateGlob(ctx, &types.QueryIterateGlobRequest{
		Namespace: "bulletin/test3",
		Glob:      "a/*/*/d/key",
	})
	require.NoError(t, err)
	require.Equal(t, []*types.Post{posts[3]}, resp4.Posts)

	// non-matching pattern should return empty
	resp5, err := keeper.IterateGlob(ctx, &types.QueryIterateGlobRequest{
		Namespace: "bulletin/test3",
		Glob:      "nonexistent/*",
	})
	require.NoError(t, err)
	require.Empty(t, resp5.Posts)

	// exact match for namespace path with no wildcards
	resp6, err := keeper.IterateGlob(ctx, &types.QueryIterateGlobRequest{
		Namespace: "bulletin/test3",
		Glob:      "mixed/post1",
	})
	require.NoError(t, err)
	require.Equal(t, []*types.Post{posts[0]}, resp6.Posts)

	// match mixed posts with empty and non-empty postIds
	resp7, err := keeper.IterateGlob(ctx, &types.QueryIterateGlobRequest{
		Namespace: "bulletin/test3",
		Glob:      "mixed*",
	})
	require.NoError(t, err)
	require.ElementsMatch(t, []*types.Post{posts[0], posts[1]}, resp7.Posts)
}
