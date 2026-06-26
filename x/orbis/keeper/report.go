package keeper

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"slices"
	"unicode/utf8"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/sourcenetwork/sourcehub/x/orbis/types"
)

const (
	ReportDomain          = "orbis-mpc-fault-report"
	NodeOfflineReportType = "node_offline"
	ReportTTLSeconds      = uint64(120)

	offlineOriginProtocolPRE        = "pre"
	offlineOriginProtocolSign       = "sign"
	offlineOriginProtocolPSSRefresh = "pss_refresh"
	offlineOriginProtocolPSSReshare = "pss_reshare"

	committeeScopeCurrent    = byte(1)
	committeeScopePendingNew = byte(2)
)

type nodeOfflineReportPayload struct {
	originProtocol        string
	originProtocolVersion uint64
	accusedCommitteeScope byte
	signingCommitteeScope byte
}

type reportCommitteeView struct {
	peerNodeKeys []string
	threshold    uint32
}

type validatedSubmittedReport struct {
	reportID      string
	ring          *types.Ring
	blockUnixTime uint64
}

func (k *Keeper) validateSubmittedReport(
	goCtx context.Context,
	ctx sdk.Context,
	report *types.ReportEnvelope,
	claimedReportID string,
	signatureScheme string,
	signature []byte,
) (*validatedSubmittedReport, error) {
	if report == nil {
		return nil, errorsmod.Wrap(types.ErrInvalidReport, "missing report")
	}

	now, err := reportBlockUnixTime(ctx)
	if err != nil {
		return nil, err
	}
	if err := validateReportEnvelopeShape(report, now); err != nil {
		return nil, err
	}
	if report.ChainId != ctx.ChainID() {
		return nil, errorsmod.Wrap(types.ErrInvalidReport, "report chain ID does not match current chain")
	}

	message, reportID, err := reportEnvelopeCanonicalMessageAndID(report)
	if err != nil {
		return nil, err
	}
	if claimedReportID != reportID {
		return nil, errorsmod.Wrap(types.ErrInvalidReport, "report_id does not match canonical report bytes")
	}
	if k.HasAcceptedReport(goCtx, reportID) {
		return nil, types.ErrReportAlreadyAccepted
	}

	var payload nodeOfflineReportPayload
	switch report.ReportType {
	case NodeOfflineReportType:
		payload, err = decodeNodeOfflinePayload(report.Payload)
		if err != nil {
			return nil, err
		}
		if !isValidOfflineOriginProtocol(payload.originProtocol) {
			return nil, errorsmod.Wrapf(types.ErrInvalidReport, "unsupported offline report origin protocol %q", payload.originProtocol)
		}
	default:
		return nil, errorsmod.Wrapf(types.ErrInvalidReport, "unsupported report type %q", report.ReportType)
	}

	ring := k.GetRing(goCtx, report.RingId)
	if ring == nil {
		return nil, types.ErrRingNotFound
	}
	if err := requireRingFinalized(ring); err != nil {
		return nil, err
	}
	if report.RingPk != ring.RingPk {
		return nil, errorsmod.Wrap(types.ErrInvalidReport, "report ring public key is stale")
	}
	ringStateHash, err := reportRingStateSHA256(ring)
	if err != nil {
		return nil, err
	}
	if report.RingStateSha256 != ringStateHash {
		return nil, errorsmod.Wrap(types.ErrInvalidReport, "report ring-state digest is stale")
	}

	effectiveVersion := effectiveReportProtocolVersion(ring, now)
	if payload.originProtocolVersion != effectiveVersion {
		return nil, errorsmod.Wrapf(
			types.ErrInvalidReport,
			"report origin protocol version %d is not effective for ring %s",
			payload.originProtocolVersion,
			report.RingId,
		)
	}

	if err := validateReportCommitteeAuthorization(goCtx, k, report, payload, ring); err != nil {
		return nil, err
	}

	if err := verifyThresholdSignature(signatureScheme, ring.RingPk, message, signature); err != nil {
		return nil, err
	}

	return &validatedSubmittedReport{
		reportID:      reportID,
		ring:          ring,
		blockUnixTime: now,
	}, nil
}

