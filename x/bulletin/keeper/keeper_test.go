package keeper

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sourcenetwork/sourcehub/x/bulletin/types"
)

func TestGetPolicyId(t *testing.T) {
	k, ctx := setupKeeper(t)

	policyId := k.GetPolicyId(ctx)
	require.Equal(t, "", policyId)

	k.SetPolicyId(ctx, "test-policy")

	policyId = k.GetPolicyId(ctx)
	require.Equal(t, "test-policy", policyId)
}

func TestGetNamespace(t *testing.T) {
	k, ctx := setupKeeper(t)

	namespaceId := getNamespaceId("ns1")
	did := "did:key:bob"

	namespace := types.Namespace{
		Id:        namespaceId,
		OwnerDid:  did,
		Creator:   "source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9",
		CreatedAt: ctx.BlockTime(),
	}
	k.SetNamespace(ctx, namespace)

	gotNamespace := k.GetNamespace(ctx, namespaceId)
	require.NotNil(t, gotNamespace)
	require.Equal(t, namespace.Id, gotNamespace.Id)
	require.Equal(t, namespace.OwnerDid, gotNamespace.OwnerDid)
	require.Equal(t, namespace.Creator, gotNamespace.Creator)
	require.Equal(t, namespace.CreatedAt, gotNamespace.CreatedAt)
}

func TestGetAllNamespaces(t *testing.T) {
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

	namespaces := k.GetAllNamespaces(ctx)
	require.Len(t, namespaces, 2)
}

func TestDeleteCollaborator(t *testing.T) {
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

	k.DeleteCollaborator(ctx, namespaceId, did)

	gotCollagorator = k.getCollaborator(ctx, namespaceId, did)
	require.Nil(t, gotCollagorator)
}

func TestGetAllCollaborators(t *testing.T) {
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

	collaborators := k.GetAllCollaborators(ctx)
	require.Len(t, collaborators, 2)
}

func TestGetNamespacePosts(t *testing.T) {
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

	ns1Posts := k.GetNamespacePosts(ctx, namespaceId1)
	require.Len(t, ns1Posts, 2)
	for _, p := range ns1Posts {
		require.Equal(t, namespaceId1, p.Namespace)
	}

	ns2Posts := k.GetNamespacePosts(ctx, namespaceId2)
	require.Len(t, ns2Posts, 1)
	require.Equal(t, namespaceId2, ns2Posts[0].Namespace)
}

func TestGetAllPosts(t *testing.T) {
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

	posts := k.GetAllPosts(ctx)
	require.Len(t, posts, 3)
}
