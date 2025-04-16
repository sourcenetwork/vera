package keeper

import (
	"bytes"
	"context"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/sourcenetwork/sourcehub/x/bulletin/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ types.QueryServer = &Keeper{}

// Params query returns bulletin module params.
func (k *Keeper) Params(ctx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	return &types.QueryParamsResponse{Params: k.GetParams(ctx)}, nil
}

// Namespaces query returns all namespaces with pagination.
func (k *Keeper) Namespaces(ctx context.Context, req *types.QueryNamespacesRequest) (*types.QueryNamespacesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	namespaces, pageRes, err := k.getNamespacesPaginated(ctx, req.Pagination)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryNamespacesResponse{Namespaces: namespaces, Pagination: pageRes}, nil
}

// Namespace query returns a namespace based on the specified namespace id.
func (k *Keeper) Namespace(ctx context.Context, req *types.QueryNamespaceRequest) (*types.QueryNamespaceResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	namespaceId := getNamespaceId(req.Namespace)
	namespace := k.GetNamespace(ctx, namespaceId)
	if namespace == nil {
		return nil, status.Error(codes.NotFound, types.ErrNamespaceNotFound.Error())
	}

	return &types.QueryNamespaceResponse{Namespace: namespace}, nil
}

// NamespaceCollaborators query returns all namespace collaborators with pagination.
func (k *Keeper) NamespaceCollaborators(ctx context.Context, req *types.QueryNamespaceCollaboratorsRequest) (
	*types.QueryNamespaceCollaboratorsResponse, error) {

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	namespaceId := getNamespaceId(req.Namespace)
	if !k.hasNamespace(ctx, namespaceId) {
		return nil, status.Error(codes.NotFound, types.ErrNamespaceNotFound.Error())
	}

	namespaces, pageRes, err := k.getNamespaceCollaboratorsPaginated(ctx, namespaceId, req.Pagination)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryNamespaceCollaboratorsResponse{Collaborators: namespaces, Pagination: pageRes}, nil
}

// NamespacePosts query returns all namespace posts with pagination.
func (k *Keeper) NamespacePosts(ctx context.Context, req *types.QueryNamespacePostsRequest) (
	*types.QueryNamespacePostsResponse, error) {

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	namespaceId := getNamespaceId(req.Namespace)
	if !k.hasNamespace(ctx, namespaceId) {
		return nil, status.Error(codes.NotFound, types.ErrNamespaceNotFound.Error())
	}

	posts, pageRes, err := k.getNamespacePostsPaginated(ctx, namespaceId, req.Pagination)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryNamespacePostsResponse{Posts: posts, Pagination: pageRes}, nil
}

// Post query returns a post based on the specified namespace and id.
func (k *Keeper) Post(ctx context.Context, req *types.QueryPostRequest) (*types.QueryPostResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	namespaceId := getNamespaceId(req.Namespace)
	if !k.hasNamespace(ctx, namespaceId) {
		return nil, status.Error(codes.NotFound, types.ErrNamespaceNotFound.Error())
	}

	post := k.getPost(ctx, namespaceId, req.Id)
	if post == nil {
		return nil, status.Error(codes.NotFound, types.ErrPostNotFound.Error())
	}

	return &types.QueryPostResponse{Post: post}, nil
}

// Posts query returns all posts with pagination.
func (k *Keeper) Posts(ctx context.Context, req *types.QueryPostsRequest) (*types.QueryPostsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	posts, pageRes, err := k.getAllPostsPaginated(ctx, req.Pagination)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryPostsResponse{Posts: posts, Pagination: pageRes}, nil
}

// getNamespacesPaginated returns all namespaces with pagination.
func (k *Keeper) getNamespacesPaginated(ctx context.Context, pageReq *query.PageRequest) (
	[]types.Namespace, *query.PageResponse, error) {

	var namespaces []types.Namespace
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.NamespaceKeyPrefix))

	onResult := func(key []byte, value []byte) error {
		var namespace types.Namespace
		k.cdc.MustUnmarshal(value, &namespace)
		namespaces = append(namespaces, namespace)
		return nil
	}

	pageRes, err := query.Paginate(store, pageReq, onResult)
	if err != nil {
		return nil, nil, err
	}

	return namespaces, pageRes, nil
}

// getNamespaceCollaboratorsPaginated returns namespace collaborators with pagination.
func (k *Keeper) getNamespaceCollaboratorsPaginated(ctx context.Context, namespaceId string, pageReq *query.PageRequest) (
	[]types.Collaborator, *query.PageResponse, error) {

	var collaborators []types.Collaborator
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.CollaboratorKeyPrefix))

	onResult := func(key []byte, value []byte) error {
		if !bytes.HasPrefix(key, []byte(namespaceId+"/")) {
			return nil
		}
		var collaborator types.Collaborator
		k.cdc.MustUnmarshal(value, &collaborator)
		collaborators = append(collaborators, collaborator)
		return nil
	}

	pageRes, err := query.Paginate(store, pageReq, onResult)
	if err != nil {
		return nil, nil, err
	}

	return collaborators, pageRes, nil
}

// getNamespacePostsPaginated returns namespace posts with pagination.
func (k *Keeper) getNamespacePostsPaginated(ctx context.Context, namespaceId string, pageReq *query.PageRequest) (
	[]types.Post, *query.PageResponse, error) {

	var posts []types.Post
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.PostKeyPrefix))

	onResult := func(key []byte, value []byte) error {
		if !bytes.HasPrefix(key, []byte(namespaceId+"/")) {
			return nil
		}
		var post types.Post
		k.cdc.MustUnmarshal(value, &post)
		posts = append(posts, post)
		return nil
	}

	pageRes, err := query.Paginate(store, pageReq, onResult)
	if err != nil {
		return nil, nil, err
	}

	return posts, pageRes, nil
}

// getAllPostsPaginated returns all posts with pagination.
func (k *Keeper) getAllPostsPaginated(ctx context.Context, pageReq *query.PageRequest) (
	[]types.Post, *query.PageResponse, error) {

	var posts []types.Post
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.PostKeyPrefix))

	onResult := func(key []byte, value []byte) error {
		var post types.Post
		k.cdc.MustUnmarshal(value, &post)
		posts = append(posts, post)
		return nil
	}

	pageRes, err := query.Paginate(store, pageReq, onResult)
	if err != nil {
		return nil, nil, err
	}

	return posts, pageRes, nil
}