func validateReportEnvelopeShape(report *types.ReportEnvelope, now uint64) error {
	switch {
	case report.Domain != ReportDomain:
		return errorsmod.Wrapf(types.ErrInvalidReport, "unexpected report domain %q", report.Domain)
	case report.ObservedAt > report.ExpiresAt:
		return errorsmod.Wrap(types.ErrInvalidReport, "report observed_at is after expires_at")
	case report.ExpiresAt-report.ObservedAt != ReportTTLSeconds:
		return errorsmod.Wrap(types.ErrInvalidReport, "invalid report validity window")
	case now > report.ExpiresAt:
		return errorsmod.Wrap(types.ErrInvalidReport, "report has expired")
	case report.ReporterNodeKey == report.AccusedNodeKey:
		return errorsmod.Wrap(types.ErrInvalidReport, "reporter and accused node must differ")
	case len(report.Payload) == 0:
		return errorsmod.Wrap(types.ErrInvalidReport, "missing report payload")
	}

	for label, value := range map[string]string{
		"report_type":       report.ReportType,
		"chain_id":          report.ChainId,
		"ring_id":           report.RingId,
		"ring_pk":           report.RingPk,
		"ring_state_sha256": report.RingStateSha256,
		"reporter_node_key": report.ReporterNodeKey,
		"accused_node_key":  report.AccusedNodeKey,
		"accused_peer_id":   report.AccusedPeerId,
	} {
		if value == "" {
			return errorsmod.Wrapf(types.ErrInvalidReport, "%s cannot be empty", label)
		}
	}
	if _, err := hex.DecodeString(report.RingStateSha256); err != nil || len(report.RingStateSha256) != sha256.Size*2 {
		return errorsmod.Wrap(types.ErrInvalidReport, "ring_state_sha256 must be 32-byte hex")
	}

	return nil
}

func reportEnvelopeCanonicalMessageAndID(report *types.ReportEnvelope) ([]byte, string, error) {
	message, err := reportEnvelopeCanonicalBytes(report)
	if err != nil {
		return nil, "", err
	}
	hash := sha256.Sum256(message)
	return message, hex.EncodeToString(hash[:]), nil
}

func reportEnvelopeCanonicalBytes(report *types.ReportEnvelope) ([]byte, error) {
	w := newReportCanonicalWriter()
	w.writeString(report.Domain)
	w.writeString(report.ReportType)
	w.writeString(report.ChainId)
	w.writeString(report.RingId)
	w.writeString(report.RingPk)
	w.writeString(report.RingStateSha256)
	w.writeString(report.ReporterNodeKey)
	w.writeString(report.AccusedNodeKey)
	w.writeString(report.AccusedPeerId)
	w.writeU64(report.ObservedAt)
	w.writeU64(report.ExpiresAt)
	w.writeBytes(report.Payload)
	return w.finish()
}

func decodeNodeOfflinePayload(payload []byte) (nodeOfflineReportPayload, error) {
	decoder := newReportCanonicalDecoder(payload)
	originProtocol, err := decoder.readString("origin_protocol")
	if err != nil {
		return nodeOfflineReportPayload{}, err
	}
	originProtocolVersion, err := decoder.readU64("origin_protocol_version")
	if err != nil {
		return nodeOfflineReportPayload{}, err
	}

	accusedScope, err := decoder.readByte("accused_committee_scope")
	if err != nil {
		return nodeOfflineReportPayload{}, err
	}
	if !isValidReportCommitteeScope(accusedScope) {
		return nodeOfflineReportPayload{}, errorsmod.Wrapf(types.ErrInvalidReport, "unknown accused committee scope %d", accusedScope)
	}
	signingScope, err := decoder.readByte("signing_committee_scope")
	if err != nil {
		return nodeOfflineReportPayload{}, err
	}
	if !isValidReportCommitteeScope(signingScope) {
		return nodeOfflineReportPayload{}, errorsmod.Wrapf(types.ErrInvalidReport, "unknown signing committee scope %d", signingScope)
	}
	if err := decoder.finish(); err != nil {
		return nodeOfflineReportPayload{}, err
	}

	return nodeOfflineReportPayload{
		originProtocol:        originProtocol,
		originProtocolVersion: originProtocolVersion,
		accusedCommitteeScope: accusedScope,
		signingCommitteeScope: signingScope,
	}, nil
}

