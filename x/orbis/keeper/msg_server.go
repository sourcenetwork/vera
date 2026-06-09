package keeper

import (
	"context"
	"encoding/hex"
	"slices"

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

	creatorDID, err := k.GetAcpKeeper().GetActorDID(ctx, msg.Creator)
	if err != nil {
		return nil, err
	}

	pssInterval := optionalCreateRingPSSInterval(msg)
	nonce := optionalCreateRingNonce(msg)

	for _, nodeKey := range msg.PeerNodeKeys {
		if k.GetNodeInfo(goCtx, nodeKey) == nil {
			return nil, errorsmod.Wrapf(types.ErrInvalidRing, "peer_node_key %q has no registered node info", nodeKey)
		}
	}

	ringID := types.GenerateRingID(msg.PeerNodeKeys, msg.Threshold, pssInterval, msg.PolicyId, nonce, msg.CurrentVersion)
	if existing := k.GetRing(goCtx, ringID); existing != nil {
		return nil, types.ErrRingAlreadyExists
	}

	ring := types.Ring{
		Id:               ringID,
		CreatorDid:       creatorDID,
		PeerNodeKeys:     append([]string(nil), msg.PeerNodeKeys...),
		Threshold:        msg.Threshold,
		PolicyId:         msg.PolicyId,
		BlockNumberNonce: 0,
		UpgradeInfo: types.UpgradeInfo{
			CurrentVersion: msg.CurrentVersion,
		},
	}
	setRingPSSInterval(&ring, pssInterval)
	if err := validateRing(&ring); err != nil {
		return nil, err
	}

	if err := k.ensureRingCreatePermission(goCtx, msg.PolicyId, creatorDID); err != nil {
		return nil, err
	}

	if err := k.registerRingACPObject(goCtx, msg.Creator, msg.PolicyId, ringID); err != nil {
		return nil, err
	}

	k.SetRing(goCtx, ring)

	if err := ctx.EventManager().EmitTypedEvent(&types.EventRingCreated{
		RingId:     ringID,
		CreatorDid: creatorDID,
	}); err != nil {
		return nil, err
	}

	return &types.MsgCreateRingResponse{RingId: ringID}, nil
}

func (k *Keeper) FinalizeRing(goCtx context.Context, msg *types.MsgFinalizeRing) (*types.MsgFinalizeRingResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	ring := k.GetRing(goCtx, msg.RingId)
	if ring == nil {
		return nil, types.ErrRingNotFound
	}
	if ring.RingPk != "" {
		return nil, types.ErrRingAlreadyFinalized
	}

	signerKey, err := signerPublicKeyHex(ctx, k, msg.Creator)
	if err != nil {
		return nil, err
	}
	authorized := false
	for _, nodeKey := range ring.PeerNodeKeys {
		if nodeKey == signerKey {
			authorized = true
			break
		}
	}
	if !authorized {
		return nil, types.ErrInvalidRingFinalizer
	}

	// Reject double confirmations from the same node before checking for
	// pk conflicts. Without this ordering a node that already confirmed
	// correctly could delete the ring by re-submitting with a different
	// ring_pk — the conflict check would fire first and wipe the ring.
	for _, c := range ring.Confirmations {
		if c.NodeKey == signerKey {
			return nil, types.ErrDuplicateConfirmation
		}
	}

	// Check for a conflicting ring_pk from a prior confirmation by a
	// different node. This is a genuine BFT violation — delete the ring.
	for _, c := range ring.Confirmations {
		if c.RingPk != msg.RingPk {
			k.DeleteRing(goCtx, ring.Id)
			return nil, types.ErrRingPkConflict
		}
	}

	ring.Confirmations = append(ring.Confirmations, &types.RingConfirmation{
		NodeKey: signerKey,
		RingPk:  msg.RingPk,
	})

	if len(ring.Confirmations) < len(ring.PeerNodeKeys) {
		k.SetRing(goCtx, *ring)
		return &types.MsgFinalizeRingResponse{}, nil
	}

	// All nodes confirmed — finalize.
	finalizerDID, err := k.GetAcpKeeper().GetActorDID(ctx, msg.Creator)
	if err != nil {
		return nil, err
	}

	ring.RingPk = msg.RingPk
	ring.Confirmations = nil
	if err := validateRing(ring); err != nil {
		return nil, err
	}

	k.SetRing(goCtx, *ring)

	if err := ctx.EventManager().EmitTypedEvent(&types.EventRingUpdated{
		RingId:     ring.Id,
		UpdaterDid: finalizerDID,
	}); err != nil {
		return nil, err
	}

	return &types.MsgFinalizeRingResponse{}, nil
}

