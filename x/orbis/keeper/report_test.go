package keeper

import (
	"bytes"
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
		SessionId:       "pre-request-1",
	}

	_, reportID, err := reportEnvelopeCanonicalMessageAndID(&report)
	require.NoError(t, err)
	require.Equal(t, "80b0f43ae215dd88a6e635de00207cd549c2492bb2086b22ceceda73a4de65f3", reportID)

	// pre_invalid_reencryption_proof payload + report_id goldens: the same
	// values are asserted by pre_invalid_proof_payload_matches_golden_vector
	// in orbis-rs reporting/v0/types.rs — regenerate both sides together.
	preInvalidPayload := preInvalidProofPayloadForTest(
		rustGoldenPreInvalidStatement().encode(),
		bytes.Repeat([]byte{42}, 64),
	)
	require.Equal(
		t,
		"000000f80000001f6f726269732d7072652d7265656e63727970742d726573706f6e73652d76310000000e736f757263656875622d746573740000000672696e672d310000000461616262000000403131313131313131313131313131313131313131313131313131313131313131313131313131313131313131313131313131313131313131313131313131313100000000000000070000000d7072652d726571756573742d31000000006553f10a0000000761636375736564000000086f626a6563742d310000000301020301000000030405060000000200000002070800000002090a000000020b0c0000000c656c67616d616c2f74657374000000402a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a",
		hex.EncodeToString(preInvalidPayload),
	)
	preInvalidReport := report
	preInvalidReport.ReportType = PreInvalidReencryptionProofReportType
	preInvalidReport.Payload = preInvalidPayload
	_, preInvalidReportID, err := reportEnvelopeCanonicalMessageAndID(&preInvalidReport)
	require.NoError(t, err)
	require.Equal(t, "29c88e561ffad3d440c862a0ee7b9092ade3f13469721677042504c6390541d9", preInvalidReportID)

	ring := &types.Ring{
		RingPk:       "pk",
		PeerNodeKeys: []string{"b", "a"},
		Threshold:    2,
		PssInterval:  types.MinPSSIntervalSeconds,
		UpgradeInfo: types.UpgradeInfo{
			CurrentVersion: 0,
		},
		Reporting: types.DefaultReportingConfig(),
	}
	ringHash, err := reportRingStateSHA256(ring)
	require.NoError(t, err)
	require.Equal(t, "f6561137d2827315c438a8e0608cdf86748e7e7d0aa4b741dedc065f536c7861", ringHash)
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

func TestPreInvalidProofPayloadDecodeRejectsMalformedPayloads(t *testing.T) {
	signature := bytes.Repeat([]byte{42}, 64)
	validStatement := rustGoldenPreInvalidStatement()
	valid := preInvalidProofPayloadForTest(validStatement.encode(), signature)

	decoded, err := decodePreInvalidReencryptionProofPayload(valid)
	require.NoError(t, err)
	require.Equal(t, reportTestChainID, decoded.chainID)
	require.Equal(t, "ring-1", decoded.ringID)
	require.Equal(t, "aabb", decoded.ringPk)
	require.Equal(t, strings.Repeat("11", 32), decoded.ringStateSha256)
	require.Equal(t, uint64(7), decoded.protocolVersion)
	require.Equal(t, "pre-request-1", decoded.requestID)
	require.Equal(t, reportTestObservedAt+reportObservedAtGraceSecs, decoded.signedAt)
	require.Equal(t, "accused", decoded.responderNodeKey)

	rejected := []struct {
		name    string
		payload []byte
	}{
		{"truncated outer", valid[:10]},
		{"trailing outer bytes", append(append([]byte{}, valid...), 0)},
		{"trailing inner bytes", preInvalidProofPayloadForTest(append(validStatement.encode(), 0), signature)},
		{"short signature", preInvalidProofPayloadForTest(validStatement.encode(), bytes.Repeat([]byte{42}, 63))},
		{"wrong domain", preInvalidProofPayloadForTest(validStatement.with(func(s *preInvalidProofStatementFields) { s.domain = "other" }).encode(), signature)},
		{"empty request_id", preInvalidProofPayloadForTest(validStatement.with(func(s *preInvalidProofStatementFields) { s.requestID = "" }).encode(), signature)},
		{"empty responder", preInvalidProofPayloadForTest(validStatement.with(func(s *preInvalidProofStatementFields) { s.responderKey = "" }).encode(), signature)},
		{"empty object_id", preInvalidProofPayloadForTest(validStatement.with(func(s *preInvalidProofStatementFields) { s.objectID = "" }).encode(), signature)},
		{"empty crypto backend", preInvalidProofPayloadForTest(validStatement.with(func(s *preInvalidProofStatementFields) { s.cryptoBackend = "" }).encode(), signature)},
		{"empty proof", preInvalidProofPayloadForTest(validStatement.with(func(s *preInvalidProofStatementFields) { s.proof = nil }).encode(), signature)},
		{"oversize share", preInvalidProofPayloadForTest(validStatement.with(func(s *preInvalidProofStatementFields) { s.share = make([]byte, preResponseMaxElementLen+1) }).encode(), signature)},
		{"oversize derivation", preInvalidProofPayloadForTest(validStatement.with(func(s *preInvalidProofStatementFields) { s.derivation = make([]byte, preResponseMaxDerivationLen+1) }).encode(), signature)},
		{"invalid derivation tag", preInvalidProofPayloadForTest(validStatement.with(func(s *preInvalidProofStatementFields) { tag := byte(2); s.derivationTagOverride = &tag }).encode(), signature)},
		{"non-utf8 string field", preInvalidProofPayloadForTest(validStatement.with(func(s *preInvalidProofStatementFields) { s.chainID = string([]byte{0xff, 0xfe}) }).encode(), signature)},
	}
	for _, tc := range rejected {
		t.Run(tc.name, func(t *testing.T) {
			_, err := decodePreInvalidReencryptionProofPayload(tc.payload)
			require.ErrorIs(t, err, types.ErrInvalidReport)
		})
	}
}