func isValidReportCommitteeScope(scope byte) bool {
	return scope == committeeScopeCurrent || scope == committeeScopePendingNew
}

func isValidOfflineOriginProtocol(originProtocol string) bool {
	switch originProtocol {
	case offlineOriginProtocolPRE,
		offlineOriginProtocolSign,
		offlineOriginProtocolPSSRefresh,
		offlineOriginProtocolPSSReshare:
		return true
	default:
		return false
	}
}

func validateReportCommitteeAuthorization(
	goCtx context.Context,
	k *Keeper,
	report *types.ReportEnvelope,
	payload nodeOfflineReportPayload,
	ring *types.Ring,
) error {
	accusedCommittee, err := reportCommitteeForScope(ring, payload.accusedCommitteeScope)
	if err != nil {
		return err
	}
	signingCommittee, err := reportCommitteeForScope(ring, payload.signingCommitteeScope)
	if err != nil {
		return err
	}

	if signingCommittee.threshold < 2 {
		return errorsmod.Wrap(types.ErrInvalidReport, "offline reporting requires ring threshold >= 2")
	}
	if int(signingCommittee.threshold) > len(signingCommittee.peerNodeKeys) {
		return errorsmod.Wrap(types.ErrInvalidReport, "offline reporting threshold exceeds signing committee size")
	}
	if slices.Contains(signingCommittee.peerNodeKeys, report.AccusedNodeKey) &&
		int(signingCommittee.threshold) > len(signingCommittee.peerNodeKeys)-1 {
		return errorsmod.Wrap(types.ErrInvalidReport, "ring threshold cannot be met while excluding the accused node")
	}
	if !slices.Contains(signingCommittee.peerNodeKeys, report.ReporterNodeKey) {
		return errorsmod.Wrapf(types.ErrInvalidReport, "reporter node %s is not in the signing committee", report.ReporterNodeKey)
	}
	if !slices.Contains(accusedCommittee.peerNodeKeys, report.AccusedNodeKey) {
		return errorsmod.Wrapf(types.ErrInvalidReport, "accused node %s is not in the accused committee", report.AccusedNodeKey)
	}

	accusedInfo := k.GetNodeInfo(goCtx, report.AccusedNodeKey)
	if accusedInfo == nil {
		return types.ErrNodeInfoNotFound
	}
	if accusedInfo.PeerId != report.AccusedPeerId {
		return errorsmod.Wrap(types.ErrInvalidReport, "accused peer ID no longer matches NodeInfo")
	}

	return nil
}

func reportCommitteeForScope(ring *types.Ring, scope byte) (reportCommitteeView, error) {
	switch scope {
	case committeeScopeCurrent:
		return reportCommitteeView{
			peerNodeKeys: slices.Clone(ring.PeerNodeKeys),
			threshold:    ring.Threshold,
		}, nil
	case committeeScopePendingNew:
		if len(ring.NewPeerNodeKeys) == 0 && ring.XNewThreshold == nil {
			return reportCommitteeView{}, errorsmod.Wrap(types.ErrInvalidReport, "pending-new committee scope requires a pending reshare")
		}
		peerNodeKeys := ring.NewPeerNodeKeys
		if len(peerNodeKeys) == 0 {
			peerNodeKeys = ring.PeerNodeKeys
		}
		threshold := ring.Threshold
		if ring.XNewThreshold != nil {
			threshold = ring.GetNewThreshold()
		}
		return reportCommitteeView{
			peerNodeKeys: slices.Clone(peerNodeKeys),
			threshold:    threshold,
		}, nil
	default:
		return reportCommitteeView{}, errorsmod.Wrapf(types.ErrInvalidReport, "unknown committee scope %d", scope)
	}
}

