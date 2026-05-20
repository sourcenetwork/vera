package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/sourcenetwork/sourcehub/x/orbis/types"
)

var _ types.MsgServer = &Keeper{}

// UpdateParams updates orbis module params.
func (k *Keeper) UpdateParams(ctx context.Context, req *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	if err := k.SetParams(ctx, req.Params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}

func (k *Keeper) CreateRing(goCtx context.Context, msg *types.MsgCreateRing) (*types.MsgCreateRingResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	namespaceID := namespaceID(msg.Namespace)
	creatorDID, err := k.GetAcpKeeper().GetActorDID(ctx, msg.Creator)
	if err != nil {
		return nil, err
	}

	var pssInterval *uint64
	if msg.HasPssInterval {
		v := msg.GetPssInterval()
		pssInterval = &v
	}

	ringID := types.GenerateRingID(namespaceID, msg.RingPk, msg.PeerIds, msg.Threshold, pssInterval, msg.PolicyId)
	if existing := k.GetRing(goCtx, ringID); existing != nil {
		return nil, types.ErrRingAlreadyExists
	}

	ring := types.Ring{
		Id:               ringID,
		Namespace:        namespaceID,
		CreatorDid:       creatorDID,
		RingPk:           msg.RingPk,
		PeerIds:          append([]string(nil), msg.PeerIds...),
		Threshold:        msg.Threshold,
		PolicyId:         msg.PolicyId,
		BlockNumberNonce: 0,
	}
	if pssInterval != nil {
		ring.PssInterval = *pssInterval
		ring.HasPssInterval = true
	}
	if err := validateRing(&ring); err != nil {
		return nil, err
	}

	k.SetRing(goCtx, ring)

	if err := ctx.EventManager().EmitTypedEvent(&types.EventRingCreated{
		Namespace:  namespaceID,
		RingId:     ringID,
		CreatorDid: creatorDID,
		Artifact:   msg.Artifact,
	}); err != nil {
		return nil, err
	}

	return &types.MsgCreateRingResponse{RingId: ringID}, nil
}

func (k *Keeper) UpdateRingByAcp(goCtx context.Context, msg *types.MsgUpdateRingByAcp) (*types.MsgUpdateRingByAcpResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	ring := k.GetRing(goCtx, msg.RingId)
	if ring == nil {
		return nil, types.ErrRingNotFound
	}
	if ring.PolicyId == "" {
		return nil, types.ErrRingMissingPolicyId
	}

	updaterDID, err := k.GetAcpKeeper().GetActorDID(ctx, msg.Creator)
	if err != nil {
		return nil, err
	}

	allowed, err := hasRingUpdatePermission(goCtx, k, ring, updaterDID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, types.ErrInvalidRingUpdater
	}

	var newThreshold *uint32
	if msg.HasNewThreshold {
		v := msg.GetNewThreshold()
		newThreshold = &v
	}
	if err := validateRingUpdate(msg.NewPeerIds, newThreshold, ring); err != nil {
		return nil, err
	}

	if len(msg.NewPeerIds) > 0 {
		ring.NewPeerIds = append([]string(nil), msg.NewPeerIds...)
	}
	if newThreshold != nil {
		ring.NewThreshold = *newThreshold
		ring.HasNewThreshold = true
	}
	if msg.HasPssInterval {
		ring.PssInterval = msg.GetPssInterval()
		ring.HasPssInterval = true
	}
	if err := validateRing(ring); err != nil {
		return nil, err
	}

	k.SetRing(goCtx, *ring)

	if err := ctx.EventManager().EmitTypedEvent(&types.EventRingUpdated{
		Namespace:  ring.Namespace,
		RingId:     ring.Id,
		UpdaterDid: updaterDID,
	}); err != nil {
		return nil, err
	}

	return &types.MsgUpdateRingByAcpResponse{}, nil
}

func (k *Keeper) FinalizeRingReshareByThresholdSignature(
	goCtx context.Context,
	msg *types.MsgFinalizeRingReshareByThresholdSignature,
) (*types.MsgFinalizeRingReshareByThresholdSignatureResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	ring := k.GetRing(goCtx, msg.RingId)
	if ring == nil {
		return nil, types.ErrRingNotFound
	}

	updaterDID, err := k.GetAcpKeeper().GetActorDID(ctx, msg.Creator)
	if err != nil {
		return nil, err
	}

	currentRingBytes, err := k.RingBytes(*ring)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrInvalidRing, "could not marshal current ring: %s", err)
	}

	signDocFinalizedRing, err := ringForReshareFinalization(ring)
	if err != nil {
		return nil, err
	}
	signDocFinalizedRingBytes, err := k.RingBytes(*signDocFinalizedRing)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrInvalidRing, "could not marshal finalized ring: %s", err)
	}

	signBytes, err := ringReshareFinalizeSignBytes(ctx.ChainID(), ring, currentRingBytes, signDocFinalizedRingBytes)
	if err != nil {
		return nil, err
	}
	if err := verifyThresholdSignatureForRingUpdate(ring, signBytes, msg.SignatureScheme, msg.Signature); err != nil {
		return nil, err
	}

	finalizedRing := *signDocFinalizedRing
	finalizedRing.BlockNumberNonce = uint64(ctx.BlockHeight())
	k.SetRing(goCtx, finalizedRing)

	if err := ctx.EventManager().EmitTypedEvent(&types.EventRingUpdated{
		Namespace:  finalizedRing.Namespace,
		RingId:     finalizedRing.Id,
		UpdaterDid: updaterDID,
	}); err != nil {
		return nil, err
	}

	return &types.MsgFinalizeRingReshareByThresholdSignatureResponse{}, nil
}

