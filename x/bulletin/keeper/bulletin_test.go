package keeper

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sourcenetwork/sourcehub/x/bulletin/types"
)

func TestGetNamespaceId(t *testing.T) {
	id1 := "ns1"
	id2 := "ns2"
	id3 := "bulletin/ns1"

	require.Equal(t, "bulletin/ns1", getNamespaceId(id1))
	require.Equal(t, "bulletin/ns2", getNamespaceId(id2))
	require.Equal(t, "bulletin/ns1", getNamespaceId(id3))
	require.Equal(t, getNamespaceId(id1), getNamespaceId(id3))
}

func TestHasPolicy(t *testing.T) {
	k, ctx := setupKeeper(t)

	require.False(t, k.hasPolicy(ctx))

	k.SetPolicyId(ctx, "policy1")
	require.True(t, k.hasPolicy(ctx))
}

func TestEnsurePolicy(t *testing.T) {
	k, ctx := setupKeeper(t)

	require.False(t, k.hasPolicy(ctx))

	k.EnsurePolicy(ctx)
	require.True(t, k.hasPolicy(ctx))
}

func TestHasNamespace(t *testing.T) {
	k, ctx := setupKeeper(t)

	namespaceId := getNamespaceId("ns1")
	did := "did:key:bob"

	require.False(t, k.hasNamespace(ctx, namespaceId))

	namespace := types.Namespace{
		Id:        namespaceId,
		OwnerDid:  did,
		Creator:   "source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9",
		CreatedAt: ctx.BlockTime(),
	}
	k.SetNamespace(ctx, namespace)

	require.True(t, k.hasNamespace(ctx, namespaceId))
}

func TestGetPost(t *testing.T) {
	k, ctx := setupKeeper(t)

	namespaceId := getNamespaceId("ns1")
	postId := "post1"
	did := "did:key:bob"

	gotPost := k.getPost(ctx, namespaceId, postId)
	require.Nil(t, gotPost)

	post := types.Post{
		Id:         postId,
		Namespace:  namespaceId,
		CreatorDid: did,
		Payload:    []byte("payload123"),
		Proof:      []byte("proof123"),
	}
	k.SetPost(ctx, post)

	gotPost = k.getPost(ctx, namespaceId, postId)
	require.NotNil(t, gotPost)
	require.Equal(t, post.Id, gotPost.Id)
	require.Equal(t, post.Namespace, gotPost.Namespace)
	require.Equal(t, post.CreatorDid, gotPost.CreatorDid)
	require.Equal(t, post.Payload, gotPost.Payload)
	require.Equal(t, post.Proof, gotPost.Proof)
}

func TestGetCollaborator(t *testing.T) {
	k, ctx := setupKeeper(t)

	namespaceId := getNamespaceId("ns1")
	did := "did:key:bob"

	gotCollagorator := k.getCollaborator(ctx, namespaceId, did)
	require.Nil(t, gotCollagorator)

	collaborator := types.Collaborator{
		Address:   "source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9",
		Did:       did,
		Namespace: namespaceId,
	}
	k.SetCollaborator(ctx, collaborator)

	gotCollagorator = k.getCollaborator(ctx, namespaceId, did)
	require.NotNil(t, gotCollagorator)
	require.Equal(t, collaborator.Address, gotCollagorator.Address)
	require.Equal(t, collaborator.Did, gotCollagorator.Did)
	require.Equal(t, collaborator.Namespace, gotCollagorator.Namespace)
}

func TestMustIterateNamespaces(t *testing.T) {
	k, ctx := setupKeeper(t)

	namespaceId1 := getNamespaceId("ns1")
	namespaceId2 := getNamespaceId("ns2")
	did := "did:key:bob"

	ns1 := types.Namespace{
		Id:        namespaceId1,
		OwnerDid:  did,
		Creator:   "source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9",
		CreatedAt: ctx.BlockTime(),
	}
	k.SetNamespace(ctx, ns1)

	ns2 := types.Namespace{
		Id:        namespaceId2,
		OwnerDid:  did,
		Creator:   "source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9",
		CreatedAt: ctx.BlockTime(),
	}
	k.SetNamespace(ctx, ns2)

	var namespaces []types.Namespace
	k.mustIterateNamespaces(ctx, func(ns types.Namespace) {
		namespaces = append(namespaces, ns)
	})

	require.ElementsMatch(t, []types.Namespace{ns1, ns2}, namespaces)
	require.Equal(t, 2, len(namespaces))
}

