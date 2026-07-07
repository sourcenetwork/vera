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
	ReportDomain                    = "orbis-mpc-fault-report"
	ReportSessionDomain             = "orbis-mpc-fault-report-session"
	NodeOfflineReportType           = "node_offline"
	InvalidCryptoResponseReportType = "invalid_crypto_response"
	ReportTTLSeconds                = uint64(120)

	// PreReencryptResponseDomain is the domain tag of the responder-signed PRE
	// response statement carried as invalid_crypto_response/pre evidence.
	PreReencryptResponseDomain = "orbis-pre-reencrypt-response-v1"
	// SignResponseDomain is the domain tag of the responder-signed Sign response
	// statement carried as invalid_crypto_response/sign evidence.
	SignResponseDomain = "orbis-sign-response-v1"
	// DkgCommitmentDomain is the domain tag of the responder-signed raw DKG
	// commitment statement nested inside invalid_crypto_response/dkg_share evidence.
	DkgCommitmentDomain = "orbis-dkg-commitment-v1"
	// DkgShareDomain is the domain tag of the responder-signed raw DKG share
	// statement carried as invalid_crypto_response/dkg_share evidence.
	DkgShareDomain = "orbis-dkg-share-v1"
	// reportObservedAtGraceSecs mirrors orbis-rs CHAIN_BLOCK_GRACE_SECS:
	// reporters backdate observed_at by this so the observed_at <= block_time
	// check passes gas simulation. invalid_crypto_response envelopes are pinned
	// to their evidence via observed_at == signed_at - grace, which makes
	// the envelope's fixed observed_at + ReportTTLSeconds expiry double as the
	// evidence expiry: acceptance can only happen in
	// [signed_at - grace, signed_at - grace + TTL], so a plain TTL dedupe
	// record always outlives any resubmission of the same evidence.
	reportObservedAtGraceSecs = uint64(10)

	// Size bounds for evidence blobs (group elements are 32-112 bytes across
	// crypto backends; derivation is capability bytes).
	preResponseMaxElementLen      = 512
	preResponseMaxDerivationLen   = 4096
	signResponseMaxElementLen     = 4096
	signResponseMaxMessageLen     = 1024 * 1024
	signResponseMaxCommitmentsLen = 1024 * 1024
	dkgStatementMaxLen            = 1024 * 1024
	dkgCommitmentMaxLen           = 1024 * 1024
	dkgShareMaxElementLen         = 4096
	dkgNonceLen                   = 16

	offlineOriginProtocolPRE        = "pre"
	offlineOriginProtocolSign       = "sign"
	offlineOriginProtocolPSSRefresh = "pss_refresh"
	offlineOriginProtocolPSSReshare = "pss_reshare"
	originProtocolReport            = "report"

	invalidCryptoEvidenceKindPRE      = "pre"
	invalidCryptoEvidenceKindSign     = "sign"
	invalidCryptoEvidenceKindDkgShare = "dkg_share"

	committeeScopeCurrent    = byte(1)
	committeeScopePendingNew = byte(2)
)

// reportPayload is normalized per-report metadata consumed by the version
// check, committee authorization, and session-dedupe ID. node_offline decodes
// it directly from its payload; other report types synthesize it from their own
// payloads.
type reportPayload struct {
	originProtocol        string
	originProtocolVersion uint64
	accusedCommitteeScope byte
	signingCommitteeScope byte
}

// invalidCryptoResponseStatement holds the common fields of responder-signed
// PRE, Sign, and raw DKG share response statements that the chain validates.
// Protocol-specific cryptographic blobs are bounds-checked during decoding and
// then dropped: the chain does not re-verify the PRE proof, Sign share, DKG
// share, or responder signature; its crypto gate remains the ring threshold
// signature on the report envelope.
type invalidCryptoResponseStatement struct {
	chainID               string
	ringID                string
	ringPk                string
	ringStateSha256       string
	protocolVersion       uint64
	requestID             string
	signedAt              uint64
	responderNodeKey      string
	originProtocol        string
	accusedCommitteeScope byte
	signingCommitteeScope byte
}

type dkgCommitmentStatement struct {
	chainID               string
	ringID                string
	ringPk                string
	ringStateSha256       string
	protocolVersion       uint64
	requestID             string
	signedAt              uint64
	responderNodeKey      string
	originProtocol        string
	accusedCommitteeScope byte
	signingCommitteeScope byte
	fromNodeID            uint32
	commitment            []byte
	cryptoBackend         string
}

type reportCommitteeView struct {
	peerNodeKeys []string
	threshold    uint32
}

