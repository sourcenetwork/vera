package keeper

import (
	"context"
	"encoding/base64"
	"encoding/json"

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
// Post creation is unrestricted: any valid signer may create a post in any
// registered namespace. Duplicate (namespace, payload) pairs are rejected
// because post_id is derived from them.
func (k *Keeper) CreatePost(goCtx context.Context, msg *types.MsgCreatePost) (*types.MsgCreatePostResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	namespaceId := getNamespaceId(msg.Namespace)
	if !k.hasNamespace(goCtx, namespaceId) {
		return nil, types.ErrNamespaceNotFound
	}

	creatorDID, err := k.GetAcpKeeper().GetActorDID(ctx, msg.Creator)
	if err != nil {
		return nil, err
	}

	postId := types.GeneratePostId(namespaceId, msg.Payload)

	if existing := k.getPost(goCtx, namespaceId, postId); existing != nil {
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

// UpdateRingPostByAcp updates the new_peer_ids, new_threshold, and/or
// pss_interval fields of a ring post. The existing post must be a valid ring
// payload containing a policy_id. The caller must hold update_post permission
// in that policy.
func (k *Keeper) UpdateRingPostByAcp(goCtx context.Context, msg *types.MsgUpdateRingPostByAcp) (*types.MsgUpdateRingPostByAcpResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	namespaceId := getNamespaceId(msg.Namespace)
	if !k.hasNamespace(goCtx, namespaceId) {
		return nil, types.ErrNamespaceNotFound
	}

	existing := k.getPost(goCtx, namespaceId, msg.PostId)
	if existing == nil {
		return nil, types.ErrPostNotFound
	}

	ringPayload, err := parseRingPayloadJSON(existing.Payload)
	if err != nil {
		return nil, err
	}

	if ringPayload.PolicyID == nil {
		return nil, types.ErrRingPayloadMissingPolicyId
	}
	policyId := *ringPayload.PolicyID

	updaterDID, err := k.GetAcpKeeper().GetActorDID(ctx, msg.Creator)
	if err != nil {
		return nil, err
	}

	allowed, err := hasPermission(goCtx, k, policyId, namespaceId, types.UpdatePostPermission, updaterDID, msg.Creator)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, types.ErrInvalidPostUpdater
	}

	var newThreshold *uint32
	if msg.XNewThreshold != nil {
		v := msg.GetNewThreshold()
		newThreshold = &v
	}
	if err := validateRingPostUpdate(msg.NewPeerIds, newThreshold); err != nil {
		return nil, err
	}

	if len(msg.NewPeerIds) > 0 {
		ringPayload.NewPeerIDs = &msg.NewPeerIds
	}
	if newThreshold != nil {
		ringPayload.NewThreshold = newThreshold
	}
	if msg.XPssInterval != nil {
		v := msg.GetPssInterval()
		ringPayload.PSSInterval = &v
	}

	updatedPayload, err := json.Marshal(ringPayload)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrInvalidPostPayload, "could not marshal updated ring payload: %s", err)
	}

	existing.Payload = updatedPayload
	k.SetPost(goCtx, *existing)

	b64Payload := base64.StdEncoding.EncodeToString(existing.Payload)
	if err := ctx.EventManager().EmitTypedEvent(&types.EventPostUpdated{
		NamespaceId: namespaceId,
		PostId:      existing.Id,
		UpdaterDid:  updaterDID,
		Payload:     b64Payload,
		Artifact:    msg.Artifact,
	}); err != nil {
		return nil, err
	}

	return &types.MsgUpdateRingPostByAcpResponse{}, nil
}

// UpdateRingPostByThresholdSignature finalizes a reshare using only the current
// bulletin payload as state. It moves new_peer_ids/new_threshold into
// peer_ids/threshold, clears the new_* fields, and verifies a threshold
// signature over the canonical transition sign doc against the current
// payload's ring_pk. Once accepted, the stored payload's block_number_nonce is
// set to the current block height.
// This path does not perform the ACP collaborator permission check used by
// UpdatePost.
func (k *Keeper) UpdateRingPostByThresholdSignature(
	goCtx context.Context,
	msg *types.MsgUpdateRingPostByThresholdSignature,
) (*types.MsgUpdateRingPostByThresholdSignatureResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	namespaceId := getNamespaceId(msg.Namespace)
	if !k.hasNamespace(goCtx, namespaceId) {
		return nil, types.ErrNamespaceNotFound
	}

	existing := k.getPost(goCtx, namespaceId, msg.PostId)
	if existing == nil {
		return nil, types.ErrPostNotFound
	}

	updaterDID, err := k.GetAcpKeeper().GetActorDID(ctx, msg.Creator)
	if err != nil {
		return nil, err
	}

	currentRingPayload, err := parseRingPayloadJSON(existing.Payload)
	if err != nil {
		return nil, err
	}

	signDocFinalizedPayload, err := deriveFinalizedRingPayloadReshare(currentRingPayload)
	if err != nil {
		return nil, err
	}

	signBytes, err := ringReshareFinalizeSignBytes(ctx.ChainID(), msg.Namespace, msg.PostId, existing.Payload, signDocFinalizedPayload, currentRingPayload)
	if err != nil {
		return nil, err
	}

	if err := verifyThresholdSignatureForRingPayloadUpdate(currentRingPayload, signBytes, msg.SignatureScheme, msg.Signature); err != nil {
		return nil, err
	}

	finalizedPayload, err := finalizeRingPayloadReshare(currentRingPayload, uint64(ctx.BlockHeight()))
	if err != nil {
		return nil, err
	}

	existing.Payload = finalizedPayload
	k.SetPost(goCtx, *existing)

	b64Payload := base64.StdEncoding.EncodeToString(existing.Payload)
	if err := ctx.EventManager().EmitTypedEvent(&types.EventPostUpdated{
		NamespaceId: namespaceId,
		PostId:      existing.Id,
		UpdaterDid:  updaterDID,
		Payload:     b64Payload,
		Artifact:    msg.Artifact,
	}); err != nil {
		return nil, err
	}

	return &types.MsgUpdateRingPostByThresholdSignatureResponse{}, nil
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
