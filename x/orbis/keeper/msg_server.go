package keeper

import (
	"context"
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

	nonce := optionalCreateRingNonce(msg)
	peerNodeKeys := canonicalNodeKeys(msg.PeerNodeKeys)

	for _, nodeKey := range peerNodeKeys {
		if k.GetNodeInfo(goCtx, nodeKey) == nil {
			return nil, errorsmod.Wrapf(types.ErrInvalidRing, "peer_node_key %q has no registered node info", nodeKey)
		}
	}

	ringID := types.GenerateRingID(peerNodeKeys, msg.Threshold, msg.PssInterval, msg.PolicyId, nonce, msg.CurrentVersion)
	if existing := k.GetRing(goCtx, ringID); existing != nil {
		return nil, types.ErrRingAlreadyExists
	}

	reporting := msg.Reporting
	if reporting == nil {
		params := k.GetParams(goCtx)
		defaultReporting := types.ReportingConfigFromDefaults(params.DefaultReporting)
		reporting = &defaultReporting
	}

	ring := types.Ring{
		Id:               ringID,
		CreatorDid:       creatorDID,
		PeerNodeKeys:     peerNodeKeys,
		Threshold:        msg.Threshold,
		PssInterval:      msg.PssInterval,
		PolicyId:         msg.PolicyId,
		BlockNumberNonce: 0,
		UpgradeInfo: types.UpgradeInfo{
			CurrentVersion: msg.CurrentVersion,
		},
		Reporting: *reporting,
	}
	if err := validateRingPSSInterval(&ring); err != nil {
		return nil, err
	}
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
			if err := ctx.EventManager().EmitTypedEvent(&types.EventRingDeleted{
				RingId: ring.Id,
				Reason: "ring_pk_conflict",
			}); err != nil {
				return nil, err
			}
			return &types.MsgFinalizeRingResponse{
				Outcome: types.FinalizeRingOutcome_CONFLICT_DELETED,
			}, nil
		}
	}

	ring.Confirmations = append(ring.Confirmations, &types.RingConfirmation{
		NodeKey: signerKey,
		RingPk:  msg.RingPk,
	})

	if len(ring.Confirmations) < len(ring.PeerNodeKeys) {
		k.SetRing(goCtx, *ring)
		return &types.MsgFinalizeRingResponse{
			Outcome: types.FinalizeRingOutcome_CONFIRMATION_RECORDED,
		}, nil
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

	return &types.MsgFinalizeRingResponse{
		Outcome: types.FinalizeRingOutcome_RING_FINALIZED,
	}, nil
}

func (k *Keeper) CancelPendingRing(goCtx context.Context, msg *types.MsgCancelPendingRing) (*types.MsgCancelPendingRingResponse, error) {
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

	authorized := slices.Contains(ring.PeerNodeKeys, signerKey)
	if !authorized {
		cancellerDID, err := k.GetAcpKeeper().GetActorDID(ctx, msg.Creator)
		if err != nil {
			return nil, err
		}
		authorized = cancellerDID == ring.CreatorDid
	}
	if !authorized {
		return nil, types.ErrInvalidRingCanceller
	}

	k.DeleteRing(goCtx, ring.Id)
	if err := ctx.EventManager().EmitTypedEvent(&types.EventRingDeleted{
		RingId: ring.Id,
		Reason: "dkg_cancelled",
	}); err != nil {
		return nil, err
	}

	return &types.MsgCancelPendingRingResponse{}, nil
}