func (k *Keeper) UpdateRingByAcp(goCtx context.Context, msg *types.MsgUpdateRingByAcp) (*types.MsgUpdateRingByAcpResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	ring := k.GetRing(goCtx, msg.RingId)
	if ring == nil {
		return nil, types.ErrRingNotFound
	}
	if err := requireRingFinalized(ring); err != nil {
		return nil, err
	}

	updaterDID, err := k.GetAcpKeeper().GetActorDID(ctx, msg.Creator)
	if err != nil {
		return nil, err
	}
	if err := k.ensureRingUpdatePermission(goCtx, ring, updaterDID); err != nil {
		return nil, err
	}

	newThreshold := optionalUpdateRingNewThreshold(msg)
	nextVersion := optionalUpdateRingNextVersion(msg)
	activationHeight := optionalUpdateRingActivationHeight(msg)
	normalized, previousVersion, normalizedActivationHeight := normalizeRingUpgrade(ring, ctx.BlockHeight())
	hadPendingUpgrade := ring.UpgradeInfo.XNextVersion != nil
	if err := validateRingUpdate(
		msg.NewPeerNodeKeys,
		newThreshold,
		nextVersion,
		activationHeight,
		msg.ClearUpgrade,
		ctx.BlockHeight(),
		ring,
	); err != nil {
		return nil, err
	}
	for _, nodeKey := range msg.NewPeerNodeKeys {
		nodeInfo := k.GetNodeInfo(goCtx, nodeKey)
		if nodeInfo == nil {
			return nil, errorsmod.Wrapf(types.ErrInvalidRing, "peer_node_key %q has no registered node info", nodeKey)
		}
		if !nodeInfoAllowsRing(nodeInfo, ring) {
			return nil, errorsmod.Wrapf(
				types.ErrInvalidRing,
				"peer_node_key %q is not whitelisted for policy_id %q or ring_id %q",
				nodeKey,
				ring.PolicyId,
				ring.Id,
			)
		}
	}

	if len(msg.NewPeerNodeKeys) > 0 {
		ring.NewPeerNodeKeys = append([]string(nil), msg.NewPeerNodeKeys...)
	}
	if newThreshold.HasValue() {
		setRingNewThreshold(ring, newThreshold)
	}
	if msg.XPssInterval != nil {
		setRingPSSInterval(ring, optionalUpdateRingPSSInterval(msg))
	}
	if msg.ClearUpgrade {
		clearRingUpgrade(ring)
	}
	if nextVersion.HasValue() {
		setRingUpgrade(ring, nextVersion.Value(), activationHeight.Value())
	}
	if err := validateRing(ring); err != nil {
		return nil, err
	}

	k.SetRing(goCtx, *ring)

	if err := ctx.EventManager().EmitTypedEvent(&types.EventRingUpdated{
		RingId:     ring.Id,
		UpdaterDid: updaterDID,
	}); err != nil {
		return nil, err
	}
	if normalized {
		if err := ctx.EventManager().EmitTypedEvent(&types.EventRingUpgradeNormalized{
			RingId:           ring.Id,
			PreviousVersion:  previousVersion,
			CurrentVersion:   ring.UpgradeInfo.CurrentVersion,
			ActivationHeight: normalizedActivationHeight,
		}); err != nil {
			return nil, err
		}
	}
	if nextVersion.HasValue() {
		if err := ctx.EventManager().EmitTypedEvent(&types.EventRingUpgradeScheduled{
			RingId:           ring.Id,
			CurrentVersion:   ring.UpgradeInfo.CurrentVersion,
			NextVersion:      nextVersion.Value(),
			ActivationHeight: activationHeight.Value(),
		}); err != nil {
			return nil, err
		}
	} else if msg.ClearUpgrade && hadPendingUpgrade && !normalized {
		if err := ctx.EventManager().EmitTypedEvent(&types.EventRingUpgradeCancelled{
			RingId:         ring.Id,
			CurrentVersion: ring.UpgradeInfo.CurrentVersion,
		}); err != nil {
			return nil, err
		}
	}

	return &types.MsgUpdateRingByAcpResponse{}, nil
}