func TestValidatePreInvalidProofStatementBindingAndAnchor(t *testing.T) {
	report := &types.ReportEnvelope{
		ChainId:         "chain",
		RingId:          "ring",
		RingPk:          "pk",
		RingStateSha256: "digest",
		SessionId:       "req",
		AccusedNodeKey:  "accused",
		ObservedAt:      reportTestObservedAt,
	}
	base := preInvalidProofStatement{
		chainID:          "chain",
		ringID:           "ring",
		ringPk:           "pk",
		ringStateSha256:  "digest",
		requestID:        "req",
		signedAt:         reportTestObservedAt + reportObservedAtGraceSecs,
		responderNodeKey: "accused",
	}
	require.NoError(t, validatePreInvalidProofStatement(report, base))

	rejected := []struct {
		name   string
		mutate func(*preInvalidProofStatement)
	}{
		{"chain mismatch", func(s *preInvalidProofStatement) { s.chainID = "other" }},
		{"ring id mismatch", func(s *preInvalidProofStatement) { s.ringID = "other" }},
		{"ring pk mismatch", func(s *preInvalidProofStatement) { s.ringPk = "other" }},
		{"ring state mismatch", func(s *preInvalidProofStatement) { s.ringStateSha256 = "other" }},
		{"session mismatch", func(s *preInvalidProofStatement) { s.requestID = "other" }},
		{"responder mismatch", func(s *preInvalidProofStatement) { s.responderNodeKey = "other" }},
		{"anchor off by one late", func(s *preInvalidProofStatement) { s.signedAt++ }},
		{"anchor off by one early", func(s *preInvalidProofStatement) { s.signedAt-- }},
		{"signed_at below grace", func(s *preInvalidProofStatement) { s.signedAt = reportObservedAtGraceSecs - 1 }},
	}
	for _, tc := range rejected {
		t.Run(tc.name, func(t *testing.T) {
			statement := base
			tc.mutate(&statement)
			err := validatePreInvalidProofStatement(report, statement)
			require.ErrorIs(t, err, types.ErrInvalidReport)
		})
	}
}

func TestDemeritAmountForReportTypeRejectsInvalidInputs(t *testing.T) {
	_, err := DemeritAmountForReportType(nil, NodeOfflineReportType)
	require.ErrorIs(t, err, types.ErrRingNotFound)

	_, err = DemeritAmountForReportType(&types.Ring{
		Reporting: types.DefaultReportingConfig(),
	}, "unknown")
	require.ErrorIs(t, err, types.ErrInvalidReport)

	amount, err := DemeritAmountForReportType(&types.Ring{
		Reporting: types.ReportingConfig{
			DemeritConfig: types.DemeritConfig{
				NodeOfflineDemerits:     5,
				PreInvalidProofDemerits: types.DefaultPreInvalidProofDemerits,
				ResetIntervalSeconds:    types.DefaultDemeritResetIntervalSecs,
			},
			KickThreshold: types.DefaultReportingKickThreshold,
		},
	}, NodeOfflineReportType)
	require.NoError(t, err)
	require.Equal(t, uint64(5), amount)

	amount, err = DemeritAmountForReportType(&types.Ring{
		Reporting: types.ReportingConfig{
			DemeritConfig: types.DemeritConfig{
				NodeOfflineDemerits:     5,
				PreInvalidProofDemerits: 7,
				ResetIntervalSeconds:    types.DefaultDemeritResetIntervalSecs,
			},
			KickThreshold: types.DefaultReportingKickThreshold,
		},
	}, PreInvalidReencryptionProofReportType)
	require.NoError(t, err)
	require.Equal(t, uint64(7), amount)
}