func effectiveReportProtocolVersion(ring *types.Ring, now uint64) uint64 {
	if ring.UpgradeInfo.XNextVersion != nil &&
		ring.UpgradeInfo.XActivationTime != nil &&
		now >= ring.UpgradeInfo.GetActivationTime() {
		return ring.UpgradeInfo.GetNextVersion()
	}
	return ring.UpgradeInfo.CurrentVersion
}

func reportBlockUnixTime(ctx sdk.Context) (uint64, error) {
	unixTime := ctx.BlockTime().Unix()
	if unixTime < 0 {
		return 0, errorsmod.Wrap(types.ErrInvalidReport, "current block time is before the Unix epoch")
	}
	return uint64(unixTime), nil
}

func reportRingStateSHA256(ring *types.Ring) (string, error) {
	canonical, err := reportRingStateCanonicalBytes(ring)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(canonical)
	return hex.EncodeToString(hash[:]), nil
}

func reportRingStateCanonicalBytes(ring *types.Ring) ([]byte, error) {
	w := newReportCanonicalWriter()
	w.writeString(ring.RingPk)
	w.writeStringSlice(ring.PeerNodeKeys)
	w.writeU32(ring.Threshold)

	if len(ring.NewPeerNodeKeys) == 0 {
		w.writeOptionalStringSlice(nil)
	} else {
		w.writeOptionalStringSlice(ring.NewPeerNodeKeys)
	}
	if ring.XNewThreshold == nil {
		w.writeOptionalU32(nil)
	} else {
		threshold := ring.GetNewThreshold()
		w.writeOptionalU32(&threshold)
	}
	w.writeU64(ring.PssInterval)
	w.writeU64(ring.BlockNumberNonce)
	if ring.PolicyId == "" {
		w.writeOptionalString(nil)
	} else {
		w.writeOptionalString(&ring.PolicyId)
	}
	w.writeU64(ring.UpgradeInfo.CurrentVersion)
	if ring.UpgradeInfo.XNextVersion == nil {
		w.writeOptionalU64(nil)
	} else {
		nextVersion := ring.UpgradeInfo.GetNextVersion()
		w.writeOptionalU64(&nextVersion)
	}
	if ring.UpgradeInfo.XActivationTime == nil {
		w.writeOptionalU64(nil)
	} else {
		activationTime := ring.UpgradeInfo.GetActivationTime()
		w.writeOptionalU64(&activationTime)
	}
	return w.finish()
}

type reportCanonicalWriter struct {
	bytes []byte
	err   error
}

func newReportCanonicalWriter() *reportCanonicalWriter {
	return &reportCanonicalWriter{}
}

func (w *reportCanonicalWriter) writeU32(value uint32) {
	var buf [4]byte
	binary.BigEndian.PutUint32(buf[:], value)
	w.bytes = append(w.bytes, buf[:]...)
}

func (w *reportCanonicalWriter) writeU64(value uint64) {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], value)
	w.bytes = append(w.bytes, buf[:]...)
}

func (w *reportCanonicalWriter) writeBytes(value []byte) {
	if len(value) > math.MaxUint32 {
		w.setErr(fmt.Errorf("byte slice exceeds u32 length"))
		return
	}
	w.writeU32(uint32(len(value)))
	w.bytes = append(w.bytes, value...)
}