func nodeInfoAllowsRing(nodeInfo *types.NodeInfo, ring *types.Ring) bool {
	return slices.Contains(nodeInfo.WhitelistedPolicyIds, ring.PolicyId) ||
		slices.Contains(nodeInfo.WhitelistedRingIds, ring.Id)
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

	signDocFinalizedRing, err := ringForReshareFinalization(ring)
	if err != nil {
		return nil, err
	}

	signBytes, err := ringReshareFinalizeSignBytes(ctx.ChainID(), ring, signDocFinalizedRing)
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
		RingId:     finalizedRing.Id,
		UpdaterDid: updaterDID,
	}); err != nil {
		return nil, err
	}

	return &types.MsgFinalizeRingReshareByThresholdSignatureResponse{}, nil
}

func (k *Keeper) StoreDocument(goCtx context.Context, msg *types.MsgStoreDocument) (*types.MsgStoreDocumentResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	ring := k.GetRing(goCtx, msg.RingId)
	if ring == nil {
		return nil, types.ErrRingNotFound
	}
	if err := requireRingFinalized(ring); err != nil {
		return nil, err
	}

	creatorDID, err := k.GetAcpKeeper().GetActorDID(ctx, msg.Creator)
	if err != nil {
		return nil, err
	}

	tier := optionalStoreDocumentTier(msg)
	timestamp := optionalStoreDocumentTimestamp(msg)

	documentID := types.GenerateDocumentID(msg.RingId, msg.Document, msg.Proof, msg.PolicyId, msg.Resource, msg.Permission, tier, timestamp)
	if existing := k.GetDocument(goCtx, documentID); existing != nil {
		return nil, types.ErrDocumentAlreadyExists
	}

	document := types.Document{
		Id:         documentID,
		CreatorDid: creatorDID,
		RingId:     msg.RingId,
		Document:   msg.Document,
		Proof:      msg.Proof,
		PolicyId:   msg.PolicyId,
		Resource:   msg.Resource,
		Permission: msg.Permission,
	}
	setDocumentTier(&document, tier)
	setDocumentTimestamp(&document, timestamp)
	if err := validateDocument(&document); err != nil {
		return nil, err
	}

	k.SetDocument(goCtx, document)

	if err := ctx.EventManager().EmitTypedEvent(&types.EventDocumentStored{
		DocumentId: documentID,
		CreatorDid: creatorDID,
	}); err != nil {
		return nil, err
	}

	return &types.MsgStoreDocumentResponse{DocumentId: documentID}, nil
}

func (k *Keeper) CreateNodeInfo(goCtx context.Context, msg *types.MsgCreateNodeInfo) (*types.MsgCreateNodeInfoResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	nodeKey, err := signerPublicKeyHex(ctx, k, msg.Creator)
	if err != nil {
		return nil, err
	}

	if existing := k.GetNodeInfo(goCtx, nodeKey); existing != nil {
		return nil, types.ErrNodeInfoAlreadyExists
	}

	nodeInfo := types.NodeInfo{
		PeerId:               msg.PeerId,
		ControllerKey:        msg.ControllerKey,
		WhitelistedPolicyIds: append([]string(nil), msg.WhitelistedPolicyIds...),
		WhitelistedRingIds:   append([]string(nil), msg.WhitelistedRingIds...),
	}
	if err := validateNodeInfo(&nodeInfo); err != nil {
		return nil, err
	}

	k.SetNodeInfo(goCtx, nodeKey, nodeInfo)

	if err := ctx.EventManager().EmitTypedEvent(&types.EventNodeInfoCreated{
		PeerId:        nodeInfo.PeerId,
		ControllerKey: nodeInfo.ControllerKey,
	}); err != nil {
		return nil, err
	}

	return &types.MsgCreateNodeInfoResponse{}, nil
}