func TestMsgServer_SubmitReport_PreInvalidProofAcceptsRetainsAndDedupes(t *testing.T) {
	fixture, sk := setupBLSReportFixture(t, "orbis-report-preproof-ikm-000", 2)

	report := fixture.validPreInvalidProofReport(t)
	msg := fixture.signBLSReport(t, sk, report)

	resp, err := fixture.k.SubmitReport(fixture.ctx, msg)
	require.NoError(t, err)
	require.Equal(t, msg.ReportId, resp.ReportId)
	require.True(t, fixture.k.HasAcceptedReport(fixture.ctx, msg.ReportId))
	require.Equal(t, uint64(1), fixture.k.GetNodeDemerits(fixture.ctx, fixture.ringID, fixture.accusedKey))

	events := parseTypedEvents(t, fixture.ctx)
	require.Contains(t, events, &types.EventReportAccepted{
		ReportId:        msg.ReportId,
		RingId:          fixture.ringID,
		ReportType:      PreInvalidReencryptionProofReportType,
		ReporterNodeKey: fixture.reporterKey,
		AccusedNodeKey:  fixture.accusedKey,
	})

	// Same report replayed: rejected by report-ID dedupe.
	_, err = fixture.k.SubmitReport(fixture.ctx, msg)
	require.ErrorIs(t, err, types.ErrReportAlreadyAccepted)

	// Same evidence re-wrapped by a different reporter: rejected by session dedupe.
	secondReport := fixture.validPreInvalidProofReport(t)
	secondReport.ReporterNodeKey = fixture.validatorKey
	second := fixture.signBLSReport(t, sk, secondReport)
	require.NotEqual(t, msg.ReportId, second.ReportId)
	_, err = fixture.k.SubmitReport(fixture.ctx, second)
	require.ErrorIs(t, err, types.ErrReportAlreadyAccepted)
	require.Equal(t, uint64(1), fixture.k.GetNodeDemerits(fixture.ctx, fixture.ringID, fixture.accusedKey))

	// Dedupe records live for the plain report TTL — the anchored envelope
	// (observed_at == signed_at - grace) guarantees no envelope over the same
	// evidence can still be valid once the record prunes.
	sessionID, err := reportSessionDedupeID(&msg.Report, reportPayload{
		originProtocol:        offlineOriginProtocolPRE,
		originProtocolVersion: 0,
		accusedCommitteeScope: committeeScopeCurrent,
		signingCommitteeScope: committeeScopeCurrent,
	})
	require.NoError(t, err)
	require.True(t, fixture.k.HasAcceptedReportSession(fixture.ctx, sessionID))

	fixture.k.DeleteExpiredAcceptedReportPairs(fixture.ctx, reportTestObservedAt+ReportTTLSeconds-1)
	require.True(t, fixture.k.HasAcceptedReport(fixture.ctx, msg.ReportId))
	require.True(t, fixture.k.HasAcceptedReportSession(fixture.ctx, sessionID))

	fixture.k.DeleteExpiredAcceptedReportPairs(fixture.ctx, reportTestObservedAt+ReportTTLSeconds)
	require.False(t, fixture.k.HasAcceptedReport(fixture.ctx, msg.ReportId))
	require.False(t, fixture.k.HasAcceptedReportSession(fixture.ctx, sessionID))
}

func TestMsgServer_SubmitReport_PreInvalidProofRejectsTamperedStatements(t *testing.T) {
	fixture, sk := setupBLSReportFixture(t, "orbis-report-preproof-bad-000", 2)

	t.Run("session_id not matching statement request_id", func(t *testing.T) {
		report := fixture.validPreInvalidProofReport(t)
		report.SessionId = "different-request"
		msg := fixture.signBLSReport(t, sk, report)
		_, err := fixture.k.SubmitReport(fixture.ctx, msg)
		require.ErrorIs(t, err, types.ErrInvalidReport)
	})

	t.Run("accused not matching statement responder", func(t *testing.T) {
		report := fixture.validPreInvalidProofReport(t)
		report.AccusedNodeKey = fixture.validatorKey
		report.AccusedPeerId = "12D3KooWReportValidator"
		msg := fixture.signBLSReport(t, sk, report)
		_, err := fixture.k.SubmitReport(fixture.ctx, msg)
		require.ErrorIs(t, err, types.ErrInvalidReport)
	})

	t.Run("envelope not anchored to evidence timestamp", func(t *testing.T) {
		report := fixture.validPreInvalidProofReport(t)
		report.Payload = preInvalidProofPayloadForTest(
			fixture.preInvalidProofStatementFields(t).with(func(s *preInvalidProofStatementFields) {
				s.signedAt++
			}).encode(),
			bytes.Repeat([]byte{42}, 64),
		)
		msg := fixture.signBLSReport(t, sk, report)
		_, err := fixture.k.SubmitReport(fixture.ctx, msg)
		require.ErrorIs(t, err, types.ErrInvalidReport)
	})
}