type validatedSubmittedReport struct {
	reportID        string
	sessionDedupeID string
	ring            *types.Ring
	blockUnixTime   uint64
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

	var payload reportPayload
	switch report.ReportType {
	case NodeOfflineReportType:
		payload, err = decodeNodeOfflinePayload(report.Payload)
		if err != nil {
			return nil, err
		}
		if !isValidOfflineOriginProtocol(payload.originProtocol) {
			return nil, errorsmod.Wrapf(types.ErrInvalidReport, "unsupported offline report origin protocol %q", payload.originProtocol)
		}
	case InvalidCryptoResponseReportType:
		statement, err := decodeInvalidCryptoResponsePayload(report.Payload)
		if err != nil {
			return nil, err
		}
		if err := validateInvalidCryptoResponseStatement(report, statement); err != nil {
			return nil, err
		}
		payload = reportPayload{
			originProtocol:        statement.originProtocol,
			originProtocolVersion: statement.protocolVersion,
			accusedCommitteeScope: statement.accusedCommitteeScope,
			signingCommitteeScope: statement.signingCommitteeScope,
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

	effectiveVersion := effectiveReportProtocolVersion(ring, report.ObservedAt)
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

	sessionDedupeID, err := reportSessionDedupeID(report, payload)
	if err != nil {
		return nil, err
	}
	if k.HasAcceptedReportSession(goCtx, sessionDedupeID) {
		return nil, types.ErrReportAlreadyAccepted
	}

	if err := verifyThresholdSignature(signatureScheme, ring.RingPk, message, signature); err != nil {
		return nil, err
	}

	return &validatedSubmittedReport{
		reportID:        reportID,
		sessionDedupeID: sessionDedupeID,
		ring:            ring,
		blockUnixTime:   now,
	}, nil
}

func validateReportEnvelopeShape(report *types.ReportEnvelope, now uint64) error {
	switch {
	case report.Domain != ReportDomain:
		return errorsmod.Wrapf(types.ErrInvalidReport, "unexpected report domain %q", report.Domain)
	case report.ObservedAt > report.ExpiresAt:
		return errorsmod.Wrap(types.ErrInvalidReport, "report observed_at is after expires_at")
	case report.ObservedAt > now:
		return errorsmod.Wrap(types.ErrInvalidReport, "report observed_at is in the future")
	case report.ExpiresAt-report.ObservedAt != ReportTTLSeconds:
		return errorsmod.Wrap(types.ErrInvalidReport, "invalid report validity window")
	case now > report.ExpiresAt:
		return errorsmod.Wrap(types.ErrInvalidReport, "report has expired")
	case report.ReporterNodeKey == report.AccusedNodeKey:
		return errorsmod.Wrap(types.ErrInvalidReport, "reporter and accused node must differ")
	case len(report.Payload) == 0:
		return errorsmod.Wrap(types.ErrInvalidReport, "missing report payload")
	case report.SessionId == "":
		return errorsmod.Wrap(types.ErrInvalidReport, "session_id cannot be empty")
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

func reportSessionDedupeID(report *types.ReportEnvelope, payload reportPayload) (string, error) {
	w := newReportCanonicalWriter()
	w.writeString(ReportSessionDomain)
	w.writeString(report.ChainId)
	w.writeString(report.RingId)
	w.writeString(report.ReportType)
	w.writeString(payload.originProtocol)
	w.writeString(report.AccusedNodeKey)
	w.writeString(report.SessionId)
	message, err := w.finish()
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(message)
	return hex.EncodeToString(hash[:]), nil
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
	w.writeString(report.SessionId)
	return w.finish()
}

func decodeNodeOfflinePayload(payload []byte) (reportPayload, error) {
	decoder := newReportCanonicalDecoder(payload)
	originProtocol, err := decoder.readString("origin_protocol")
	if err != nil {
		return reportPayload{}, err
	}
	originProtocolVersion, err := decoder.readU64("origin_protocol_version")
	if err != nil {
		return reportPayload{}, err
	}

	accusedScope, err := decoder.readByte("accused_committee_scope")
	if err != nil {
		return reportPayload{}, err
	}
	if !isValidReportCommitteeScope(accusedScope) {
		return reportPayload{}, errorsmod.Wrapf(types.ErrInvalidReport, "unknown accused committee scope %d", accusedScope)
	}
	signingScope, err := decoder.readByte("signing_committee_scope")
	if err != nil {
		return reportPayload{}, err
	}
	if !isValidReportCommitteeScope(signingScope) {
		return reportPayload{}, errorsmod.Wrapf(types.ErrInvalidReport, "unknown signing committee scope %d", signingScope)
	}
	if err := decoder.finish(); err != nil {
		return reportPayload{}, err
	}

	return reportPayload{
		originProtocol:        originProtocol,
		originProtocolVersion: originProtocolVersion,
		accusedCommitteeScope: accusedScope,
		signingCommitteeScope: signingScope,
	}, nil
}

// decodeInvalidCryptoResponsePayload decodes and shape-validates an
// invalid_crypto_response payload: a tagged responder-signed PRE, Sign, or DKG
// share response statement plus a 64-byte secp256k1 signature. The field order
// matches the orbis-rs canonical encoding in reporting/v0/types.rs exactly.
func decodeInvalidCryptoResponsePayload(payload []byte) (invalidCryptoResponseStatement, error) {
	outer := newReportCanonicalDecoder(payload)
	evidenceKind, err := outer.readString("evidence_kind")
	if err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	statementBytes, err := outer.readBytes("statement")
	if err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	responseSignature, err := outer.readBytes("response_signature")
	if err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	if err := outer.finish(); err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	if len(responseSignature) != 64 {
		return invalidCryptoResponseStatement{}, errorsmod.Wrap(types.ErrInvalidReport, "response signature must be 64 bytes")
	}

	switch evidenceKind {
	case invalidCryptoEvidenceKindPRE:
		return decodePreReencryptResponseStatement(statementBytes)
	case invalidCryptoEvidenceKindSign:
		return decodeSignResponseStatement(statementBytes)
	case invalidCryptoEvidenceKindDkgShare:
		return decodeDkgShareStatement(statementBytes)
	default:
		return invalidCryptoResponseStatement{}, errorsmod.Wrapf(types.ErrInvalidReport, "unsupported invalid crypto evidence kind %q", evidenceKind)
	}
}

func decodePreReencryptResponseStatement(statementBytes []byte) (invalidCryptoResponseStatement, error) {
	decoder := newReportCanonicalDecoder(statementBytes)
	domain, err := decoder.readString("domain")
	if err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	if domain != PreReencryptResponseDomain {
		return invalidCryptoResponseStatement{}, errorsmod.Wrapf(types.ErrInvalidReport, "unexpected PRE response domain %q", domain)
	}
	chainID, err := decoder.readString("chain_id")
	if err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	ringID, err := decoder.readString("ring_id")
	if err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	ringPk, err := decoder.readString("ring_pk")
	if err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	ringStateSha256, err := decoder.readString("ring_state_sha256")
	if err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	protocolVersion, err := decoder.readU64("protocol_version")
	if err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	requestID, err := decoder.readString("request_id")
	if err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	signedAt, err := decoder.readU64("signed_at")
	if err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	responderNodeKey, err := decoder.readString("responder_node_key")
	if err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	objectID, err := decoder.readString("object_id")
	if err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	if requestID == "" || responderNodeKey == "" || objectID == "" {
		return invalidCryptoResponseStatement{}, errorsmod.Wrap(types.ErrInvalidReport, "PRE response statement has empty identity fields")
	}
	rdrPk, err := decoder.readBytes("rdr_pk")
	if err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	derivation, err := decoder.readOptionalBytes("derivation")
	if err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	if len(derivation) > preResponseMaxDerivationLen {
		return invalidCryptoResponseStatement{}, errorsmod.Wrap(types.ErrInvalidReport, "PRE response derivation exceeds size bound")
	}
	if _, err := decoder.readU32("from_node_id"); err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	share, err := decoder.readBytes("share")
	if err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	challenge, err := decoder.readBytes("challenge")
	if err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	proof, err := decoder.readBytes("proof")
	if err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	cryptoBackend, err := decoder.readString("crypto_backend")
	if err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	if cryptoBackend == "" {
		return invalidCryptoResponseStatement{}, errorsmod.Wrap(types.ErrInvalidReport, "PRE response crypto backend cannot be empty")
	}
	if err := decoder.finish(); err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	for label, blob := range map[string][]byte{
		"rdr_pk":    rdrPk,
		"share":     share,
		"challenge": challenge,
		"proof":     proof,
	} {
		if len(blob) == 0 {
			return invalidCryptoResponseStatement{}, errorsmod.Wrapf(types.ErrInvalidReport, "PRE response %s cannot be empty", label)
		}
		if len(blob) > preResponseMaxElementLen {
			return invalidCryptoResponseStatement{}, errorsmod.Wrapf(types.ErrInvalidReport, "PRE response %s exceeds size bound", label)
		}
	}

	return invalidCryptoResponseStatement{
		chainID:               chainID,
		ringID:                ringID,
		ringPk:                ringPk,
		ringStateSha256:       ringStateSha256,
		protocolVersion:       protocolVersion,
		requestID:             requestID,
		signedAt:              signedAt,
		responderNodeKey:      responderNodeKey,
		originProtocol:        offlineOriginProtocolPRE,
		accusedCommitteeScope: committeeScopeCurrent,
		signingCommitteeScope: committeeScopeCurrent,
	}, nil
}

func decodeSignResponseStatement(statementBytes []byte) (invalidCryptoResponseStatement, error) {
	decoder := newReportCanonicalDecoder(statementBytes)
	domain, err := decoder.readString("domain")
	if err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	if domain != SignResponseDomain {
		return invalidCryptoResponseStatement{}, errorsmod.Wrapf(types.ErrInvalidReport, "unexpected Sign response domain %q", domain)
	}
	chainID, err := decoder.readString("chain_id")
	if err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	ringID, err := decoder.readString("ring_id")
	if err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	ringPk, err := decoder.readString("ring_pk")
	if err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	ringStateSha256, err := decoder.readString("ring_state_sha256")
	if err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	protocolVersion, err := decoder.readU64("protocol_version")
	if err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	requestID, err := decoder.readString("request_id")
	if err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	signedAt, err := decoder.readU64("signed_at")
	if err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	responderNodeKey, err := decoder.readString("responder_node_key")
	if err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	originProtocol, err := decoder.readString("origin_protocol")
	if err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	if !isValidInvalidCryptoOriginProtocol(originProtocol) {
		return invalidCryptoResponseStatement{}, errorsmod.Wrapf(types.ErrInvalidReport, "unsupported Sign response origin protocol %q", originProtocol)
	}
	accusedScope, err := decoder.readByte("accused_committee_scope")
	if err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	if !isValidReportCommitteeScope(accusedScope) {
		return invalidCryptoResponseStatement{}, errorsmod.Wrapf(types.ErrInvalidReport, "unknown accused committee scope %d", accusedScope)
	}
	signingScope, err := decoder.readByte("signing_committee_scope")
	if err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	if !isValidReportCommitteeScope(signingScope) {
		return invalidCryptoResponseStatement{}, errorsmod.Wrapf(types.ErrInvalidReport, "unknown signing committee scope %d", signingScope)
	}
	if _, err := decoder.readU32("from_node_id"); err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	message, err := decoder.readBytes("message")
	if err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	signingCommitments, err := decoder.readBytes("signing_commitments")
	if err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	derivation, err := decoder.readOptionalBytes("derivation")
	if err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	metadata, err := decoder.readOptionalBytes("metadata")
	if err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	sigShare, err := decoder.readBytes("sig_share")
	if err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	cryptoBackend, err := decoder.readString("crypto_backend")
	if err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	if err := decoder.finish(); err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	switch {
	case requestID == "" || responderNodeKey == "":
		return invalidCryptoResponseStatement{}, errorsmod.Wrap(types.ErrInvalidReport, "Sign response statement has empty identity fields")
	case cryptoBackend == "":
		return invalidCryptoResponseStatement{}, errorsmod.Wrap(types.ErrInvalidReport, "Sign response crypto backend cannot be empty")
	case len(message) == 0:
		return invalidCryptoResponseStatement{}, errorsmod.Wrap(types.ErrInvalidReport, "Sign response message cannot be empty")
	case len(message) > signResponseMaxMessageLen:
		return invalidCryptoResponseStatement{}, errorsmod.Wrap(types.ErrInvalidReport, "Sign response message exceeds size bound")
	case len(signingCommitments) > signResponseMaxCommitmentsLen:
		return invalidCryptoResponseStatement{}, errorsmod.Wrap(types.ErrInvalidReport, "Sign response commitments exceed size bound")
	case len(derivation) > signResponseMaxElementLen:
		return invalidCryptoResponseStatement{}, errorsmod.Wrap(types.ErrInvalidReport, "Sign response derivation exceeds size bound")
	case len(metadata) > signResponseMaxElementLen:
		return invalidCryptoResponseStatement{}, errorsmod.Wrap(types.ErrInvalidReport, "Sign response metadata exceeds size bound")
	case len(sigShare) == 0:
		return invalidCryptoResponseStatement{}, errorsmod.Wrap(types.ErrInvalidReport, "Sign response sig_share cannot be empty")
	case len(sigShare) > signResponseMaxElementLen:
		return invalidCryptoResponseStatement{}, errorsmod.Wrap(types.ErrInvalidReport, "Sign response sig_share exceeds size bound")
	}

	return invalidCryptoResponseStatement{
		chainID:               chainID,
		ringID:                ringID,
		ringPk:                ringPk,
		ringStateSha256:       ringStateSha256,
		protocolVersion:       protocolVersion,
		requestID:             requestID,
		signedAt:              signedAt,
		responderNodeKey:      responderNodeKey,
		originProtocol:        originProtocol,
		accusedCommitteeScope: accusedScope,
		signingCommitteeScope: signingScope,
	}, nil
}

func decodeDkgShareStatement(statementBytes []byte) (invalidCryptoResponseStatement, error) {
	if len(statementBytes) > dkgStatementMaxLen {
		return invalidCryptoResponseStatement{}, errorsmod.Wrap(types.ErrInvalidReport, "DKG share statement exceeds size bound")
	}
	decoder := newReportCanonicalDecoder(statementBytes)
	domain, err := decoder.readString("domain")
	if err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	if domain != DkgShareDomain {
		return invalidCryptoResponseStatement{}, errorsmod.Wrapf(types.ErrInvalidReport, "unexpected DKG share domain %q", domain)
	}
	chainID, err := decoder.readString("chain_id")
	if err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	ringID, err := decoder.readString("ring_id")
	if err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	ringPk, err := decoder.readString("ring_pk")
	if err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	ringStateSha256, err := decoder.readString("ring_state_sha256")
	if err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	protocolVersion, err := decoder.readU64("protocol_version")
	if err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	requestID, err := decoder.readString("request_id")
	if err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	signedAt, err := decoder.readU64("signed_at")
	if err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	responderNodeKey, err := decoder.readString("responder_node_key")
	if err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	receiverNodeKey, err := decoder.readString("receiver_node_key")
	if err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	originProtocol, err := decoder.readString("origin_protocol")
	if err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	if !isValidInvalidCryptoDkgOriginProtocol(originProtocol) {
		return invalidCryptoResponseStatement{}, errorsmod.Wrapf(types.ErrInvalidReport, "unsupported DKG share origin protocol %q", originProtocol)
	}
	accusedScope, err := decoder.readByte("accused_committee_scope")
	if err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	if !isValidReportCommitteeScope(accusedScope) {
		return invalidCryptoResponseStatement{}, errorsmod.Wrapf(types.ErrInvalidReport, "unknown accused committee scope %d", accusedScope)
	}
	signingScope, err := decoder.readByte("signing_committee_scope")
	if err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	if !isValidReportCommitteeScope(signingScope) {
		return invalidCryptoResponseStatement{}, errorsmod.Wrapf(types.ErrInvalidReport, "unknown signing committee scope %d", signingScope)
	}
	fromNodeID, err := decoder.readU32("from_node_id")
	if err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	toNodeID, err := decoder.readU32("to_node_id")
	if err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	commitmentStatementBytes, err := decoder.readBytes("commitment_statement")
	if err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	commitmentSignature, err := decoder.readBytes("commitment_signature")
	if err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	shareValue, err := decoder.readBytes("share_value")
	if err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	nonce, err := decoder.readBytes("nonce")
	if err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	cryptoBackend, err := decoder.readString("crypto_backend")
	if err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	if err := decoder.finish(); err != nil {
		return invalidCryptoResponseStatement{}, err
	}

	switch {
	case requestID == "" || responderNodeKey == "" || receiverNodeKey == "":
		return invalidCryptoResponseStatement{}, errorsmod.Wrap(types.ErrInvalidReport, "DKG share statement has empty identity fields")
	case accusedScope != committeeScopeCurrent || signingScope != committeeScopeCurrent:
		return invalidCryptoResponseStatement{}, errorsmod.Wrap(types.ErrInvalidReport, "DKG share reports must use current accused and signing scopes")
	case fromNodeID == 0 || toNodeID == 0:
		return invalidCryptoResponseStatement{}, errorsmod.Wrap(types.ErrInvalidReport, "DKG share node IDs must be non-zero")
	case len(commitmentStatementBytes) == 0:
		return invalidCryptoResponseStatement{}, errorsmod.Wrap(types.ErrInvalidReport, "DKG commitment statement cannot be empty")
	case len(commitmentStatementBytes) > dkgStatementMaxLen:
		return invalidCryptoResponseStatement{}, errorsmod.Wrap(types.ErrInvalidReport, "DKG commitment statement exceeds size bound")
	case len(commitmentSignature) != 64:
		return invalidCryptoResponseStatement{}, errorsmod.Wrap(types.ErrInvalidReport, "DKG commitment signature must be 64 bytes")
	case len(shareValue) == 0:
		return invalidCryptoResponseStatement{}, errorsmod.Wrap(types.ErrInvalidReport, "DKG share value cannot be empty")
	case len(shareValue) > dkgShareMaxElementLen:
		return invalidCryptoResponseStatement{}, errorsmod.Wrap(types.ErrInvalidReport, "DKG share value exceeds size bound")
	case len(nonce) != dkgNonceLen:
		return invalidCryptoResponseStatement{}, errorsmod.Wrapf(types.ErrInvalidReport, "DKG share nonce must be %d bytes", dkgNonceLen)
	case cryptoBackend == "":
		return invalidCryptoResponseStatement{}, errorsmod.Wrap(types.ErrInvalidReport, "DKG share crypto backend cannot be empty")
	}

	commitment, err := decodeDkgCommitmentStatement(commitmentStatementBytes)
	if err != nil {
		return invalidCryptoResponseStatement{}, err
	}
	if commitment.chainID != chainID ||
		commitment.ringID != ringID ||
		commitment.ringPk != ringPk ||
		commitment.ringStateSha256 != ringStateSha256 ||
		commitment.protocolVersion != protocolVersion ||
		commitment.requestID != requestID ||
		commitment.responderNodeKey != responderNodeKey ||
		commitment.originProtocol != originProtocol ||
		commitment.accusedCommitteeScope != accusedScope ||
		commitment.signingCommitteeScope != signingScope ||
		commitment.fromNodeID != fromNodeID ||
		commitment.cryptoBackend != cryptoBackend {
		return invalidCryptoResponseStatement{}, errorsmod.Wrap(types.ErrInvalidReport, "DKG commitment binding does not match DKG share statement")
	}
	if commitment.signedAt > signedAt {
		return invalidCryptoResponseStatement{}, errorsmod.Wrap(types.ErrInvalidReport, "DKG commitment was signed after the DKG share")
	}

	return invalidCryptoResponseStatement{
		chainID:               chainID,
		ringID:                ringID,
		ringPk:                ringPk,
		ringStateSha256:       ringStateSha256,
		protocolVersion:       protocolVersion,
		requestID:             requestID,
		signedAt:              signedAt,
		responderNodeKey:      responderNodeKey,
		originProtocol:        originProtocol,
		accusedCommitteeScope: accusedScope,
		signingCommitteeScope: signingScope,
	}, nil
}

func decodeDkgCommitmentStatement(statementBytes []byte) (dkgCommitmentStatement, error) {
	if len(statementBytes) > dkgStatementMaxLen {
		return dkgCommitmentStatement{}, errorsmod.Wrap(types.ErrInvalidReport, "DKG commitment statement exceeds size bound")
	}
	decoder := newReportCanonicalDecoder(statementBytes)
	domain, err := decoder.readString("domain")
	if err != nil {
		return dkgCommitmentStatement{}, err
	}
	if domain != DkgCommitmentDomain {
		return dkgCommitmentStatement{}, errorsmod.Wrapf(types.ErrInvalidReport, "unexpected DKG commitment domain %q", domain)
	}
	chainID, err := decoder.readString("chain_id")
	if err != nil {
		return dkgCommitmentStatement{}, err
	}
	ringID, err := decoder.readString("ring_id")
	if err != nil {
		return dkgCommitmentStatement{}, err
	}
	ringPk, err := decoder.readString("ring_pk")
	if err != nil {
		return dkgCommitmentStatement{}, err
	}
	ringStateSha256, err := decoder.readString("ring_state_sha256")
	if err != nil {
		return dkgCommitmentStatement{}, err
	}
	protocolVersion, err := decoder.readU64("protocol_version")
	if err != nil {
		return dkgCommitmentStatement{}, err
	}
	requestID, err := decoder.readString("request_id")
	if err != nil {
		return dkgCommitmentStatement{}, err
	}
	signedAt, err := decoder.readU64("signed_at")
	if err != nil {
		return dkgCommitmentStatement{}, err
	}
	responderNodeKey, err := decoder.readString("responder_node_key")
	if err != nil {
		return dkgCommitmentStatement{}, err
	}
	originProtocol, err := decoder.readString("origin_protocol")
	if err != nil {
		return dkgCommitmentStatement{}, err
	}
	if !isValidInvalidCryptoDkgOriginProtocol(originProtocol) {
		return dkgCommitmentStatement{}, errorsmod.Wrapf(types.ErrInvalidReport, "unsupported DKG commitment origin protocol %q", originProtocol)
	}
	accusedScope, err := decoder.readByte("accused_committee_scope")
	if err != nil {
		return dkgCommitmentStatement{}, err
	}
	if !isValidReportCommitteeScope(accusedScope) {
		return dkgCommitmentStatement{}, errorsmod.Wrapf(types.ErrInvalidReport, "unknown accused committee scope %d", accusedScope)
	}
	signingScope, err := decoder.readByte("signing_committee_scope")
	if err != nil {
		return dkgCommitmentStatement{}, err
	}
	if !isValidReportCommitteeScope(signingScope) {
		return dkgCommitmentStatement{}, errorsmod.Wrapf(types.ErrInvalidReport, "unknown signing committee scope %d", signingScope)
	}
	fromNodeID, err := decoder.readU32("from_node_id")
	if err != nil {
		return dkgCommitmentStatement{}, err
	}
	commitment, err := decoder.readBytes("commitment")
	if err != nil {
		return dkgCommitmentStatement{}, err
	}
	cryptoBackend, err := decoder.readString("crypto_backend")
	if err != nil {
		return dkgCommitmentStatement{}, err
	}
	if err := decoder.finish(); err != nil {
		return dkgCommitmentStatement{}, err
	}

	switch {
	case requestID == "" || responderNodeKey == "":
		return dkgCommitmentStatement{}, errorsmod.Wrap(types.ErrInvalidReport, "DKG commitment statement has empty identity fields")
	case accusedScope != committeeScopeCurrent || signingScope != committeeScopeCurrent:
		return dkgCommitmentStatement{}, errorsmod.Wrap(types.ErrInvalidReport, "DKG commitment reports must use current accused and signing scopes")
	case fromNodeID == 0:
		return dkgCommitmentStatement{}, errorsmod.Wrap(types.ErrInvalidReport, "DKG commitment from_node_id must be non-zero")
	case len(commitment) == 0:
		return dkgCommitmentStatement{}, errorsmod.Wrap(types.ErrInvalidReport, "DKG commitment cannot be empty")
	case len(commitment) > dkgCommitmentMaxLen:
		return dkgCommitmentStatement{}, errorsmod.Wrap(types.ErrInvalidReport, "DKG commitment exceeds size bound")
	case cryptoBackend == "":
		return dkgCommitmentStatement{}, errorsmod.Wrap(types.ErrInvalidReport, "DKG commitment crypto backend cannot be empty")
	}

	return dkgCommitmentStatement{
		chainID:               chainID,
		ringID:                ringID,
		ringPk:                ringPk,
		ringStateSha256:       ringStateSha256,
		protocolVersion:       protocolVersion,
		requestID:             requestID,
		signedAt:              signedAt,
		responderNodeKey:      responderNodeKey,
		originProtocol:        originProtocol,
		accusedCommitteeScope: accusedScope,
		signingCommitteeScope: signingScope,
		fromNodeID:            fromNodeID,
		commitment:            commitment,
		cryptoBackend:         cryptoBackend,
	}, nil
}

// validateInvalidCryptoResponseStatement binds the signed statement to the report
// envelope. Pinning observed_at to signed_at - grace makes the envelope's
// fixed observed_at+TTL expiry double as the evidence expiry, so the shared
// envelope shape checks (observed_at <= now, now <= expires_at) already bound
// how long this evidence stays reportable — no separate timing rules needed.
func validateInvalidCryptoResponseStatement(report *types.ReportEnvelope, statement invalidCryptoResponseStatement) error {
	switch {
	case statement.chainID != report.ChainId:
		return errorsmod.Wrap(types.ErrInvalidReport, "response chain ID does not match report envelope")
	case statement.ringID != report.RingId,
		statement.ringPk != report.RingPk,
		statement.ringStateSha256 != report.RingStateSha256:
		return errorsmod.Wrap(types.ErrInvalidReport, "response ring binding does not match report envelope")
	case statement.requestID != report.SessionId:
		return errorsmod.Wrap(types.ErrInvalidReport, "response request_id does not match report session_id")
	case statement.responderNodeKey != report.AccusedNodeKey:
		return errorsmod.Wrap(types.ErrInvalidReport, "response responder does not match accused node")
	case statement.signedAt < reportObservedAtGraceSecs,
		report.ObservedAt != statement.signedAt-reportObservedAtGraceSecs:
		return errorsmod.Wrap(types.ErrInvalidReport, "report envelope is not anchored to the evidence timestamp")
	}
	return nil
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

func isValidInvalidCryptoOriginProtocol(originProtocol string) bool {
	switch originProtocol {
	case offlineOriginProtocolPRE,
		offlineOriginProtocolSign,
		offlineOriginProtocolPSSRefresh,
		offlineOriginProtocolPSSReshare,
		originProtocolReport:
		return true
	default:
		return false
	}
}

func isValidInvalidCryptoDkgOriginProtocol(originProtocol string) bool {
	switch originProtocol {
	case offlineOriginProtocolPSSRefresh,
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
	payload reportPayload,
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

func (k *Keeper) maybeScheduleAutoReshareForReport(
	goCtx context.Context,
	ctx sdk.Context,
	ring *types.Ring,
	accusedNodeKey string,
	effectiveDemerits uint64,
) error {
	if effectiveDemerits < ring.Reporting.KickThreshold {
		return nil
	}
	if len(ring.NewPeerNodeKeys) > 0 || ring.XNewThreshold != nil {
		return nil
	}
	if !slices.Contains(ring.PeerNodeKeys, accusedNodeKey) {
		return nil
	}

	replacementNodeKey, remainingBackups, found := k.selectAutoReshareReplacement(goCtx, ring, accusedNodeKey)
	if !found {
		return nil
	}

	nextPeerNodeKeys := make([]string, 0, len(ring.PeerNodeKeys))
	for _, nodeKey := range ring.PeerNodeKeys {
		if nodeKey != accusedNodeKey {
			nextPeerNodeKeys = append(nextPeerNodeKeys, nodeKey)
		}
	}
	nextPeerNodeKeys = append(nextPeerNodeKeys, replacementNodeKey)

	nextRing := *ring
	nextRing.NewPeerNodeKeys = canonicalNodeKeys(nextPeerNodeKeys)
	nextRing.XNewThreshold = nil
	nextRing.Reporting.BackupNodeKeys = remainingBackups
	if err := validateRing(&nextRing); err != nil {
		return err
	}
	k.SetRing(goCtx, nextRing)

	return ctx.EventManager().EmitTypedEvent(&types.EventRingAutoReshareScheduled{
		RingId:             ring.Id,
		AccusedNodeKey:     accusedNodeKey,
		ReplacementNodeKey: replacementNodeKey,
		KickThreshold:      ring.Reporting.KickThreshold,
		Demerits:           effectiveDemerits,
	})
}

func (k *Keeper) selectAutoReshareReplacement(
	goCtx context.Context,
	ring *types.Ring,
	accusedNodeKey string,
) (string, []string, bool) {
	active := make(map[string]struct{}, len(ring.PeerNodeKeys))
	for _, nodeKey := range ring.PeerNodeKeys {
		active[nodeKey] = struct{}{}
	}

	for i, backupNodeKey := range ring.Reporting.BackupNodeKeys {
		if backupNodeKey == accusedNodeKey {
			continue
		}
		if _, alreadyActive := active[backupNodeKey]; alreadyActive {
			continue
		}
		nodeInfo := k.GetNodeInfo(goCtx, backupNodeKey)
		if nodeInfo == nil || !nodeInfoAllowsRing(nodeInfo, ring) {
			continue
		}

		remainingBackups := slices.Clone(ring.Reporting.BackupNodeKeys[:i])
		remainingBackups = append(remainingBackups, ring.Reporting.BackupNodeKeys[i+1:]...)
		return backupNodeKey, remainingBackups, true
	}

	return "", slices.Clone(ring.Reporting.BackupNodeKeys), false
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
	w.writeReportingConfig(ring.Reporting)
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

func (w *reportCanonicalWriter) writeDemeritConfig(value types.DemeritConfig) {
	w.writeU64(value.NodeOfflineDemerits)
	w.writeU64(value.InvalidCryptoResponseDemerits)
	w.writeU64(value.ResetIntervalSeconds)
}

func (w *reportCanonicalWriter) writeReportingConfig(value types.ReportingConfig) {
	w.writeDemeritConfig(value.DemeritConfig)
	w.writeStringSlice(value.BackupNodeKeys)
	w.writeU64(value.KickThreshold)
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

func (d *reportCanonicalDecoder) readBytes(label string) ([]byte, error) {
	length, err := d.readU32(label + "_length")
	if err != nil {
		return nil, err
	}
	if uint64(len(d.bytes)-d.cursor) < uint64(length) {
		return nil, errorsmod.Wrapf(types.ErrInvalidReport, "truncated %s", label)
	}
	value := d.bytes[d.cursor : d.cursor+int(length)]
	d.cursor += int(length)
	return value, nil
}

func (d *reportCanonicalDecoder) readOptionalBytes(label string) ([]byte, error) {
	tag, err := d.readByte(label + "_present")
	if err != nil {
		return nil, err
	}
	switch tag {
	case 0:
		return nil, nil
	case 1:
		return d.readBytes(label)
	default:
		return nil, errorsmod.Wrapf(types.ErrInvalidReport, "invalid optional %s tag %d", label, tag)
	}
}

func (d *reportCanonicalDecoder) finish() error {
	if d.cursor != len(d.bytes) {
		return errorsmod.Wrap(types.ErrInvalidReport, "trailing payload bytes")
	}
	return nil
}
