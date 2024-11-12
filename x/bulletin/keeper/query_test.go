package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/sourcenetwork/sourcehub/testutil/keeper"
	"github.com/sourcenetwork/sourcehub/x/bulletin/types"
)

func TestParamsQueryIterateGlob(t *testing.T) {
	keeper, ctx := keepertest.BulletinKeeper(t)

	posts := []*types.Post{
		// sub1 set
		{
			Namespace: "test1/foo/bar/key1",
			Payload:   []byte("val1"),
		},
		{
			Namespace: "test1/foo/bar/key2",
			Payload:   []byte("val2"),
		},
		{
			Namespace: "test1/foo/baz/key1",
			Payload:   []byte("val3"),
		},

		// base set
		{
			Namespace: "test1/key1",
			Payload:   []byte("val1"),
		},
		{
			Namespace: "test1/key2",
			Payload:   []byte("val2"),
		},
		{
			Namespace: "test1/key3",
			Payload:   []byte("val3"),
		},

		// sub1 set
		{
			Namespace: "test1/sub1/key1",
			Payload:   []byte("val1"),
		},
		{
			Namespace: "test1/sub1/key2",
			Payload:   []byte("val2"),
		},
		{
			Namespace: "test1/sub1/key3",
			Payload:   []byte("val3"),
		},
	}

	// add posts
	for _, post := range posts {
		keeper.AddPost(ctx, *post)
	}

	// iterate all
	resp1, err := keeper.IterateGlob(ctx, &types.QueryIterateGlobRequest{
		Namespace: "test1",
		Glob:      "*",
	})
	require.NoError(t, err)
	require.Equal(t, posts, resp1.Posts)

	// iterate sub1
	resp2, err := keeper.IterateGlob(ctx, &types.QueryIterateGlobRequest{
		Namespace: "test1",
		Glob:      "sub1*",
	})
	require.NoError(t, err)
	require.Equal(t, posts[6:9], resp2.Posts)

	// iterate no subset
	resp3, err := keeper.IterateGlob(ctx, &types.QueryIterateGlobRequest{
		Namespace: "test1",
		Glob:      "k*",
	})
	require.NoError(t, err)
	require.Equal(t, posts[3:6], resp3.Posts)

	// empty glob will return the empty set
	resp4, err := keeper.IterateGlob(ctx, &types.QueryIterateGlobRequest{
		Namespace: "test1",
		Glob:      "",
	})
	require.NoError(t, err)
	require.Zero(t, resp4.Posts)

	// mid selector
	resp5, err := keeper.IterateGlob(ctx, &types.QueryIterateGlobRequest{
		Namespace: "test1",
		Glob:      "foo/*/key1",
	})
	require.NoError(t, err)
	require.Equal(t,
		[]*types.Post{
			{
				Namespace: "test1/foo/bar/key1",
				Payload:   []byte("val1"),
			},
			{
				Namespace: "test1/foo/baz/key1",
				Payload:   []byte("val3"),
			},
		}, resp5.Posts)
}