func TestMsgServer_SubmitReport_PreInvalidProofDecaf377FROSTAccepts(t *testing.T) {
	fixture := setupReportTestFixture(t)
	secretScalar := new(big.Int).SetBytes([]byte("orbis-preproof-decaf377-secret-k"))
	secretScalar.Mod(secretScalar, decaf377.ScalarOrder())

	ringPkBytes, err := decaf377PublicKeyBytes(secretScalar)
	require.NoError(t, err)
	fixture.setRing(t, hex.EncodeToString(ringPkBytes), 2)

	report := fixture.validPreInvalidProofReport(t)
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

func TestMsgServer_SubmitReportRejectsDuplicateSession(t *testing.T) {
	fixture := setupReportTestFixture(t)
	ikm := make([]byte, 32)
	copy(ikm, "orbis-report-session-ikm-0000")
	sk := blst.KeyGen(ikm)
	fixture.setRing(t, hex.EncodeToString(new(blst.P1Affine).From(sk).Compress()), 2)

	first := fixture.signBLSReport(t, sk, fixture.validReport(t, committeeScopeCurrent, committeeScopeCurrent, 0))
	firstPayload, err := decodeNodeOfflinePayload(first.Report.Payload)
	require.NoError(t, err)
	firstSessionID, err := reportSessionDedupeID(&first.Report, firstPayload)
	require.NoError(t, err)
	_, err = fixture.k.SubmitReport(fixture.ctx, first)
	require.NoError(t, err)

	secondReport := fixture.validReport(t, committeeScopeCurrent, committeeScopeCurrent, 0)
	secondReport.ReporterNodeKey = fixture.validatorKey
	second := fixture.signBLSReport(t, sk, secondReport)
	secondPayload, err := decodeNodeOfflinePayload(second.Report.Payload)
	require.NoError(t, err)
	secondSessionID, err := reportSessionDedupeID(&second.Report, secondPayload)
	require.NoError(t, err)
	require.NotEqual(t, first.ReportId, second.ReportId)
	require.Equal(t, firstSessionID, secondSessionID)

	_, err = fixture.k.SubmitReport(fixture.ctx, second)
	require.ErrorIs(t, err, types.ErrReportAlreadyAccepted)
	require.Equal(t, uint64(1), fixture.k.GetNodeDemerits(fixture.ctx, fixture.ringID, fixture.accusedKey))
	require.True(t, fixture.k.HasAcceptedReport(fixture.ctx, first.ReportId))
	require.False(t, fixture.k.HasAcceptedReport(fixture.ctx, second.ReportId))
	require.True(t, fixture.k.HasAcceptedReportSession(fixture.ctx, firstSessionID))
}

func TestMsgServer_SubmitReportIncrementsDemeritsForDistinctReports(t *testing.T) {
	fixture := setupReportTestFixture(t)
	ikm := make([]byte, 32)
	copy(ikm, "orbis-report-demerit-ikm-0000")
	sk := blst.KeyGen(ikm)
	fixture.setRingWithDemeritConfig(t, hex.EncodeToString(new(blst.P1Affine).From(sk).Compress()), 2, types.DemeritConfig{
		NodeOfflineDemerits:     3,
		PreInvalidProofDemerits: types.DefaultPreInvalidProofDemerits,
		ResetIntervalSeconds:    types.DefaultDemeritResetIntervalSecs,
	})

	first := fixture.signBLSReport(t, sk, fixture.validReport(t, committeeScopeCurrent, committeeScopeCurrent, 0))
	_, err := fixture.k.SubmitReport(fixture.ctx, first)
	require.NoError(t, err)
	require.Equal(t, uint64(3), fixture.k.GetNodeDemerits(fixture.ctx, fixture.ringID, fixture.accusedKey))

	secondReport := fixture.validReport(t, committeeScopeCurrent, committeeScopeCurrent, 0)
	secondReport.SessionId = "pre-request-2"
	second := fixture.signBLSReport(t, sk, secondReport)
	_, err = fixture.k.SubmitReport(fixture.ctx, second)
	require.NoError(t, err)
	require.Equal(t, uint64(6), fixture.k.GetNodeDemerits(fixture.ctx, fixture.ringID, fixture.accusedKey))
}

func TestMsgServer_SubmitReportSchedulesAutoReshareAtKickThreshold(t *testing.T) {
	fixture := setupReportTestFixture(t)
	ikm := make([]byte, 32)
	copy(ikm, "orbis-report-auto-kick-ikm")
	sk := blst.KeyGen(ikm)

	backup1Addr, backup1Key := setupPeerWithNodeInfo(t, fixture.k, fixture.authKeeper, fixture.ctx, "12D3KooWBackup1")
	backup2Addr, backup2Key := setupPeerWithNodeInfo(t, fixture.k, fixture.authKeeper, fixture.ctx, "12D3KooWBackup2")
	updatePeerNodeWhitelists(t, fixture.k, fixture.ctx, backup1Addr, backup1Key, []string{"report-policy"}, nil)
	updatePeerNodeWhitelists(t, fixture.k, fixture.ctx, backup2Addr, backup2Key, []string{"report-policy"}, nil)

	fixture.setRingWithReportingConfig(t, hex.EncodeToString(new(blst.P1Affine).From(sk).Compress()), 2, types.ReportingConfig{
		DemeritConfig: types.DemeritConfig{
			NodeOfflineDemerits:     3,
			PreInvalidProofDemerits: types.DefaultPreInvalidProofDemerits,
			ResetIntervalSeconds:    types.DefaultDemeritResetIntervalSecs,
		},
		BackupNodeKeys: []string{backup1Key, backup2Key},
		KickThreshold:  3,
	})

	msg := fixture.signBLSReport(t, sk, fixture.validReport(t, committeeScopeCurrent, committeeScopeCurrent, 0))
	_, err := fixture.k.SubmitReport(fixture.ctx, msg)
	require.NoError(t, err)

	ring := fixture.k.GetRing(fixture.ctx, fixture.ringID)
	require.NotNil(t, ring)
	require.Equal(t, canonicalNodeKeys([]string{fixture.reporterKey, fixture.validatorKey, backup1Key}), ring.NewPeerNodeKeys)
	require.Nil(t, ring.XNewThreshold)
	require.Equal(t, []string{backup2Key}, ring.Reporting.BackupNodeKeys)
	require.Equal(t, uint64(3), fixture.k.GetNodeDemerits(fixture.ctx, fixture.ringID, fixture.accusedKey))

	events := parseTypedEvents(t, fixture.ctx)
	require.Contains(t, events, &types.EventRingAutoReshareScheduled{
		RingId:             fixture.ringID,
		AccusedNodeKey:     fixture.accusedKey,
		ReplacementNodeKey: backup1Key,
		KickThreshold:      3,
		Demerits:           3,
	})
}

func TestMsgServer_SubmitReportDoesNotScheduleAutoReshareBelowThreshold(t *testing.T) {
	fixture := setupReportTestFixture(t)
	ikm := make([]byte, 32)
	copy(ikm, "orbis-report-below-kick")
	sk := blst.KeyGen(ikm)

	backupAddr, backupKey := setupPeerWithNodeInfo(t, fixture.k, fixture.authKeeper, fixture.ctx, "12D3KooWBackupBelow")
	updatePeerNodeWhitelists(t, fixture.k, fixture.ctx, backupAddr, backupKey, []string{"report-policy"}, nil)

	fixture.setRingWithReportingConfig(t, hex.EncodeToString(new(blst.P1Affine).From(sk).Compress()), 2, types.ReportingConfig{
		DemeritConfig: types.DemeritConfig{
			NodeOfflineDemerits:     1,
			PreInvalidProofDemerits: types.DefaultPreInvalidProofDemerits,
			ResetIntervalSeconds:    types.DefaultDemeritResetIntervalSecs,
		},
		BackupNodeKeys: []string{backupKey},
		KickThreshold:  2,
	})

	msg := fixture.signBLSReport(t, sk, fixture.validReport(t, committeeScopeCurrent, committeeScopeCurrent, 0))
	_, err := fixture.k.SubmitReport(fixture.ctx, msg)
	require.NoError(t, err)

	ring := fixture.k.GetRing(fixture.ctx, fixture.ringID)
	require.Empty(t, ring.NewPeerNodeKeys)
	require.Equal(t, []string{backupKey}, ring.Reporting.BackupNodeKeys)
}

func TestMsgServer_SubmitReportSuppressesAutoReshareWhenAlreadyPending(t *testing.T) {
	fixture := setupReportTestFixture(t)
	ikm := make([]byte, 32)
	copy(ikm, "orbis-report-pending-kick")
	sk := blst.KeyGen(ikm)

	backupAddr, backupKey := setupPeerWithNodeInfo(t, fixture.k, fixture.authKeeper, fixture.ctx, "12D3KooWBackupPending")
	updatePeerNodeWhitelists(t, fixture.k, fixture.ctx, backupAddr, backupKey, []string{"report-policy"}, nil)

	fixture.setRingWithReportingConfig(t, hex.EncodeToString(new(blst.P1Affine).From(sk).Compress()), 2, types.ReportingConfig{
		DemeritConfig: types.DemeritConfig{
			NodeOfflineDemerits:     3,
			PreInvalidProofDemerits: types.DefaultPreInvalidProofDemerits,
			ResetIntervalSeconds:    types.DefaultDemeritResetIntervalSecs,
		},
		BackupNodeKeys: []string{backupKey},
		KickThreshold:  3,
	})
	pendingRing := *fixture.originalRing
	pendingRing.NewPeerNodeKeys = canonicalNodeKeys([]string{fixture.reporterKey, fixture.validatorKey})
	fixture.replaceRing(t, pendingRing)

	msg := fixture.signBLSReport(t, sk, fixture.validReport(t, committeeScopeCurrent, committeeScopeCurrent, 0))
	_, err := fixture.k.SubmitReport(fixture.ctx, msg)
	require.NoError(t, err)

	ring := fixture.k.GetRing(fixture.ctx, fixture.ringID)
	require.Equal(t, pendingRing.NewPeerNodeKeys, ring.NewPeerNodeKeys)
	require.Equal(t, []string{backupKey}, ring.Reporting.BackupNodeKeys)
}

func TestMsgServer_SubmitReportSuppressesAutoReshareWithoutEligibleBackup(t *testing.T) {
	fixture := setupReportTestFixture(t)
	ikm := make([]byte, 32)
	copy(ikm, "orbis-report-no-backup")
	sk := blst.KeyGen(ikm)

	_, ineligibleBackupKey := setupPeerWithNodeInfo(t, fixture.k, fixture.authKeeper, fixture.ctx, "12D3KooWBackupNoWhitelist")

	fixture.setRingWithReportingConfig(t, hex.EncodeToString(new(blst.P1Affine).From(sk).Compress()), 2, types.ReportingConfig{
		DemeritConfig: types.DemeritConfig{
			NodeOfflineDemerits:     3,
			PreInvalidProofDemerits: types.DefaultPreInvalidProofDemerits,
			ResetIntervalSeconds:    types.DefaultDemeritResetIntervalSecs,
		},
		BackupNodeKeys: []string{fixture.reporterKey, ineligibleBackupKey},
		KickThreshold:  3,
	})

	msg := fixture.signBLSReport(t, sk, fixture.validReport(t, committeeScopeCurrent, committeeScopeCurrent, 0))
	_, err := fixture.k.SubmitReport(fixture.ctx, msg)
	require.NoError(t, err)

	ring := fixture.k.GetRing(fixture.ctx, fixture.ringID)
	require.Empty(t, ring.NewPeerNodeKeys)
	require.Equal(t, []string{fixture.reporterKey, ineligibleBackupKey}, ring.Reporting.BackupNodeKeys)
}

func TestMsgServer_SubmitReportAppliesLazyDemeritReset(t *testing.T) {
	fixture := setupReportTestFixture(t)
	ikm := make([]byte, 32)
	copy(ikm, "orbis-report-reset-ikm-00000")
	sk := blst.KeyGen(ikm)
	fixture.setRingWithDemeritConfig(t, hex.EncodeToString(new(blst.P1Affine).From(sk).Compress()), 2, types.DemeritConfig{
		NodeOfflineDemerits:     3,
		PreInvalidProofDemerits: types.DefaultPreInvalidProofDemerits,
		ResetIntervalSeconds:    ReportTTLSeconds * 2,
	})

	first := fixture.signBLSReport(t, sk, fixture.validReport(t, committeeScopeCurrent, committeeScopeCurrent, 0))
	_, err := fixture.k.SubmitReport(fixture.ctx, first)
	require.NoError(t, err)
	require.Equal(t, uint64(3), fixture.k.GetNodeDemerits(fixture.ctx, fixture.ringID, fixture.accusedKey))

	secondReport := fixture.validReport(t, committeeScopeCurrent, committeeScopeCurrent, 0)
	secondReport.ObservedAt += ReportTTLSeconds
	secondReport.ExpiresAt += ReportTTLSeconds
	secondReport.SessionId = "pre-request-2"
	second := fixture.signBLSReport(t, sk, secondReport)
	withinWindowCtx := fixture.ctx.WithBlockTime(time.Unix(int64(reportTestObservedAt+ReportTTLSeconds), 0))
	_, err = fixture.k.SubmitReport(withinWindowCtx, second)
	require.NoError(t, err)
	require.Equal(t, uint64(6), fixture.k.GetNodeDemerits(fixture.ctx, fixture.ringID, fixture.accusedKey))

	thirdReport := fixture.validReport(t, committeeScopeCurrent, committeeScopeCurrent, 0)
	thirdReport.ObservedAt += ReportTTLSeconds * 2
	thirdReport.ExpiresAt += ReportTTLSeconds * 2
	thirdReport.SessionId = "pre-request-3"
	third := fixture.signBLSReport(t, sk, thirdReport)
	resetBoundaryCtx := fixture.ctx.WithBlockTime(time.Unix(int64(reportTestObservedAt+ReportTTLSeconds*2), 0))
	_, err = fixture.k.SubmitReport(resetBoundaryCtx, third)
	require.NoError(t, err)
	require.Equal(t, uint64(3), fixture.k.GetNodeDemerits(fixture.ctx, fixture.ringID, fixture.accusedKey))
}

func TestMsgServer_SubmitReportDoesNotResetDemeritsOnReplayOrRejectedReport(t *testing.T) {
	fixture := setupReportTestFixture(t)
	ikm := make([]byte, 32)
	copy(ikm, "orbis-report-no-reset-ikm-00")
	sk := blst.KeyGen(ikm)
	fixture.setRingWithDemeritConfig(t, hex.EncodeToString(new(blst.P1Affine).From(sk).Compress()), 2, types.DemeritConfig{
		NodeOfflineDemerits:     3,
		PreInvalidProofDemerits: types.DefaultPreInvalidProofDemerits,
		ResetIntervalSeconds:    10,
	})

	accepted := fixture.signBLSReport(t, sk, fixture.validReport(t, committeeScopeCurrent, committeeScopeCurrent, 0))
	_, err := fixture.k.SubmitReport(fixture.ctx, accepted)
	require.NoError(t, err)
	require.Equal(t, uint64(3), fixture.k.GetNodeDemerits(fixture.ctx, fixture.ringID, fixture.accusedKey))

	expiredWindowCtx := fixture.ctx.WithBlockTime(time.Unix(int64(reportTestObservedAt+10), 0))
	_, err = fixture.k.SubmitReport(expiredWindowCtx, accepted)
	require.ErrorIs(t, err, types.ErrReportAlreadyAccepted)
	require.Equal(t, uint64(3), fixture.k.GetNodeDemerits(fixture.ctx, fixture.ringID, fixture.accusedKey))

	rejectedReport := fixture.validReport(t, committeeScopeCurrent, committeeScopeCurrent, 0)
	rejectedReport.ObservedAt++
	rejectedReport.ExpiresAt++
	rejectedReport.Domain = "wrong"
	_, err = fixture.k.SubmitReport(expiredWindowCtx, fixture.signBLSReport(t, sk, rejectedReport))
	require.ErrorIs(t, err, types.ErrInvalidReport)
	require.Equal(t, uint64(3), fixture.k.GetNodeDemerits(fixture.ctx, fixture.ringID, fixture.accusedKey))
}

func TestQueryServer_NodeDemeritsReturnsEffectiveScoreWithoutResettingExpiredWindow(t *testing.T) {
	fixture := setupReportTestFixture(t)
	ikm := make([]byte, 32)
	copy(ikm, "orbis-report-query-reset-ikm")
	sk := blst.KeyGen(ikm)
	fixture.setRingWithDemeritConfig(t, hex.EncodeToString(new(blst.P1Affine).From(sk).Compress()), 2, types.DemeritConfig{
		NodeOfflineDemerits:     3,
		PreInvalidProofDemerits: types.DefaultPreInvalidProofDemerits,
		ResetIntervalSeconds:    10,
	})

	accepted := fixture.signBLSReport(t, sk, fixture.validReport(t, committeeScopeCurrent, committeeScopeCurrent, 0))
	_, err := fixture.k.SubmitReport(fixture.ctx, accepted)
	require.NoError(t, err)

	resp, err := fixture.k.NodeDemerits(fixture.ctx, &types.QueryNodeDemeritsRequest{
		RingId:  fixture.ringID,
		NodeKey: fixture.accusedKey,
	})
	require.NoError(t, err)
	require.Equal(t, uint64(3), resp.Points)

	expiredWindowCtx := fixture.ctx.WithBlockTime(time.Unix(int64(reportTestObservedAt+10), 0))
	resp, err = fixture.k.NodeDemerits(expiredWindowCtx, &types.QueryNodeDemeritsRequest{
		RingId:  fixture.ringID,
		NodeKey: fixture.accusedKey,
	})
	require.NoError(t, err)
	require.Equal(t, uint64(0), resp.Points)
	require.Equal(t, uint64(3), fixture.k.GetNodeDemerits(fixture.ctx, fixture.ringID, fixture.accusedKey))

	state, found := fixture.k.getNodeDemeritState(fixture.ctx, fixture.ringID, fixture.accusedKey)
	require.True(t, found)
	require.Equal(t, reportTestObservedAt, state.windowStartedAt)

	nextReport := fixture.validReport(t, committeeScopeCurrent, committeeScopeCurrent, 0)
	nextReport.ObservedAt += ReportTTLSeconds
	nextReport.ExpiresAt += ReportTTLSeconds
	nextReport.SessionId = "pre-request-2"
	nextReportCtx := fixture.ctx.WithBlockTime(time.Unix(int64(reportTestObservedAt+ReportTTLSeconds), 0))
	_, err = fixture.k.SubmitReport(nextReportCtx, fixture.signBLSReport(t, sk, nextReport))
	require.NoError(t, err)
	require.Equal(t, uint64(3), fixture.k.GetNodeDemerits(fixture.ctx, fixture.ringID, fixture.accusedKey))

	state, found = fixture.k.getNodeDemeritState(fixture.ctx, fixture.ringID, fixture.accusedKey)
	require.True(t, found)
	require.Equal(t, reportTestObservedAt+ReportTTLSeconds, state.windowStartedAt)
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

func TestMsgServer_SubmitReportValidatesSigningCommitteeCapacity(t *testing.T) {
	t.Run("current signing threshold must be at least two", func(t *testing.T) {
		fixture, sk := setupBLSReportFixture(t, "orbis-report-capacity-low", 1)
		msg := fixture.signBLSReport(t, sk, fixture.validReport(t, committeeScopeCurrent, committeeScopeCurrent, 0))

		_, err := fixture.k.SubmitReport(fixture.ctx, msg)
		require.ErrorIs(t, err, types.ErrInvalidReport)
		require.ErrorContains(t, err, "threshold >= 2")
	})

	t.Run("current signing threshold cannot exceed committee size", func(t *testing.T) {
		fixture, sk := setupBLSReportFixture(t, "orbis-report-capacity-big", 4)
		msg := fixture.signBLSReport(t, sk, fixture.validReport(t, committeeScopeCurrent, committeeScopeCurrent, 0))

		_, err := fixture.k.SubmitReport(fixture.ctx, msg)
		require.ErrorIs(t, err, types.ErrInvalidReport)
		require.ErrorContains(t, err, "threshold exceeds signing committee size")
	})

	t.Run("current signing threshold must be reachable without accused", func(t *testing.T) {
		fixture, sk := setupBLSReportFixture(t, "orbis-report-capacity-exc", 3)
		msg := fixture.signBLSReport(t, sk, fixture.validReport(t, committeeScopeCurrent, committeeScopeCurrent, 0))

		_, err := fixture.k.SubmitReport(fixture.ctx, msg)
		require.ErrorIs(t, err, types.ErrInvalidReport)
		require.ErrorContains(t, err, "cannot be met while excluding the accused")
	})

	t.Run("pending-new signing threshold cannot exceed pending committee size", func(t *testing.T) {
		fixture, sk := setupBLSReportFixture(t, "orbis-report-capacity-pbig", 2)
		ring := *fixture.originalRing
		ring.NewPeerNodeKeys = []string{fixture.reporterKey, fixture.validatorKey}
		ring.XNewThreshold = &types.Ring_NewThreshold{NewThreshold: 3}
		fixture.replaceRing(t, ring)
		msg := fixture.signBLSReport(t, sk, fixture.validReport(t, committeeScopeCurrent, committeeScopePendingNew, 0))

		_, err := fixture.k.SubmitReport(fixture.ctx, msg)
		require.ErrorIs(t, err, types.ErrInvalidReport)
		require.ErrorContains(t, err, "threshold exceeds signing committee size")
	})

	t.Run("pending-new signing threshold must be reachable without pending accused", func(t *testing.T) {
		fixture, sk := setupBLSReportFixture(t, "orbis-report-capacity-pexc", 2)
		pendingPeerID := "12D3KooWReportPendingCapacity"
		_, pendingKey := setupPeerWithNodeInfo(t, fixture.k, fixture.authKeeper, fixture.ctx, pendingPeerID)
		ring := *fixture.originalRing
		ring.NewPeerNodeKeys = []string{fixture.reporterKey, pendingKey}
		ring.XNewThreshold = &types.Ring_NewThreshold{NewThreshold: 2}
		fixture.replaceRing(t, ring)
		report := fixture.validReport(t, committeeScopePendingNew, committeeScopePendingNew, 0)
		report.AccusedNodeKey = pendingKey
		report.AccusedPeerId = pendingPeerID
		msg := fixture.signBLSReport(t, sk, report)

		_, err := fixture.k.SubmitReport(fixture.ctx, msg)
		require.ErrorIs(t, err, types.ErrInvalidReport)
		require.ErrorContains(t, err, "cannot be met while excluding the accused")
	})

	t.Run("full pending-new threshold is allowed when accused is outside signing committee", func(t *testing.T) {
		fixture, sk := setupBLSReportFixture(t, "orbis-report-capacity-ok", 2)
		ring := *fixture.originalRing
		ring.NewPeerNodeKeys = []string{fixture.reporterKey, fixture.validatorKey}
		ring.XNewThreshold = &types.Ring_NewThreshold{NewThreshold: 2}
		fixture.replaceRing(t, ring)
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

func setupBLSReportFixture(t *testing.T, ikmLabel string, threshold uint32) (reportTestFixture, *blst.SecretKey) {
	t.Helper()
	fixture := setupReportTestFixture(t)
	ikm := make([]byte, 32)
	copy(ikm, ikmLabel)
	sk := blst.KeyGen(ikm)
	fixture.setRing(t, hex.EncodeToString(new(blst.P1Affine).From(sk).Compress()), threshold)
	return fixture, sk
}

func (f *reportTestFixture) setRing(t *testing.T, ringPk string, threshold uint32) {
	f.setRingWithDemeritConfig(t, ringPk, threshold, types.DefaultDemeritConfig())
}

func (f *reportTestFixture) setRingWithDemeritConfig(t *testing.T, ringPk string, threshold uint32, demeritConfig types.DemeritConfig) {
	f.setRingWithReportingConfig(t, ringPk, threshold, types.ReportingConfig{
		DemeritConfig: demeritConfig,
		KickThreshold: types.DefaultReportingKickThreshold,
	})
}

func (f *reportTestFixture) setRingWithReportingConfig(t *testing.T, ringPk string, threshold uint32, reporting types.ReportingConfig) {
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
		Reporting: reporting,
	}
	f.k.SetRing(f.ctx, ring)
	stored := f.k.GetRing(f.ctx, f.ringID)
	require.NotNil(t, stored)
	f.originalRing = stored
}

func (f *reportTestFixture) replaceRing(t *testing.T, ring types.Ring) {
	t.Helper()
	f.k.SetRing(f.ctx, ring)
	stored := f.k.GetRing(f.ctx, ring.Id)
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
		SessionId:       "pre-request-1",
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

// preInvalidProofStatementFields mirrors the orbis-rs PreReencryptResponseStatement
// canonical encoding for tests. derivationTagOverride forces an invalid optional
// tag byte for decoder rejection cases.
type preInvalidProofStatementFields struct {
	domain                string
	chainID               string
	ringID                string
	ringPk                string
	ringStateSha256       string
	protocolVersion       uint64
	requestID             string
	signedAt              uint64
	responderKey          string
	objectID              string
	rdrPk                 []byte
	derivation            []byte
	derivationTagOverride *byte
	fromNodeID            uint32
	share                 []byte
	challenge             []byte
	proof                 []byte
	cryptoBackend         string
}

func (s preInvalidProofStatementFields) with(mutate func(*preInvalidProofStatementFields)) preInvalidProofStatementFields {
	mutate(&s)
	return s
}

func (s preInvalidProofStatementFields) encode() []byte {
	w := newReportCanonicalWriter()
	w.writeString(s.domain)
	w.writeString(s.chainID)
	w.writeString(s.ringID)
	w.writeString(s.ringPk)
	w.writeString(s.ringStateSha256)
	w.writeU64(s.protocolVersion)
	w.writeString(s.requestID)
	w.writeU64(s.signedAt)
	w.writeString(s.responderKey)
	w.writeString(s.objectID)
	w.writeBytes(s.rdrPk)
	switch {
	case s.derivationTagOverride != nil:
		w.bytes = append(w.bytes, *s.derivationTagOverride)
	case s.derivation == nil:
		w.bytes = append(w.bytes, 0)
	default:
		w.bytes = append(w.bytes, 1)
		w.writeBytes(s.derivation)
	}
	w.writeU32(s.fromNodeID)
	w.writeBytes(s.share)
	w.writeBytes(s.challenge)
	w.writeBytes(s.proof)
	w.writeString(s.cryptoBackend)
	statement, err := w.finish()
	if err != nil {
		panic(err)
	}
	return statement
}

func preInvalidProofPayloadForTest(statement []byte, signature []byte) []byte {
	w := newReportCanonicalWriter()
	w.writeBytes(statement)
	w.writeBytes(signature)
	payload, err := w.finish()
	if err != nil {
		panic(err)
	}
	return payload
}

// rustGoldenPreInvalidStatement matches the pre_statement() fixture in
// orbis-rs reporting/v0/types.rs so the golden vectors line up byte-for-byte.
func rustGoldenPreInvalidStatement() preInvalidProofStatementFields {
	return preInvalidProofStatementFields{
		domain:          PreReencryptResponseDomain,
		chainID:         reportTestChainID,
		ringID:          "ring-1",
		ringPk:          "aabb",
		ringStateSha256: strings.Repeat("11", 32),
		protocolVersion: 7,
		requestID:       "pre-request-1",
		signedAt:        reportTestObservedAt + reportObservedAtGraceSecs,
		responderKey:    "accused",
		objectID:        "object-1",
		rdrPk:           []byte{1, 2, 3},
		derivation:      []byte{4, 5, 6},
		fromNodeID:      2,
		share:           []byte{7, 8},
		challenge:       []byte{9, 10},
		proof:           []byte{11, 12},
		cryptoBackend:   "elgamal/test",
	}
}

// preInvalidProofStatementFields builds a statement consistent with the
// fixture's ring and a validPreInvalidProofReport envelope.
func (f reportTestFixture) preInvalidProofStatementFields(t *testing.T) preInvalidProofStatementFields {
	t.Helper()
	require.NotNil(t, f.originalRing)
	ringDigest, err := reportRingStateSHA256(f.originalRing)
	require.NoError(t, err)
	return preInvalidProofStatementFields{
		domain:          PreReencryptResponseDomain,
		chainID:         f.ctx.ChainID(),
		ringID:          f.ringID,
		ringPk:          f.originalRing.RingPk,
		ringStateSha256: ringDigest,
		protocolVersion: 0,
		requestID:       "pre-request-1",
		signedAt:        reportTestObservedAt + reportObservedAtGraceSecs,
		responderKey:    f.accusedKey,
		objectID:        "object-1",
		rdrPk:           []byte{1, 2, 3},
		derivation:      nil,
		fromNodeID:      2,
		share:           []byte{7, 8},
		challenge:       []byte{9, 10},
		proof:           []byte{11, 12},
		cryptoBackend:   "elgamal/test",
	}
}

func (f reportTestFixture) validPreInvalidProofReport(t *testing.T) types.ReportEnvelope {
	t.Helper()
	report := f.validReport(t, committeeScopeCurrent, committeeScopeCurrent, 0)
	report.ReportType = PreInvalidReencryptionProofReportType
	report.Payload = preInvalidProofPayloadForTest(
		f.preInvalidProofStatementFields(t).encode(),
		bytes.Repeat([]byte{42}, 64),
	)
	return report
}

func reportIDForTest(t *testing.T, report *types.ReportEnvelope) string {
	t.Helper()
	message, err := reportEnvelopeCanonicalBytes(report)
	require.NoError(t, err)
	sum := sha256.Sum256(message)
	return hex.EncodeToString(sum[:])
}
