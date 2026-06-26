package keeper

import (
	"crypto/sha256"
	"encoding/hex"
	"math/big"
	"strings"
	"testing"
	"time"

	decaf377 "github.com/mizufinance/decaf377-go"
	"github.com/mizufinance/decaf377-go/orbisfrost"
	"github.com/stretchr/testify/require"
	blst "github.com/supranational/blst/bindings/go"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"

	"github.com/sourcenetwork/sourcehub/x/orbis/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const reportTestChainID = "sourcehub-test"
const reportTestObservedAt = uint64(1_700_000_000)

func TestReportCanonicalEncodingMatchesRustGoldenVectors(t *testing.T) {
	payload := nodeOfflinePayloadForTest("pre", 0, committeeScopeCurrent, committeeScopeCurrent)
	report := types.ReportEnvelope{
		Domain:          ReportDomain,
		ReportType:      NodeOfflineReportType,
		ChainId:         reportTestChainID,
		RingId:          "ring-1",
		RingPk:          "aabb",
		RingStateSha256: strings.Repeat("11", 32),
		ReporterNodeKey: "reporter",
		AccusedNodeKey:  "accused",
		AccusedPeerId:   strings.Repeat("22", 32),
		ObservedAt:      reportTestObservedAt,
		ExpiresAt:       reportTestObservedAt + ReportTTLSeconds,
		Payload:         payload,
	}

	_, reportID, err := reportEnvelopeCanonicalMessageAndID(&report)
	require.NoError(t, err)
	require.Equal(t, "dfb170015fd469566dadfedadf6ff110f840e6a1e53b35a2850581bcf74da797", reportID)

	ring := &types.Ring{
		RingPk:       "pk",
		PeerNodeKeys: []string{"b", "a"},
		Threshold:    2,
		PssInterval:  types.MinPSSIntervalSeconds,
		UpgradeInfo: types.UpgradeInfo{
			CurrentVersion: 0,
		},
		DemeritConfig: types.DefaultDemeritConfig(),
	}
	ringHash, err := reportRingStateSHA256(ring)
	require.NoError(t, err)
	require.Equal(t, "a597b5c00a60c75728b40780bf26efe66150560ca3f511264e7f804e3bd2c870", ringHash)
}

func TestNodeOfflinePayloadDecodeRejectsMalformedPayloads(t *testing.T) {
	valid := nodeOfflinePayloadForTest("pre", 0, committeeScopeCurrent, committeeScopePendingNew)
	decoded, err := decodeNodeOfflinePayload(valid)
	require.NoError(t, err)
	require.Equal(t, "pre", decoded.originProtocol)
	require.Equal(t, uint64(0), decoded.originProtocolVersion)
	require.Equal(t, committeeScopeCurrent, decoded.accusedCommitteeScope)
	require.Equal(t, committeeScopePendingNew, decoded.signingCommitteeScope)

	unknownAccusedScope := append([]byte{}, valid...)
	unknownAccusedScope[len(unknownAccusedScope)-2] = 99
	_, err = decodeNodeOfflinePayload(unknownAccusedScope)
	require.ErrorIs(t, err, types.ErrInvalidReport)

	unknownSigningScope := append([]byte{}, valid...)
	unknownSigningScope[len(unknownSigningScope)-1] = 99
	_, err = decodeNodeOfflinePayload(unknownSigningScope)
	require.ErrorIs(t, err, types.ErrInvalidReport)

	trailing := append(append([]byte{}, valid...), 0)
	_, err = decodeNodeOfflinePayload(trailing)
	require.ErrorIs(t, err, types.ErrInvalidReport)

	_, err = decodeNodeOfflinePayload(valid[:3])
	require.ErrorIs(t, err, types.ErrInvalidReport)
}

func TestDemeritAmountForReportTypeRejectsInvalidInputs(t *testing.T) {
	_, err := DemeritAmountForReportType(nil, NodeOfflineReportType)
	require.ErrorIs(t, err, types.ErrRingNotFound)

	_, err = DemeritAmountForReportType(&types.Ring{
		DemeritConfig: types.DefaultDemeritConfig(),
	}, "unknown")
	require.ErrorIs(t, err, types.ErrInvalidReport)

	amount, err := DemeritAmountForReportType(&types.Ring{
		DemeritConfig: types.DemeritConfig{
			NodeOfflineDemerits:   5,
			ResetIntervalSeconds: types.DefaultDemeritResetIntervalSecs,
		},
	}, NodeOfflineReportType)
	require.NoError(t, err)
	require.Equal(t, uint64(5), amount)
}

func TestMsgServer_SubmitReport_BLS12381AcceptsAndRejectsReplay(t *testing.T) {
	fixture := setupReportTestFixture(t)
	ikm := make([]byte, 32)
	copy(ikm, "orbis-report-bls-ikm-000000000")
	sk := blst.KeyGen(ikm)
	ringPk := hex.EncodeToString(new(blst.P1Affine).From(sk).Compress())
	fixture.setRing(t, ringPk, 2)

	report := fixture.validReport(t, committeeScopeCurrent, committeeScopeCurrent, 0)
	msg := fixture.signBLSReport(t, sk, report)

	resp, err := fixture.k.SubmitReport(fixture.ctx, msg)
	require.NoError(t, err)
	require.Equal(t, msg.ReportId, resp.ReportId)
	require.True(t, fixture.k.HasAcceptedReport(fixture.ctx, msg.ReportId))
	require.Equal(t, uint64(1), fixture.k.GetNodeDemerits(fixture.ctx, fixture.ringID, fixture.accusedKey))
	queryResp, err := fixture.k.NodeDemerits(fixture.ctx, &types.QueryNodeDemeritsRequest{
		RingId:  fixture.ringID,
		NodeKey: fixture.accusedKey,
	})
	require.NoError(t, err)
	require.Equal(t, uint64(1), queryResp.Points)
	require.Equal(t, *fixture.originalRing, *fixture.k.GetRing(fixture.ctx, fixture.ringID))

	events := parseTypedEvents(t, fixture.ctx)
	require.Contains(t, events, &types.EventReportAccepted{
		ReportId:        msg.ReportId,
		RingId:          fixture.ringID,
		ReportType:      NodeOfflineReportType,
		ReporterNodeKey: fixture.reporterKey,
		AccusedNodeKey:  fixture.accusedKey,
	})

	_, err = fixture.k.SubmitReport(fixture.ctx, msg)
	require.ErrorIs(t, err, types.ErrReportAlreadyAccepted)
	require.Equal(t, uint64(1), fixture.k.GetNodeDemerits(fixture.ctx, fixture.ringID, fixture.accusedKey))
}

func TestMsgServer_SubmitReportIncrementsDemeritsForDistinctReports(t *testing.T) {
	fixture := setupReportTestFixture(t)
	ikm := make([]byte, 32)
	copy(ikm, "orbis-report-demerit-ikm-0000")
	sk := blst.KeyGen(ikm)
	fixture.setRingWithDemeritConfig(t, hex.EncodeToString(new(blst.P1Affine).From(sk).Compress()), 2, types.DemeritConfig{
		NodeOfflineDemerits:   3,
		ResetIntervalSeconds: types.DefaultDemeritResetIntervalSecs,
	})

	first := fixture.signBLSReport(t, sk, fixture.validReport(t, committeeScopeCurrent, committeeScopeCurrent, 0))
	_, err := fixture.k.SubmitReport(fixture.ctx, first)
	require.NoError(t, err)
	require.Equal(t, uint64(3), fixture.k.GetNodeDemerits(fixture.ctx, fixture.ringID, fixture.accusedKey))

	secondReport := fixture.validReport(t, committeeScopeCurrent, committeeScopeCurrent, 0)
	secondReport.ObservedAt++
	secondReport.ExpiresAt++
	second := fixture.signBLSReport(t, sk, secondReport)
	_, err = fixture.k.SubmitReport(fixture.ctx, second)
	require.NoError(t, err)
	require.Equal(t, uint64(6), fixture.k.GetNodeDemerits(fixture.ctx, fixture.ringID, fixture.accusedKey))
}

func TestMsgServer_SubmitReportDemeritsAreIsolatedByNodeAndRing(t *testing.T) {
	fixture := setupReportTestFixture(t)
	ikm := make([]byte, 32)
	copy(ikm, "orbis-report-isolate-ikm-000")
	sk := blst.KeyGen(ikm)
	fixture.setRing(t, hex.EncodeToString(new(blst.P1Affine).From(sk).Compress()), 2)

	accusedReport := fixture.validReport(t, committeeScopeCurrent, committeeScopeCurrent, 0)
	_, err := fixture.k.SubmitReport(fixture.ctx, fixture.signBLSReport(t, sk, accusedReport))
	require.NoError(t, err)

	validatorReport := fixture.validReport(t, committeeScopeCurrent, committeeScopeCurrent, 0)
	validatorReport.AccusedNodeKey = fixture.validatorKey
	validatorReport.AccusedPeerId = "12D3KooWReportValidator"
	_, err = fixture.k.SubmitReport(fixture.ctx, fixture.signBLSReport(t, sk, validatorReport))
	require.NoError(t, err)

	secondRingID := "report-ring-2"
	secondRing := *fixture.originalRing
	secondRing.Id = secondRingID
	fixture.k.SetRing(fixture.ctx, secondRing)
	secondRingReport := fixture.validReport(t, committeeScopeCurrent, committeeScopeCurrent, 0)
	secondRingReport.RingId = secondRingID
	_, err = fixture.k.SubmitReport(fixture.ctx, fixture.signBLSReport(t, sk, secondRingReport))
	require.NoError(t, err)

	require.Equal(t, uint64(1), fixture.k.GetNodeDemerits(fixture.ctx, fixture.ringID, fixture.accusedKey))
	require.Equal(t, uint64(1), fixture.k.GetNodeDemerits(fixture.ctx, fixture.ringID, fixture.validatorKey))
	require.Equal(t, uint64(1), fixture.k.GetNodeDemerits(fixture.ctx, secondRingID, fixture.accusedKey))
	require.Equal(t, uint64(0), fixture.k.GetNodeDemerits(fixture.ctx, secondRingID, fixture.validatorKey))

	accusedResp, err := fixture.k.NodeDemerits(fixture.ctx, &types.QueryNodeDemeritsRequest{
		RingId:  fixture.ringID,
		NodeKey: fixture.accusedKey,
	})
	require.NoError(t, err)
	require.Equal(t, uint64(1), accusedResp.Points)

	validatorResp, err := fixture.k.NodeDemerits(fixture.ctx, &types.QueryNodeDemeritsRequest{
		RingId:  fixture.ringID,
		NodeKey: fixture.validatorKey,
	})
	require.NoError(t, err)
	require.Equal(t, uint64(1), validatorResp.Points)

	secondRingValidatorResp, err := fixture.k.NodeDemerits(fixture.ctx, &types.QueryNodeDemeritsRequest{
		RingId:  secondRingID,
		NodeKey: fixture.validatorKey,
	})
	require.NoError(t, err)
	require.Equal(t, uint64(0), secondRingValidatorResp.Points)
}

func TestQueryServer_NodeDemeritsValidationAndEmptyScore(t *testing.T) {
	fixture := setupReportTestFixture(t)
	fixture.setRing(t, "query-ring-pk", 2)

	resp, err := fixture.k.NodeDemerits(fixture.ctx, &types.QueryNodeDemeritsRequest{
		RingId:  fixture.ringID,
		NodeKey: "unknown-node",
	})
	require.NoError(t, err)
	require.Equal(t, uint64(0), resp.Points)

	_, err = fixture.k.NodeDemerits(fixture.ctx, nil)
	require.Equal(t, codes.InvalidArgument, status.Code(err))

	_, err = fixture.k.NodeDemerits(fixture.ctx, &types.QueryNodeDemeritsRequest{
		NodeKey: fixture.accusedKey,
	})
	require.Equal(t, codes.InvalidArgument, status.Code(err))

	_, err = fixture.k.NodeDemerits(fixture.ctx, &types.QueryNodeDemeritsRequest{
		RingId: fixture.ringID,
	})
	require.Equal(t, codes.InvalidArgument, status.Code(err))

	_, err = fixture.k.NodeDemerits(fixture.ctx, &types.QueryNodeDemeritsRequest{
		RingId:  "missing-ring",
		NodeKey: fixture.accusedKey,
	})
	require.Equal(t, codes.NotFound, status.Code(err))
}

func TestMsgServer_SubmitReportRejectedReportDoesNotIncrementDemerits(t *testing.T) {
	fixture := setupReportTestFixture(t)
	ikm := make([]byte, 32)
	copy(ikm, "orbis-report-no-demerit-0000")
	sk := blst.KeyGen(ikm)
	fixture.setRing(t, hex.EncodeToString(new(blst.P1Affine).From(sk).Compress()), 2)

	report := fixture.validReport(t, committeeScopeCurrent, committeeScopeCurrent, 0)
	report.Domain = "wrong"
	_, err := fixture.k.SubmitReport(fixture.ctx, fixture.signBLSReport(t, sk, report))
	require.ErrorIs(t, err, types.ErrInvalidReport)
	require.Equal(t, uint64(0), fixture.k.GetNodeDemerits(fixture.ctx, fixture.ringID, fixture.accusedKey))
}

func TestMsgServer_SubmitReport_Decaf377FROSTAccepts(t *testing.T) {
	fixture := setupReportTestFixture(t)
	secretScalar := new(big.Int).SetBytes([]byte("orbis-report-decaf377-secret-key"))
	secretScalar.Mod(secretScalar, decaf377.ScalarOrder())

	ringPkBytes, err := decaf377PublicKeyBytes(secretScalar)
	require.NoError(t, err)
	fixture.setRing(t, hex.EncodeToString(ringPkBytes), 2)

	report := fixture.validReport(t, committeeScopeCurrent, committeeScopeCurrent, 0)
	message, reportID, err := reportEnvelopeCanonicalMessageAndID(&report)
	require.NoError(t, err)
	signature, err := decaf377SchnorrSign(secretScalar, ringPkBytes, message)
	require.NoError(t, err)
	ok, err := orbisfrost.Verify(ringPkBytes, message, signature)
	require.NoError(t, err)
	require.True(t, ok)

	_, err = fixture.k.SubmitReport(fixture.ctx, &types.MsgSubmitReport{
		Creator:         fixture.creator,
		Report:          report,
		ReportId:        reportID,
		SignatureScheme: ThresholdSignatureSchemeDecaf377FROST,
		Signature:       signature,
	})
	require.NoError(t, err)
	require.True(t, fixture.k.HasAcceptedReport(fixture.ctx, reportID))
}

func TestMsgServer_SubmitReportAllowsPendingReshareScopes(t *testing.T) {
	fixture := setupReportTestFixture(t)
	ikm := make([]byte, 32)
	copy(ikm, "orbis-report-pending-ikm-0000")
	sk := blst.KeyGen(ikm)
	fixture.setRing(t, hex.EncodeToString(new(blst.P1Affine).From(sk).Compress()), 2)

	pendingPeerID := "12D3KooWReportPending"
	_, pendingKey := setupPeerWithNodeInfo(t, fixture.k, fixture.authKeeper, fixture.ctx, pendingPeerID)
	ring := *fixture.originalRing
	ring.NewPeerNodeKeys = []string{fixture.reporterKey, fixture.validatorKey, pendingKey}
	ring.XNewThreshold = &types.Ring_NewThreshold{NewThreshold: 2}
	fixture.k.SetRing(fixture.ctx, ring)
	fixture.originalRing = fixture.k.GetRing(fixture.ctx, fixture.ringID)

	t.Run("pending-new accused can be reported by current signing committee", func(t *testing.T) {
		report := fixture.validReport(t, committeeScopePendingNew, committeeScopeCurrent, 0)
		report.AccusedNodeKey = pendingKey
		report.AccusedPeerId = pendingPeerID
		msg := fixture.signBLSReport(t, sk, report)

		_, err := fixture.k.SubmitReport(fixture.ctx, msg)
		require.NoError(t, err)
		require.True(t, fixture.k.HasAcceptedReport(fixture.ctx, msg.ReportId))
	})

	t.Run("pending-new signing committee can authorize current accused report", func(t *testing.T) {
		report := fixture.validReport(t, committeeScopeCurrent, committeeScopePendingNew, 0)
		msg := fixture.signBLSReport(t, sk, report)

		_, err := fixture.k.SubmitReport(fixture.ctx, msg)
		require.NoError(t, err)
		require.True(t, fixture.k.HasAcceptedReport(fixture.ctx, msg.ReportId))
	})
}

func TestMsgServer_SubmitReportRejectsSecurityFailures(t *testing.T) {
	fixture := setupReportTestFixture(t)
	ikm := make([]byte, 32)
	copy(ikm, "orbis-report-reject-ikm-000000")
	sk := blst.KeyGen(ikm)
	fixture.setRing(t, hex.EncodeToString(new(blst.P1Affine).From(sk).Compress()), 2)

	t.Run("wrong fixed identity", func(t *testing.T) {
		report := fixture.validReport(t, committeeScopeCurrent, committeeScopeCurrent, 0)
		report.Domain = "wrong"
		msg := fixture.signBLSReport(t, sk, report)
		_, err := fixture.k.SubmitReport(fixture.ctx, msg)
		require.ErrorIs(t, err, types.ErrInvalidReport)
	})

	t.Run("report ID mismatch", func(t *testing.T) {
		msg := fixture.signBLSReport(t, sk, fixture.validReport(t, committeeScopeCurrent, committeeScopeCurrent, 0))
		msg.ReportId = "00" + msg.ReportId[2:]
		_, err := fixture.k.SubmitReport(fixture.ctx, msg)
		require.ErrorIs(t, err, types.ErrInvalidReport)
	})

	t.Run("unknown report type is rejected by dispatch", func(t *testing.T) {
		report := fixture.validReport(t, committeeScopeCurrent, committeeScopeCurrent, 0)
		report.ReportType = "future_fault"
		msg := fixture.signBLSReport(t, sk, report)
		_, err := fixture.k.SubmitReport(fixture.ctx, msg)
		require.ErrorIs(t, err, types.ErrInvalidReport)
	})

	t.Run("expired", func(t *testing.T) {
		ctx := fixture.ctx.WithBlockTime(time.Unix(int64(reportTestObservedAt+ReportTTLSeconds+1), 0))
		msg := fixture.signBLSReport(t, sk, fixture.validReport(t, committeeScopeCurrent, committeeScopeCurrent, 0))
		_, err := fixture.k.SubmitReport(ctx, msg)
		require.ErrorIs(t, err, types.ErrInvalidReport)
	})

	t.Run("stale ring digest from block nonce change", func(t *testing.T) {
		report := fixture.validReport(t, committeeScopeCurrent, committeeScopeCurrent, 0)
		ring := fixture.k.GetRing(fixture.ctx, fixture.ringID)
		ring.BlockNumberNonce++
		fixture.k.SetRing(fixture.ctx, *ring)
		defer fixture.k.SetRing(fixture.ctx, *fixture.originalRing)

		msg := fixture.signBLSReport(t, sk, report)
		_, err := fixture.k.SubmitReport(fixture.ctx, msg)
		require.ErrorIs(t, err, types.ErrInvalidReport)
	})

	t.Run("signature over report ID instead of canonical envelope", func(t *testing.T) {
		report := fixture.validReport(t, committeeScopeCurrent, committeeScopeCurrent, 0)
		_, reportID, err := reportEnvelopeCanonicalMessageAndID(&report)
		require.NoError(t, err)
		sig := new(blst.P2Affine).Sign(sk, []byte(reportID), []byte(bls12381G2SignatureDST))
		require.NotNil(t, sig)
		_, err = fixture.k.SubmitReport(fixture.ctx, &types.MsgSubmitReport{
			Creator:         fixture.creator,
			Report:          report,
			ReportId:        reportID,
			SignatureScheme: ThresholdSignatureSchemeBLS12381G1PKG2SigNUL,
			Signature:       sig.Compress(),
		})
		require.ErrorIs(t, err, types.ErrInvalidThresholdSignature)
	})

	t.Run("reporter outside signing committee", func(t *testing.T) {
		_, outsiderKey := setupPeerWithNodeInfo(t, fixture.k, fixture.authKeeper, fixture.ctx, "12D3KooWReportOutsider")
		report := fixture.validReport(t, committeeScopeCurrent, committeeScopeCurrent, 0)
		report.ReporterNodeKey = outsiderKey
		msg := fixture.signBLSReport(t, sk, report)
		_, err := fixture.k.SubmitReport(fixture.ctx, msg)
		require.ErrorIs(t, err, types.ErrInvalidReport)
	})

	t.Run("accused peer mismatch", func(t *testing.T) {
		report := fixture.validReport(t, committeeScopeCurrent, committeeScopeCurrent, 0)
		report.AccusedPeerId = "wrong-peer"
		msg := fixture.signBLSReport(t, sk, report)
		_, err := fixture.k.SubmitReport(fixture.ctx, msg)
		require.ErrorIs(t, err, types.ErrInvalidReport)
	})

	t.Run("threshold cannot exclude accused", func(t *testing.T) {
		ring := *fixture.originalRing
		ring.Threshold = 3
		originalRing := fixture.originalRing
		fixture.originalRing = &ring
		fixture.k.SetRing(fixture.ctx, ring)
		defer func() {
			fixture.originalRing = originalRing
			fixture.k.SetRing(fixture.ctx, *originalRing)
		}()

		report := fixture.validReport(t, committeeScopeCurrent, committeeScopeCurrent, 0)
		msg := fixture.signBLSReport(t, sk, report)
		_, err := fixture.k.SubmitReport(fixture.ctx, msg)
		require.ErrorIs(t, err, types.ErrInvalidReport)
	})

	t.Run("origin protocol version mismatch", func(t *testing.T) {
		report := fixture.validReport(t, committeeScopeCurrent, committeeScopeCurrent, 99)
		msg := fixture.signBLSReport(t, sk, report)
		_, err := fixture.k.SubmitReport(fixture.ctx, msg)
		require.ErrorIs(t, err, types.ErrInvalidReport)
	})

	t.Run("unknown origin protocol", func(t *testing.T) {
		report := fixture.validReport(t, committeeScopeCurrent, committeeScopeCurrent, 0)
		report.Payload = nodeOfflinePayloadForTest("unknown", 0, committeeScopeCurrent, committeeScopeCurrent)
		msg := fixture.signBLSReport(t, sk, report)
		_, err := fixture.k.SubmitReport(fixture.ctx, msg)
		require.ErrorIs(t, err, types.ErrInvalidReport)
	})
}

type reportTestFixture struct {
	k            Keeper
	authKeeper   authkeeper.AccountKeeper
	ctx          sdk.Context
	creator      string
	ringID       string
	reporterKey  string
	validatorKey string
	accusedKey   string
	accusedPeer  string
	originalRing *types.Ring
}

func setupReportTestFixture(t *testing.T) reportTestFixture {
	t.Helper()
	k, authKeeper, ctx := setupOrbisKeeper(t)
	ctx = ctx.
		WithChainID(reportTestChainID).
		WithBlockTime(time.Unix(int64(reportTestObservedAt), 0)).
		WithEventManager(sdk.NewEventManager())

	creator, _ := testAccountWithPubKey(t, ctx, authKeeper)
	_, reporterKey := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWReportReporter")
	_, validatorKey := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWReportValidator")
	_, accusedKey := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWReportAccused")

	return reportTestFixture{
		k:            k,
		authKeeper:   authKeeper,
		ctx:          ctx,
		creator:      creator,
		ringID:       "report-ring",
		reporterKey:  reporterKey,
		validatorKey: validatorKey,
		accusedKey:   accusedKey,
		accusedPeer:  "12D3KooWReportAccused",
	}
}

func (f *reportTestFixture) setRing(t *testing.T, ringPk string, threshold uint32) {
	f.setRingWithDemeritConfig(t, ringPk, threshold, types.DefaultDemeritConfig())
}

func (f *reportTestFixture) setRingWithDemeritConfig(t *testing.T, ringPk string, threshold uint32, demeritConfig types.DemeritConfig) {
	t.Helper()
	ring := types.Ring{
		Id:           f.ringID,
		CreatorDid:   testDID,
		RingPk:       ringPk,
		PeerNodeKeys: []string{f.reporterKey, f.validatorKey, f.accusedKey},
		Threshold:    threshold,
		PssInterval:  types.MinPSSIntervalSeconds,
		PolicyId:     "report-policy",
		UpgradeInfo: types.UpgradeInfo{
			CurrentVersion: 0,
		},
		DemeritConfig: demeritConfig,
	}
	f.k.SetRing(f.ctx, ring)
	stored := f.k.GetRing(f.ctx, f.ringID)
	require.NotNil(t, stored)
	f.originalRing = stored
}

func (f reportTestFixture) validReport(
	t *testing.T,
	accusedScope byte,
	signingScope byte,
	originProtocolVersion uint64,
) types.ReportEnvelope {
	t.Helper()
	require.NotNil(t, f.originalRing)
	ringDigest, err := reportRingStateSHA256(f.originalRing)
	require.NoError(t, err)

	return types.ReportEnvelope{
		Domain:          ReportDomain,
		ReportType:      NodeOfflineReportType,
		ChainId:         f.ctx.ChainID(),
		RingId:          f.ringID,
		RingPk:          f.originalRing.RingPk,
		RingStateSha256: ringDigest,
		ReporterNodeKey: f.reporterKey,
		AccusedNodeKey:  f.accusedKey,
		AccusedPeerId:   f.accusedPeer,
		ObservedAt:      reportTestObservedAt,
		ExpiresAt:       reportTestObservedAt + ReportTTLSeconds,
		Payload: nodeOfflinePayloadForTest(
			"pre",
			originProtocolVersion,
			accusedScope,
			signingScope,
		),
	}
}

func (f reportTestFixture) signBLSReport(t *testing.T, sk *blst.SecretKey, report types.ReportEnvelope) *types.MsgSubmitReport {
	t.Helper()
	message, reportID, err := reportEnvelopeCanonicalMessageAndID(&report)
	require.NoError(t, err)
	sig := new(blst.P2Affine).Sign(sk, message, []byte(bls12381G2SignatureDST))
	require.NotNil(t, sig)
	return &types.MsgSubmitReport{
		Creator:         f.creator,
		Report:          report,
		ReportId:        reportID,
		SignatureScheme: ThresholdSignatureSchemeBLS12381G1PKG2SigNUL,
		Signature:       sig.Compress(),
	}
}

func nodeOfflinePayloadForTest(
	originProtocol string,
	originProtocolVersion uint64,
	accusedScope byte,
	signingScope byte,
) []byte {
	w := newReportCanonicalWriter()
	w.writeString(originProtocol)
	w.writeU64(originProtocolVersion)
	w.bytes = append(w.bytes, accusedScope, signingScope)
	payload, err := w.finish()
	if err != nil {
		panic(err)
	}
	return payload
}

func reportIDForTest(t *testing.T, report *types.ReportEnvelope) string {
	t.Helper()
	message, err := reportEnvelopeCanonicalBytes(report)
	require.NoError(t, err)
	sum := sha256.Sum256(message)
	return hex.EncodeToString(sum[:])
}
