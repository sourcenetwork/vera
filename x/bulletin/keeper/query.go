package keeper

import (
	"context"
	"fmt"
	"strings"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/sourcenetwork/sourcehub/utils"
	"github.com/sourcenetwork/sourcehub/x/bulletin/types"
)

var _ types.QueryServer = Keeper{}

func (k Keeper) IterateGlob(goCtx context.Context, req *types.QueryIterateGlobRequest) (*types.QueryIterateGlobResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	if len(req.Namespace) == 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid namespace")
	}
	// prefix store only returns "/" prefixed keys
	if !strings.HasPrefix(req.Glob, "/") {
		req.Glob = "/" + req.Glob
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.PostKey+req.Namespace))
	var posts []*types.Post

	// Define the iterator function.
	iterator := store.Iterator(nil, nil)
	defer iterator.Close()

	// Use the Paginate function to handle the pagination logic.
	i := 0
	pageRes, err := query.Paginate(store, req.Pagination, func(key []byte, value []byte) error {
		// Process each key-value pair here.
		fmt.Println("paginate key:", string(key))
		if utils.Glob(req.Glob, string(key)) {
			posts = append(posts, &types.Post{})
			k.cdc.MustUnmarshal(value, posts[i])
			i++
		}

		return nil // return an error if you need to halt the iteration
	})
	if err != nil {
		return nil, err
	}

	resp := &types.QueryIterateGlobResponse{
		Posts:      posts,
		Pagination: pageRes,
	}

	return resp, nil
}
