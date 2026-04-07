package keeper

import (
	"context"
	"encoding/base64"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sourcenetwork/sourcehub/x/bulletin/types"
)

var _ types.MsgServer = &Keeper{}

// UpdateParams updates bulletin module params.
// Request authority must match module authority.
func (k *Keeper) UpdateParams(ctx context.Context, req *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	if err := k.SetParams(ctx, req.Params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}

// RegisterNamespace registers a new namespace resource under the genesis bulletin policy.
// The namespace must have a unique, non-existent namespaceId.
func (k *Keeper) RegisterNamespace(goCtx context.Context, msg *types.MsgRegisterNamespace) (*types.MsgRegisterNamespaceResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Create module policy and claim capability if it does not exist yet
	policyId, err := k.EnsurePolicy(ctx)
	if err != nil {
		return nil, types.ErrCouldNotEnsurePolicy
	}

	namespaceId := getNamespaceId(msg.Namespace)
	if k.hasNamespace(goCtx, namespaceId) {
		return nil, types.ErrNamespaceAlreadyExists
	}

	ownerDID, err := k.GetAcpKeeper().GetActorDID(ctx, msg.Creator)
	if err != nil {
		return nil, err
	}

	err = RegisterNamespace(ctx, k, policyId, namespaceId, ownerDID, msg.Creator)
	if err != nil {
		return nil, err
	}

	namespace := types.Namespace{
		Id:        namespaceId,
		OwnerDid:  ownerDID,
		Creator:   msg.Creator,
		CreatedAt: ctx.BlockTime(),
	}
	k.SetNamespace(goCtx, namespace)

	if err := ctx.EventManager().EmitTypedEvent(&types.EventNamespaceRegistered{
		NamespaceId: namespaceId,
		OwnerDid:    ownerDID,
		CreatedAt:   namespace.CreatedAt,
	}); err != nil {
		return nil, err
	}

	return &types.MsgRegisterNamespaceResponse{Namespace: &namespace}, nil
}

// CreatePost adds a new post to the specified (existing) namespace.
// The signer must have permission to create posts in that namespace.
func (k *Keeper) CreatePost(goCtx context.Context, msg *types.MsgCreatePost) (*types.MsgCreatePostResponse, error) {
	policyId := k.GetPolicyId(goCtx)
	if policyId == "" {
		return nil, types.ErrInvalidPolicyId
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	namespaceId := getNamespaceId(msg.Namespace)
	if !k.hasNamespace(goCtx, namespaceId) {
		return nil, types.ErrNamespaceNotFound
	}

	creatorDID, err := k.GetAcpKeeper().GetActorDID(ctx, msg.Creator)
	if err != nil {
		return nil, err
	}

	hasPermission, err := hasPermission(goCtx, k, policyId, namespaceId, types.CreatePostPermission, creatorDID, msg.Creator)
	if err != nil {
		return nil, err
	}
	if !hasPermission {
		return nil, types.ErrInvalidPostCreator
	}

	postId := types.GeneratePostId(namespaceId, msg.Payload)

	existingPost := k.getPost(goCtx, namespaceId, postId)
	if existingPost != nil {
		return nil, types.ErrPostAlreadyExists
	}

	post := types.Post{
		Id:         postId,
		Namespace:  namespaceId,
		CreatorDid: creatorDID,
		Payload:    msg.Payload,
	}
	k.SetPost(goCtx, post)

	b64Payload := base64.StdEncoding.EncodeToString(post.Payload)
	if err := ctx.EventManager().EmitTypedEvent(&types.EventPostCreated{
		NamespaceId: namespaceId,
		PostId:      postId,
		CreatorDid:  creatorDID,
		Payload:     b64Payload,
		Artifact:    msg.Artifact,
	}); err != nil {
		return nil, err
	}

	return &types.MsgCreatePostResponse{}, nil
}

// AddCollaborator adds a new collaborator to the specified namespace.
// The signer must have permission to manage collaborators of that namespace object.
func (k *Keeper) AddCollaborator(goCtx context.Context, msg *types.MsgAddCollaborator) (*types.MsgAddCollaboratorResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	policyId := k.GetPolicyId(goCtx)
	if policyId == "" {
		return nil, types.ErrInvalidPolicyId
	}

	namespaceId := getNamespaceId(msg.Namespace)
	if !k.hasNamespace(goCtx, namespaceId) {
		return nil, types.ErrNamespaceNotFound
	}

	ownerDID, err := k.GetAcpKeeper().GetActorDID(ctx, msg.Creator)
	if err != nil {
		return nil, err
	}

	collaboratorDID, err := k.GetAcpKeeper().IssueDIDFromAccountAddr(ctx, msg.Collaborator)
	if err != nil {
		return nil, err
	}

	err = AddCollaborator(goCtx, k, policyId, namespaceId, collaboratorDID, ownerDID, msg.Creator)
	if err != nil {
		return nil, err
	}

	collaborator := types.Collaborator{
		Address:   msg.Collaborator,
		Did:       collaboratorDID,
		Namespace: namespaceId,
	}
	k.SetCollaborator(goCtx, collaborator)

	if err := ctx.EventManager().EmitTypedEvent(&types.EventCollaboratorAdded{
		NamespaceId:     namespaceId,
		CollaboratorDid: collaboratorDID,
		AddedBy:         ownerDID,
	}); err != nil {
		return nil, err
	}

	return &types.MsgAddCollaboratorResponse{}, nil
}

// RemoveCollaborator removes existing collaborator from the specified namespace.
// The signer must have permission to manage collaborators of that namespace object.
func (k *Keeper) RemoveCollaborator(goCtx context.Context, msg *types.MsgRemoveCollaborator) (*types.MsgRemoveCollaboratorResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	policyId := k.GetPolicyId(goCtx)
	if policyId == "" {
		return nil, types.ErrInvalidPolicyId
	}

	namespaceId := getNamespaceId(msg.Namespace)
	if !k.hasNamespace(goCtx, namespaceId) {
		return nil, types.ErrNamespaceNotFound
	}

	ownerDID, err := k.GetAcpKeeper().GetActorDID(ctx, msg.Creator)
	if err != nil {
		return nil, err
	}

	collaboratorDID, err := k.GetAcpKeeper().IssueDIDFromAccountAddr(ctx, msg.Collaborator)
	if err != nil {
		return nil, err
	}

	err = deleteCollaborator(goCtx, k, policyId, namespaceId, collaboratorDID, ownerDID, msg.Creator)
	if err != nil {
		return nil, err
	}

	k.DeleteCollaborator(goCtx, namespaceId, collaboratorDID)

	if err := ctx.EventManager().EmitTypedEvent(&types.EventCollaboratorRemoved{
		NamespaceId:     namespaceId,
		CollaboratorDid: collaboratorDID,
	}); err != nil {
		return nil, err
	}

	return &types.MsgRemoveCollaboratorResponse{}, nil
}