func (w *reportCanonicalWriter) writeString(value string) {
	w.writeBytes([]byte(value))
}

func (w *reportCanonicalWriter) writeStringSlice(values []string) {
	if len(values) > math.MaxUint32 {
		w.setErr(fmt.Errorf("string slice exceeds u32 length"))
		return
	}
	w.writeU32(uint32(len(values)))
	for _, value := range values {
		w.writeString(value)
	}
}

func (w *reportCanonicalWriter) writeOptionalString(value *string) {
	if value == nil {
		w.bytes = append(w.bytes, 0)
		return
	}
	w.bytes = append(w.bytes, 1)
	w.writeString(*value)
}

func (w *reportCanonicalWriter) writeOptionalStringSlice(value []string) {
	if value == nil {
		w.bytes = append(w.bytes, 0)
		return
	}
	w.bytes = append(w.bytes, 1)
	w.writeStringSlice(value)
}

func (w *reportCanonicalWriter) writeOptionalU32(value *uint32) {
	if value == nil {
		w.bytes = append(w.bytes, 0)
		return
	}
	w.bytes = append(w.bytes, 1)
	w.writeU32(*value)
}

func (w *reportCanonicalWriter) writeOptionalU64(value *uint64) {
	if value == nil {
		w.bytes = append(w.bytes, 0)
		return
	}
	w.bytes = append(w.bytes, 1)
	w.writeU64(*value)
}

func (w *reportCanonicalWriter) finish() ([]byte, error) {
	if w.err != nil {
		return nil, errorsmod.Wrap(types.ErrInvalidReport, w.err.Error())
	}
	return w.bytes, nil
}

func (w *reportCanonicalWriter) setErr(err error) {
	if w.err == nil {
		w.err = err
	}
}

type reportCanonicalDecoder struct {
	bytes  []byte
	cursor int
}

func newReportCanonicalDecoder(bytes []byte) *reportCanonicalDecoder {
	return &reportCanonicalDecoder{bytes: bytes}
}

func (d *reportCanonicalDecoder) readByte(label string) (byte, error) {
	if len(d.bytes)-d.cursor < 1 {
		return 0, errorsmod.Wrapf(types.ErrInvalidReport, "missing %s", label)
	}
	value := d.bytes[d.cursor]
	d.cursor++
	return value, nil
}

func (d *reportCanonicalDecoder) readU32(label string) (uint32, error) {
	if len(d.bytes)-d.cursor < 4 {
		return 0, errorsmod.Wrapf(types.ErrInvalidReport, "missing %s", label)
	}
	value := binary.BigEndian.Uint32(d.bytes[d.cursor : d.cursor+4])
	d.cursor += 4
	return value, nil
}

func (d *reportCanonicalDecoder) readU64(label string) (uint64, error) {
	if len(d.bytes)-d.cursor < 8 {
		return 0, errorsmod.Wrapf(types.ErrInvalidReport, "missing %s", label)
	}
	value := binary.BigEndian.Uint64(d.bytes[d.cursor : d.cursor+8])
	d.cursor += 8
	return value, nil
}

func (d *reportCanonicalDecoder) readString(label string) (string, error) {
	length, err := d.readU32(label + "_length")
	if err != nil {
		return "", err
	}
	if uint64(len(d.bytes)-d.cursor) < uint64(length) {
		return "", errorsmod.Wrapf(types.ErrInvalidReport, "truncated %s", label)
	}
	valueBytes := d.bytes[d.cursor : d.cursor+int(length)]
	d.cursor += int(length)
	if !utf8.Valid(valueBytes) {
		return "", errorsmod.Wrapf(types.ErrInvalidReport, "%s is not utf-8", label)
	}
	return string(valueBytes), nil
}

func (d *reportCanonicalDecoder) finish() error {
	if d.cursor != len(d.bytes) {
		return errorsmod.Wrap(types.ErrInvalidReport, "trailing payload bytes")
	}
	return nil
}