func TestMustIterateCollaborators(t *testing.T) {
	k, ctx := setupKeeper(t)

	namespaceId := getNamespaceId("ns1")

	c1 := types.Collaborator{
		Address:   "source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9",
		Did:       "did:key:bob",
		Namespace: namespaceId,
	}
	k.SetCollaborator(ctx, c1)

	c2 := types.Collaborator{
		Address:   "source18jtkvj0995fy7lggqayg2f5syna92ndq5mkuv4",
		Did:       "did:key:sam",
		Namespace: namespaceId,
	}
	k.SetCollaborator(ctx, c2)

	var collaborators []types.Collaborator
	k.mustIterateCollaborators(ctx, func(c types.Collaborator) {
		collaborators = append(collaborators, c)
	})

	require.ElementsMatch(t, []types.Collaborator{c1, c2}, collaborators)
	require.Equal(t, 2, len(collaborators))
}

func TestMustIteratePosts(t *testing.T) {
	k, ctx := setupKeeper(t)

	namespaceId := getNamespaceId("ns1")
	did := "did:key:bob"

	post1 := types.Post{
		Id:         "post1",
		Namespace:  namespaceId,
		CreatorDid: did,
		Payload:    []byte("payload123"),
		Proof:      []byte("proof123"),
	}
	k.SetPost(ctx, post1)

	post2 := types.Post{
		Id:         "post2",
		Namespace:  namespaceId,
		CreatorDid: did,
		Payload:    []byte("payload456"),
		Proof:      []byte("proof456"),
	}
	k.SetPost(ctx, post2)

	var posts []types.Post
	k.mustIteratePosts(ctx, func(p types.Post) {
		posts = append(posts, p)
	})

	require.ElementsMatch(t, []types.Post{post1, post2}, posts)
	require.Equal(t, 2, len(posts))
}

func TestMustIterateNamespacePosts(t *testing.T) {
	k, ctx := setupKeeper(t)

	namespaceId1 := getNamespaceId("ns1")
	namespaceId2 := getNamespaceId("ns2")
	did := "did:key:bob"

	post1 := types.Post{
		Id:         "post1",
		Namespace:  namespaceId1,
		CreatorDid: did,
		Payload:    []byte("payload123"),
		Proof:      []byte("proof123"),
	}
	k.SetPost(ctx, post1)

	post2 := types.Post{
		Id:         "post2",
		Namespace:  namespaceId1,
		CreatorDid: did,
		Payload:    []byte("payload456"),
		Proof:      []byte("proof456"),
	}
	k.SetPost(ctx, post2)

	post3 := types.Post{
		Id:         "post3",
		Namespace:  namespaceId2,
		CreatorDid: did,
		Payload:    []byte("payload789"),
		Proof:      []byte("proof789"),
	}
	k.SetPost(ctx, post3)

	var ns1Posts []types.Post
	k.mustIterateNamespacePosts(ctx, namespaceId1, func(namespaceId string, p types.Post) {
		require.Equal(t, namespaceId1, namespaceId)
		ns1Posts = append(ns1Posts, p)
	})

	require.ElementsMatch(t, []types.Post{post1, post2}, ns1Posts)
	require.Equal(t, 2, len(ns1Posts))

	var ns2Posts []types.Post
	k.mustIterateNamespacePosts(ctx, namespaceId2, func(namespaceId string, p types.Post) {
		require.Equal(t, namespaceId2, namespaceId)
		ns2Posts = append(ns2Posts, p)
	})

	require.ElementsMatch(t, []types.Post{post3}, ns2Posts)
	require.Equal(t, 1, len(ns2Posts))
}
