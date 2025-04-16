package keeper

import (
	"context"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/sourcenetwork/sourcehub/x/bulletin/types"
)

// hasNamespace checks if namespace with specified namespaceId exists.
func (k *Keeper) hasNamespace(ctx context.Context, namespaceId string) bool {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.NamespaceKeyPrefix))

	b := store.Get([]byte(namespaceId))

	return b != nil
}

// getPost retrieves a post based on existing namespaceId and postId.
func (k *Keeper) getPost(ctx context.Context, namespaceId string, postId string) *types.Post {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.PostKeyPrefix))

	key := types.PostKey(namespaceId, postId)
	b := store.Get(key)
	if b == nil {
		return nil
	}

	var post types.Post
	k.cdc.MustUnmarshal(b, &post)

	return &post
}

// getCollaborator retrieves a post based on existing namespaceId and collaboratorDID.
func (k *Keeper) getCollaborator(ctx context.Context, namespaceId string, collaboratorDID string) *types.Collaborator {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.CollaboratorKeyPrefix))

	key := types.CollaboratorKey(namespaceId, collaboratorDID)
	b := store.Get(key)
	if b == nil {
		return nil
	}

	var collaborator types.Collaborator
	k.cdc.MustUnmarshal(b, &collaborator)

	return &collaborator
}

// mustIterateNamespaces iterates over all namespaces and performs the provided callback function.
func (k *Keeper) mustIterateNamespaces(ctx sdk.Context, cb func(namespace types.Namespace)) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.NamespaceKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var namespace types.Namespace
		k.cdc.MustUnmarshal(iterator.Value(), &namespace)
		cb(namespace)
	}
}

// mustIterateCollaborators iterates over all collaborators and performs the provided callback function.
func (k *Keeper) mustIterateCollaborators(ctx context.Context, cb func(collaborator types.Collaborator)) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.CollaboratorKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var collaborator types.Collaborator
		k.cdc.MustUnmarshal(iterator.Value(), &collaborator)
		cb(collaborator)
	}
}

// mustIteratePosts iterates over all posts and performs the provided callback function.
func (k *Keeper) mustIteratePosts(ctx context.Context, cb func(post types.Post)) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.PostKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var post types.Post
		k.cdc.MustUnmarshal(iterator.Value(), &post)
		cb(post)
	}
}

// mustIterateNamespacePosts iterates over namespace posts and performs the provided callback function.
func (k *Keeper) mustIterateNamespacePosts(ctx context.Context, namespaceId string, cb func(namespaceId string, post types.Post)) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.PostKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte(namespaceId+"/"))

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var post types.Post
		k.cdc.MustUnmarshal(iterator.Value(), &post)
		cb(namespaceId, post)
	}
}
