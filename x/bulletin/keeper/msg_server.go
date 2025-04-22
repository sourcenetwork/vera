package keeper

import (
	"context"
	"encoding/base64"
	"time"

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

	ownerDID, err := k.GetAcpKeeper().IssueDIDFromAccountAddr(goCtx, msg.Creator)
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

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventRegisterNamespace,
			sdk.NewAttribute(types.AttributeKeyNamespaceId, namespaceId),
			sdk.NewAttribute(types.AttributeKeyOwnerDid, ownerDID),
			sdk.NewAttribute(types.AttributeKeyCreatedAt, namespace.CreatedAt.Format(time.RFC3339)),
		),
	)

	return &types.MsgRegisterNamespaceResponse{Namespace: &namespace}, nil
}

// CreatePost adds a new post to the specified (existing) namespace.
// The signer must have permission to create posts in that namespace.
func (k *Keeper) CreatePost(goCtx context.Context, msg *types.MsgCreatePost) (*types.MsgCreatePostResponse, error) {
	policyId := k.GetPolicyId(goCtx)
	if policyId == "" {
		return nil, types.ErrInvalidPolicyId
	}

	namespaceId := getNamespaceId(msg.Namespace)
	if !k.hasNamespace(goCtx, namespaceId) {
		return nil, types.ErrNamespaceNotFound
	}

	creatorDID, err := k.GetAcpKeeper().IssueDIDFromAccountAddr(goCtx, msg.Creator)
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
		Proof:      msg.Proof,
	}
	k.SetPost(goCtx, post)

	b64Payload := base64.StdEncoding.EncodeToString(post.Payload)
	sdk.UnwrapSDKContext(goCtx).EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventCreatePost,
			sdk.NewAttribute(types.AttributeKeyNamespaceId, namespaceId),
			sdk.NewAttribute(types.AttributeKeyPostId, postId),
			sdk.NewAttribute(types.AttributeKeyCreatorDid, creatorDID),
			sdk.NewAttribute(types.AttributeKeyPayload, b64Payload),
		),
	)

	return &types.MsgCreatePostResponse{}, nil
}

// AddCollaborator adds a new collaborator to the specified namespace.
// The signer must have permission to manage collaborators of that namespace object.
func (k *Keeper) AddCollaborator(ctx context.Context, msg *types.MsgAddCollaborator) (*types.MsgAddCollaboratorResponse, error) {
	policyId := k.GetPolicyId(ctx)
	if policyId == "" {
		return nil, types.ErrInvalidPolicyId
	}

	namespaceId := getNamespaceId(msg.Namespace)
	if !k.hasNamespace(ctx, namespaceId) {
		return nil, types.ErrNamespaceNotFound
	}

	ownerDID, err := k.GetAcpKeeper().IssueDIDFromAccountAddr(ctx, msg.Creator)
	if err != nil {
		return nil, err
	}

	collaboratorDID, err := k.GetAcpKeeper().IssueDIDFromAccountAddr(ctx, msg.Collaborator)
	if err != nil {
		return nil, err
	}

	err = AddCollaborator(ctx, k, policyId, namespaceId, collaboratorDID, ownerDID, msg.Creator)
	if err != nil {
		return nil, err
	}

	collaborator := types.Collaborator{
		Address:   msg.Collaborator,
		Did:       collaboratorDID,
		Namespace: namespaceId,
	}
	k.SetCollaborator(ctx, collaborator)

	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventAddCollaborator,
			sdk.NewAttribute(types.AttributeKeyNamespaceId, namespaceId),
			sdk.NewAttribute(types.AttributeKeyCollaboratorDid, collaboratorDID),
			sdk.NewAttribute(types.AttributeKeyAddedBy, ownerDID),
		),
	)

	return &types.MsgAddCollaboratorResponse{}, nil
}

// RemoveCollaborator removes existing collaborator from the specified namespace.
// The signer must have permission to manage collaborators of that namespace object.
func (k *Keeper) RemoveCollaborator(ctx context.Context, msg *types.MsgRemoveCollaborator) (*types.MsgRemoveCollaboratorResponse, error) {
	policyId := k.GetPolicyId(ctx)
	if policyId == "" {
		return nil, types.ErrInvalidPolicyId
	}

	namespaceId := getNamespaceId(msg.Namespace)
	if !k.hasNamespace(ctx, namespaceId) {
		return nil, types.ErrNamespaceNotFound
	}

	ownerDID, err := k.GetAcpKeeper().IssueDIDFromAccountAddr(ctx, msg.Creator)
	if err != nil {
		return nil, err
	}

	collaboratorDID, err := k.GetAcpKeeper().IssueDIDFromAccountAddr(ctx, msg.Collaborator)
	if err != nil {
		return nil, err
	}

	err = deleteCollaborator(ctx, k, policyId, namespaceId, collaboratorDID, ownerDID, msg.Creator)
	if err != nil {
		return nil, err
	}

	k.DeleteCollaborator(ctx, namespaceId, collaboratorDID)

	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventRemoveCollaborator,
			sdk.NewAttribute(types.AttributeKeyNamespaceId, namespaceId),
			sdk.NewAttribute(types.AttributeKeyCollaboratorDid, collaboratorDID),
		),
	)

	return &types.MsgRemoveCollaboratorResponse{}, nil
}