func (k *Keeper) StartRingReshareByAcp(goCtx context.Context, msg *types.MsgStartRingReshareByAcp) (*types.MsgStartRingReshareByAcpResponse, error) {
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

	upgradeNormalizedEvent, err := maybeNormalizeMaturedRingUpgrade(ring, ctx)
	if err != nil {
		return nil, err
	}

	newPeerNodeKeys := canonicalNodeKeys(msg.NewPeerNodeKeys)
	newThreshold := optionalStartRingReshareNewThreshold(msg)
	if err := validateStartRingReshare(
		newPeerNodeKeys,
		newThreshold,
		ring,
	); err != nil {
		return nil, err
	}
	for _, nodeKey := range newPeerNodeKeys {
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

	if len(newPeerNodeKeys) > 0 {
		ring.NewPeerNodeKeys = newPeerNodeKeys
	}
	if newThreshold.HasValue() {
		setRingNewThreshold(ring, newThreshold)
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
	if upgradeNormalizedEvent != nil {
		if err := ctx.EventManager().EmitTypedEvent(upgradeNormalizedEvent); err != nil {
			return nil, err
		}
	}

	return &types.MsgStartRingReshareByAcpResponse{}, nil
}

func (k *Keeper) SetRingPssIntervalByAcp(goCtx context.Context, msg *types.MsgSetRingPssIntervalByAcp) (*types.MsgSetRingPssIntervalByAcpResponse, error) {
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

	upgradeNormalizedEvent, err := maybeNormalizeMaturedRingUpgrade(ring, ctx)
	if err != nil {
		return nil, err
	}

	nextRing := *ring
	nextRing.PssInterval = msg.PssInterval
	if err := validateRingPSSInterval(&nextRing); err != nil {
		return nil, err
	}
	if ring.GetPssInterval() == msg.PssInterval {
		return nil, types.ErrRingPssIntervalUnchanged
	}

	ring.PssInterval = msg.PssInterval
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
	if upgradeNormalizedEvent != nil {
		if err := ctx.EventManager().EmitTypedEvent(upgradeNormalizedEvent); err != nil {
			return nil, err
		}
	}

	return &types.MsgSetRingPssIntervalByAcpResponse{}, nil
}

func (k *Keeper) ScheduleRingUpgradeByAcp(goCtx context.Context, msg *types.MsgScheduleRingUpgradeByAcp) (*types.MsgScheduleRingUpgradeByAcpResponse, error) {
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

	// currentTime is needed for validateRingUpgradeSchedule below, so we call
	// normalizeMaturedRingUpgrade directly instead of maybeNormalizeMaturedRingUpgrade.
	currentTime, err := currentBlockUnixTime(ctx)
	if err != nil {
		return nil, err
	}
	upgradeNormalizedEvent := normalizeMaturedRingUpgrade(ring, currentTime)

	if ring.UpgradeInfo.XNextVersion != nil &&
		ring.UpgradeInfo.GetNextVersion() == msg.NextVersion &&
		ring.UpgradeInfo.GetActivationTime() == msg.ActivationTime {
		return nil, types.ErrRingUpgradeUnchanged
	}
	if err := validateRingUpgradeSchedule(msg.NextVersion, msg.ActivationTime, currentTime, ring); err != nil {
		return nil, err
	}

	setRingUpgrade(ring, msg.NextVersion, msg.ActivationTime)
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
	if upgradeNormalizedEvent != nil {
		if err := ctx.EventManager().EmitTypedEvent(upgradeNormalizedEvent); err != nil {
			return nil, err
		}
	}
	if err := ctx.EventManager().EmitTypedEvent(&types.EventRingUpgradeScheduled{
		RingId:         ring.Id,
		CurrentVersion: ring.UpgradeInfo.CurrentVersion,
		NextVersion:    msg.NextVersion,
		ActivationTime: msg.ActivationTime,
	}); err != nil {
		return nil, err
	}

	return &types.MsgScheduleRingUpgradeByAcpResponse{}, nil
}

func (k *Keeper) CancelRingUpgradeByAcp(goCtx context.Context, msg *types.MsgCancelRingUpgradeByAcp) (*types.MsgCancelRingUpgradeByAcpResponse, error) {
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

	if ring.UpgradeInfo.XNextVersion == nil || ring.UpgradeInfo.XActivationTime == nil {
		return nil, types.ErrRingUpgradeNotScheduled
	}
	currentTime, err := currentBlockUnixTime(ctx)
	if err != nil {
		return nil, err
	}
	if currentTime >= ring.UpgradeInfo.GetActivationTime() {
		return nil, types.ErrRingUpgradeAlreadyActive
	}

	clearRingUpgrade(ring)
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
	if err := ctx.EventManager().EmitTypedEvent(&types.EventRingUpgradeCancelled{
		RingId:         ring.Id,
		CurrentVersion: ring.UpgradeInfo.CurrentVersion,
	}); err != nil {
		return nil, err
	}

	return &types.MsgCancelRingUpgradeByAcpResponse{}, nil
}

func (k *Keeper) SetRingReportingByAcp(goCtx context.Context, msg *types.MsgSetRingReportingByAcp) (*types.MsgSetRingReportingByAcpResponse, error) {
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

	upgradeNormalizedEvent, err := maybeNormalizeMaturedRingUpgrade(ring, ctx)
	if err != nil {
		return nil, err
	}

	ring.Reporting = msg.Reporting
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
	if upgradeNormalizedEvent != nil {
		if err := ctx.EventManager().EmitTypedEvent(upgradeNormalizedEvent); err != nil {
			return nil, err
		}
	}

	return &types.MsgSetRingReportingByAcpResponse{}, nil
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

func (k *Keeper) SubmitReport(
	goCtx context.Context,
	msg *types.MsgSubmitReport,
) (*types.MsgSubmitReportResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	report := msg.GetReport()
	validatedReport, err := k.validateSubmittedReport(
		goCtx,
		ctx,
		&report,
		msg.ReportId,
		msg.SignatureScheme,
		msg.Signature,
	)
	if err != nil {
		return nil, err
	}

	demeritAmount, err := DemeritAmountForReportType(validatedReport.ring, report.ReportType)
	if err != nil {
		return nil, err
	}

	k.SetAcceptedReportPair(goCtx, validatedReport.reportID, validatedReport.sessionDedupeID, validatedReport.blockUnixTime+ReportTTLSeconds)
	effectiveDemerits := k.IncrementNodeDemerits(
		goCtx,
		report.RingId,
		report.AccusedNodeKey,
		demeritAmount,
		validatedReport.blockUnixTime,
		validatedReport.ring.Reporting.DemeritConfig.ResetIntervalSeconds,
	)
	if err := k.maybeScheduleAutoReshareForReport(
		goCtx,
		ctx,
		validatedReport.ring,
		report.AccusedNodeKey,
		effectiveDemerits,
	); err != nil {
		return nil, err
	}

	if err := ctx.EventManager().EmitTypedEvent(&types.EventReportAccepted{
		ReportId:        validatedReport.reportID,
		RingId:          report.RingId,
		ReportType:      report.ReportType,
		ReporterNodeKey: report.ReporterNodeKey,
		AccusedNodeKey:  report.AccusedNodeKey,
	}); err != nil {
		return nil, err
	}

	return &types.MsgSubmitReportResponse{ReportId: validatedReport.reportID}, nil
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
	if err := normalizeAndValidateNodeInfo(&nodeInfo); err != nil {
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

func (k *Keeper) UpdateNodePeerId(goCtx context.Context, msg *types.MsgUpdateNodePeerId) (*types.MsgUpdateNodePeerIdResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	nodeInfo, err := k.authorizeNodeInfoUpdate(goCtx, ctx, msg.NodeKey, msg.Creator)
	if err != nil {
		return nil, err
	}

	if msg.PeerId == "" {
		return nil, errorsmod.Wrap(types.ErrInvalidNodeInfo, "missing peer_id")
	}
	if nodeInfo.PeerId == msg.PeerId {
		return nil, types.ErrNodeInfoUnchanged
	}

	nodeInfo.PeerId = msg.PeerId
	if err := normalizeAndValidateNodeInfo(nodeInfo); err != nil {
		return nil, err
	}
	k.SetNodeInfo(goCtx, msg.NodeKey, *nodeInfo)
	if err := ctx.EventManager().EmitTypedEvent(&types.EventNodeInfoUpdated{
		PeerId:        nodeInfo.PeerId,
		ControllerKey: nodeInfo.ControllerKey,
	}); err != nil {
		return nil, err
	}

	return &types.MsgUpdateNodePeerIdResponse{}, nil
}

func (k *Keeper) TransferNodeController(goCtx context.Context, msg *types.MsgTransferNodeController) (*types.MsgTransferNodeControllerResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	nodeInfo, err := k.authorizeNodeInfoUpdate(goCtx, ctx, msg.NodeKey, msg.Creator)
	if err != nil {
		return nil, err
	}

	previousControllerKey := nodeInfo.ControllerKey
	nodeInfo.ControllerKey = msg.ControllerKey
	// normalizeAndValidateNodeInfo normalizes ControllerKey (strips 0x, lowercases),
	// so the equality check below correctly identifies e.g. "0xABCD" as unchanged
	// against the stored "abcd". Validation must run before the comparison.
	if err := normalizeAndValidateNodeInfo(nodeInfo); err != nil {
		return nil, err
	}
	if nodeInfo.ControllerKey == previousControllerKey {
		return nil, types.ErrNodeInfoUnchanged
	}

	k.SetNodeInfo(goCtx, msg.NodeKey, *nodeInfo)
	if err := ctx.EventManager().EmitTypedEvent(&types.EventNodeInfoUpdated{
		PeerId:        nodeInfo.PeerId,
		ControllerKey: nodeInfo.ControllerKey,
	}); err != nil {
		return nil, err
	}

	return &types.MsgTransferNodeControllerResponse{}, nil
}

func (k *Keeper) AddNodeToWhitelist(goCtx context.Context, msg *types.MsgAddNodeToWhitelist) (*types.MsgAddNodeToWhitelistResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	nodeInfo, err := k.authorizeNodeInfoUpdate(goCtx, ctx, msg.NodeKey, msg.Creator)
	if err != nil {
		return nil, err
	}

	switch target := msg.Target.(type) {
	case *types.MsgAddNodeToWhitelist_PolicyId:
		if target.PolicyId == "" {
			return nil, errorsmod.Wrap(types.ErrInvalidNodeInfo, "missing policy_id")
		}
		if slices.Contains(nodeInfo.WhitelistedPolicyIds, target.PolicyId) {
			return nil, types.ErrNodeWhitelistEntryExists
		}
		nodeInfo.WhitelistedPolicyIds = append(nodeInfo.WhitelistedPolicyIds, target.PolicyId)
	case *types.MsgAddNodeToWhitelist_RingId:
		if target.RingId == "" {
			return nil, errorsmod.Wrap(types.ErrInvalidNodeInfo, "missing ring_id")
		}
		if slices.Contains(nodeInfo.WhitelistedRingIds, target.RingId) {
			return nil, types.ErrNodeWhitelistEntryExists
		}
		nodeInfo.WhitelistedRingIds = append(nodeInfo.WhitelistedRingIds, target.RingId)
	default:
		return nil, errorsmod.Wrap(types.ErrInvalidNodeInfo, "missing whitelist target")
	}

	if err := normalizeAndValidateNodeInfo(nodeInfo); err != nil {
		return nil, err
	}
	k.SetNodeInfo(goCtx, msg.NodeKey, *nodeInfo)
	if err := ctx.EventManager().EmitTypedEvent(&types.EventNodeInfoUpdated{
		PeerId:        nodeInfo.PeerId,
		ControllerKey: nodeInfo.ControllerKey,
	}); err != nil {
		return nil, err
	}

	return &types.MsgAddNodeToWhitelistResponse{}, nil
}

func (k *Keeper) RemoveNodeFromWhitelist(goCtx context.Context, msg *types.MsgRemoveNodeFromWhitelist) (*types.MsgRemoveNodeFromWhitelistResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	nodeInfo, err := k.authorizeNodeInfoUpdate(goCtx, ctx, msg.NodeKey, msg.Creator)
	if err != nil {
		return nil, err
	}

	switch target := msg.Target.(type) {
	case *types.MsgRemoveNodeFromWhitelist_PolicyId:
		if target.PolicyId == "" {
			return nil, errorsmod.Wrap(types.ErrInvalidNodeInfo, "missing policy_id")
		}
		index := slices.Index(nodeInfo.WhitelistedPolicyIds, target.PolicyId)
		if index < 0 {
			return nil, types.ErrNodeWhitelistEntryNotFound
		}
		nodeInfo.WhitelistedPolicyIds = slices.Delete(nodeInfo.WhitelistedPolicyIds, index, index+1)
	case *types.MsgRemoveNodeFromWhitelist_RingId:
		if target.RingId == "" {
			return nil, errorsmod.Wrap(types.ErrInvalidNodeInfo, "missing ring_id")
		}
		index := slices.Index(nodeInfo.WhitelistedRingIds, target.RingId)
		if index < 0 {
			return nil, types.ErrNodeWhitelistEntryNotFound
		}
		nodeInfo.WhitelistedRingIds = slices.Delete(nodeInfo.WhitelistedRingIds, index, index+1)
	default:
		return nil, errorsmod.Wrap(types.ErrInvalidNodeInfo, "missing whitelist target")
	}

	if err := normalizeAndValidateNodeInfo(nodeInfo); err != nil {
		return nil, err
	}
	k.SetNodeInfo(goCtx, msg.NodeKey, *nodeInfo)
	if err := ctx.EventManager().EmitTypedEvent(&types.EventNodeInfoUpdated{
		PeerId:        nodeInfo.PeerId,
		ControllerKey: nodeInfo.ControllerKey,
	}); err != nil {
		return nil, err
	}

	return &types.MsgRemoveNodeFromWhitelistResponse{}, nil
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
