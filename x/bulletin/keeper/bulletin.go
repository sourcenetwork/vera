package keeper

import (
	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/sourcenetwork/sourcehub/x/bulletin/types"
)

func (k Keeper) AddPost(ctx sdk.Context, post types.Post) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.PostKey))
	bz := k.cdc.MustMarshal(&post)

	store.Set([]byte(post.Namespace), bz)
}

func (k Keeper) GetPost(ctx sdk.Context, namespace string) (post types.Post, found bool) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.PostKey))

	b := store.Get([]byte(namespace))
	if b == nil {
		return post, false
	}

	k.cdc.MustUnmarshal(b, &post)

	return post, true
}