func (k *Keeper) UpdateNodeInfo(goCtx context.Context, msg *types.MsgUpdateNodeInfo) (*types.MsgUpdateNodeInfoResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	nodeInfo := k.GetNodeInfo(goCtx, msg.NodeKey)
	if nodeInfo == nil {
		return nil, types.ErrNodeInfoNotFound
	}

	signerKey, err := signerPublicKeyHex(ctx, k, msg.Creator)
	if err != nil {
		return nil, err
	}
	if signerKey != nodeInfo.ControllerKey {
		return nil, types.ErrUnauthorizedNodeInfoUpdate
	}

	if msg.XPeerId != nil {
		nodeInfo.PeerId = msg.GetPeerId()
	}
	if msg.XControllerKey != nil {
		nodeInfo.ControllerKey = msg.GetControllerKey()
	}
	nodeInfo.WhitelistedPolicyIds = append([]string(nil), msg.WhitelistedPolicyIds...)
	nodeInfo.WhitelistedRingIds = append([]string(nil), msg.WhitelistedRingIds...)
	if err := validateNodeInfo(nodeInfo); err != nil {
		return nil, err
	}

	k.SetNodeInfo(goCtx, msg.NodeKey, *nodeInfo)

	if err := ctx.EventManager().EmitTypedEvent(&types.EventNodeInfoUpdated{
		PeerId:        nodeInfo.PeerId,
		ControllerKey: nodeInfo.ControllerKey,
	}); err != nil {
		return nil, err
	}

	return &types.MsgUpdateNodeInfoResponse{}, nil
}

// signerPublicKeyHex returns the hex-encoded compressed public key for a bech32 address.
// The ante handler populates the account's public key before message handlers run,
// so it is guaranteed non-nil for the transaction signer.
func signerPublicKeyHex(ctx sdk.Context, k *Keeper, address string) (string, error) {
	addr, err := sdk.AccAddressFromBech32(address)
	if err != nil {
		return "", errorsmod.Wrapf(types.ErrInvalidNodeInfo, "invalid signer address: %s", err)
	}

	account := k.accountKeeper.GetAccount(ctx, addr)
	if account == nil {
		return "", errorsmod.Wrapf(types.ErrInvalidNodeInfo, "account not found for address %s", address)
	}

	pubKey := account.GetPubKey()
	if pubKey == nil {
		return "", errorsmod.Wrapf(types.ErrInvalidNodeInfo, "public key not set for account %s", address)
	}

	return hex.EncodeToString(pubKey.Bytes()), nil
}

func (k *Keeper) StoreKeyDerivation(goCtx context.Context, msg *types.MsgStoreKeyDerivation) (*types.MsgStoreKeyDerivationResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	ring := k.GetRing(goCtx, msg.RingId)
	if ring == nil {
		return nil, types.ErrRingNotFound
	}
	if err := requireRingFinalized(ring); err != nil {
		return nil, err
	}

	creatorDID, err := k.GetAcpKeeper().GetActorDID(ctx, msg.Creator)
	if err != nil {
		return nil, err
	}

	keyDerivationID := types.GenerateKeyDerivationID(msg.RingId, msg.Derivation, msg.PolicyId, msg.Resource, msg.Permission)
	if existing := k.GetKeyDerivation(goCtx, keyDerivationID); existing != nil {
		return nil, types.ErrKeyDerivationAlreadyExists
	}

	keyDerivation := types.KeyDerivation{
		Id:         keyDerivationID,
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
		KeyDerivationId: keyDerivationID,
		CreatorDid:      creatorDID,
	}); err != nil {
		return nil, err
	}

	return &types.MsgStoreKeyDerivationResponse{KeyDerivationId: keyDerivationID}, nil
}