func (k *Keeper) StoreDocument(goCtx context.Context, msg *types.MsgStoreDocument) (*types.MsgStoreDocumentResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	namespaceID := namespaceID(msg.Namespace)
	if k.GetRing(goCtx, msg.RingId) == nil {
		return nil, types.ErrRingNotFound
	}

	creatorDID, err := k.GetAcpKeeper().GetActorDID(ctx, msg.Creator)
	if err != nil {
		return nil, err
	}

	var tier *string
	if msg.HasTier {
		v := msg.GetTier()
		tier = &v
	}
	var timestamp *uint64
	if msg.HasTimestamp {
		v := msg.GetTimestamp()
		timestamp = &v
	}

	documentID := types.GenerateDocumentID(namespaceID, msg.RingId, msg.Document, msg.Proof, msg.PolicyId, msg.Resource, msg.Permission, tier, timestamp)
	if existing := k.GetDocument(goCtx, namespaceID, documentID); existing != nil {
		return nil, types.ErrDocumentAlreadyExists
	}

	document := types.Document{
		Id:         documentID,
		Namespace:  namespaceID,
		CreatorDid: creatorDID,
		RingId:     msg.RingId,
		Document:   msg.Document,
		Proof:      msg.Proof,
		PolicyId:   msg.PolicyId,
		Resource:   msg.Resource,
		Permission: msg.Permission,
	}
	if tier != nil {
		document.Tier = *tier
		document.HasTier = true
	}
	if timestamp != nil {
		document.Timestamp = *timestamp
		document.HasTimestamp = true
	}
	if err := validateDocument(&document); err != nil {
		return nil, err
	}

	k.SetDocument(goCtx, document)

	if err := ctx.EventManager().EmitTypedEvent(&types.EventDocumentStored{
		Namespace:  namespaceID,
		DocumentId: documentID,
		CreatorDid: creatorDID,
	}); err != nil {
		return nil, err
	}

	return &types.MsgStoreDocumentResponse{DocumentId: documentID}, nil
}

func (k *Keeper) StoreKeyDerivation(goCtx context.Context, msg *types.MsgStoreKeyDerivation) (*types.MsgStoreKeyDerivationResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	namespaceID := namespaceID(msg.Namespace)
	if k.GetRing(goCtx, msg.RingId) == nil {
		return nil, types.ErrRingNotFound
	}

	creatorDID, err := k.GetAcpKeeper().GetActorDID(ctx, msg.Creator)
	if err != nil {
		return nil, err
	}

	keyDerivationID := types.GenerateKeyDerivationID(namespaceID, msg.RingId, msg.Derivation, msg.PolicyId, msg.Resource, msg.Permission)
	if existing := k.GetKeyDerivation(goCtx, namespaceID, keyDerivationID); existing != nil {
		return nil, types.ErrKeyDerivationAlreadyExists
	}

	keyDerivation := types.KeyDerivation{
		Id:         keyDerivationID,
		Namespace:  namespaceID,
		CreatorDid: creatorDID,
		RingId:     msg.RingId,
		Derivation: msg.Derivation,
		PolicyId:   msg.PolicyId,
		Resource:   msg.Resource,
		Permission: msg.Permission,
	}
	if err := validateKeyDerivation(&keyDerivation); err != nil {
		return nil, err
	}

	k.SetKeyDerivation(goCtx, keyDerivation)

	if err := ctx.EventManager().EmitTypedEvent(&types.EventKeyDerivationStored{
		Namespace:       namespaceID,
		KeyDerivationId: keyDerivationID,
		CreatorDid:      creatorDID,
	}); err != nil {
		return nil, err
	}

	return &types.MsgStoreKeyDerivationResponse{KeyDerivationId: keyDerivationID}, nil
}
