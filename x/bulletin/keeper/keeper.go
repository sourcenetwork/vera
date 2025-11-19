package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
	acpkeeper "github.com/sourcenetwork/sourcehub/x/acp/keeper"
	"github.com/sourcenetwork/sourcehub/x/bulletin/types"
)

type (
	Keeper struct {
		cdc          codec.BinaryCodec
		storeService store.KVStoreService
		logger       log.Logger

		// the address capable of executing a MsgUpdateParams message. Typically, this
		// should be the x/gov module account.
		authority string

		accountKeeper types.AccountKeeper
		acpKeeper     *acpkeeper.Keeper
		capKeeper     *capabilitykeeper.ScopedKeeper
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	logger log.Logger,
	authority string,
	accountKeeper types.AccountKeeper,
	acpKeeper *acpkeeper.Keeper,
	capKeeper *capabilitykeeper.ScopedKeeper,
) Keeper {
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address: %s", authority))
	}

	return Keeper{
		cdc:           cdc,
		storeService:  storeService,
		logger:        logger,
		authority:     authority,
		accountKeeper: accountKeeper,
		acpKeeper:     acpKeeper,
		capKeeper:     capKeeper,
	}
}

// GetAuthority returns the module's authority.
func (k *Keeper) GetAuthority() string {
	return k.authority
}

// GetAcpKeeper returns the module's AcpKeeper.
func (k *Keeper) GetAcpKeeper() *acpkeeper.Keeper {
	return k.acpKeeper
}

func (k *Keeper) GetScopedKeeper() *capabilitykeeper.ScopedKeeper {
	return k.capKeeper
}

// Logger returns a module-specific logger.
func (k *Keeper) Logger() log.Logger {
	return k.logger.With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// SetPolicyId stores id of the genesis bulletin policy.
func (k *Keeper) SetPolicyId(ctx context.Context, policyId string) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store.Set([]byte(types.PolicyIdKey), []byte(policyId))
}

// GetPolicyId returns genesis bulletin policy id.
func (k *Keeper) GetPolicyId(ctx context.Context) string {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get([]byte(types.PolicyIdKey))
	if bz == nil {
		return ""
	}

	return string(bz)
}

// SetNamespace adds new namespace to the store.
func (k *Keeper) SetNamespace(ctx context.Context, namespace types.Namespace) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.NamespaceKeyPrefix))
	bz := k.cdc.MustMarshal(&namespace)

	store.Set([]byte(namespace.Id), bz)
}

// GetNamespace retrieves existing namespace based on the namespaceId.
func (k *Keeper) GetNamespace(ctx context.Context, namespaceId string) *types.Namespace {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.NamespaceKeyPrefix))

	b := store.Get([]byte(namespaceId))
	if b == nil {
		return nil
	}

	var namespace types.Namespace
	k.cdc.MustUnmarshal(b, &namespace)

	return &namespace
}

// GetAllNamespaces returns all namespaces.
func (k *Keeper) GetAllNamespaces(ctx sdk.Context) []types.Namespace {
	var namespaces []types.Namespace

	namespacesCallback := func(namespace types.Namespace) {
		namespaces = append(namespaces, namespace)
	}

	k.mustIterateNamespaces(ctx, namespacesCallback)

	return namespaces
}

// SetCollaborator adds new collaborator to the store.
func (k *Keeper) SetCollaborator(ctx context.Context, collaborator types.Collaborator) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.CollaboratorKeyPrefix))

	bz := k.cdc.MustMarshal(&collaborator)
	key := types.CollaboratorKey(collaborator.Namespace, collaborator.Did)
	store.Set(key, bz)
}

// DeleteCollaborator removes a collaborator from the store.
func (k *Keeper) DeleteCollaborator(ctx context.Context, namespaceId string, collaboratorDID string) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.CollaboratorKeyPrefix))

	key := types.CollaboratorKey(namespaceId, collaboratorDID)
	store.Delete(key)
}

// GetAllCollaborators returns all collaborators.
func (k *Keeper) GetAllCollaborators(ctx context.Context) []types.Collaborator {
	var collaborators []types.Collaborator

	collaboratorsCallback := func(collaborator types.Collaborator) {
		collaborators = append(collaborators, collaborator)
	}

	k.mustIterateCollaborators(ctx, collaboratorsCallback)

	return collaborators
}

// SetPost adds new post to the store.
func (k *Keeper) SetPost(ctx context.Context, post types.Post) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.PostKeyPrefix))

	bz := k.cdc.MustMarshal(&post)
	key := types.PostKey(post.Namespace, post.Id)
	store.Set(key, bz)
}

// GetNamespacePosts returns namespace posts.
func (k *Keeper) GetNamespacePosts(ctx context.Context, namespaceId string) []types.Post {
	var posts []types.Post

	postsCallback := func(namespaceId string, post types.Post) {
		posts = append(posts, post)
	}

	k.mustIterateNamespacePosts(ctx, namespaceId, postsCallback)

	return posts
}

// GetAllPosts returns all posts.
func (k *Keeper) GetAllPosts(ctx context.Context) []types.Post {
	var posts []types.Post

	postsCallback := func(post types.Post) {
		posts = append(posts, post)
	}

	k.mustIteratePosts(ctx, postsCallback)

	return posts
}

// InitializeCapabilityKeeper allows app to set the capability keeper after the moment of creation.
//
// This is supported since currently the capability module
// does not integrate with the new module dependency injection system.
//
// Panics if the keeper was previously initialized (ie inner pointer != nil).
func (k *Keeper) InitializeCapabilityKeeper(keeper *capabilitykeeper.ScopedKeeper) {
	if k.capKeeper != nil {
		panic("capability keeper already initialized")
	}
	k.capKeeper = keeper
}
