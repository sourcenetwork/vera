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

	// invalid_crypto_response payload + report_id goldens: the same
	// values are asserted by invalid_crypto_response_payload_matches_golden_vector
	// in orbis-rs reporting/v0/types.rs — regenerate both sides together.
	preInvalidPayload := preInvalidProofPayloadForTest(
		rustGoldenPreInvalidStatement().encode(),
		bytes.Repeat([]byte{42}, 64),
	)
	require.Equal(
		t,
		"00000003707265000000ff0000001f6f726269732d7072652d7265656e63727970742d726573706f6e73652d76310000000e736f757263656875622d746573740000000672696e672d310000000461616262000000403131313131313131313131313131313131313131313131313131313131313131313131313131313131313131313131313131313131313131313131313131313100000000000000070000000d7072652d726571756573742d31000000006553f10a000000076163637573656400000003707265000000086f626a6563742d310000000301020301000000030405060000000200000002070800000002090a000000020b0c0000000c656c67616d616c2f74657374000000402a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a",
		hex.EncodeToString(preInvalidPayload),
	)
	preInvalidReport := report
	preInvalidReport.ReportType = InvalidCryptoResponseReportType
	preInvalidReport.Payload = preInvalidPayload
	_, preInvalidReportID, err := reportEnvelopeCanonicalMessageAndID(&preInvalidReport)
	require.NoError(t, err)
	require.Equal(t, "65450d9adeeb115d5985655b0d850af988af3cf7dd5e60d0c9d55df091a983e2", preInvalidReportID)

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
	require.Equal(t, "ad10dfb463d1d3aeca03644e200aa192c49d6689300e90fe243d710e645aba18", ringHash)
}

func TestReportRingStateSHA256IncludesUnauthorizedRequestDemerits(t *testing.T) {
	baseRing := func(unauthorizedRequestDemerits uint64) *types.Ring {
		reporting := types.DefaultReportingConfig()
		reporting.DemeritConfig.UnauthorizedRequestDemerits = unauthorizedRequestDemerits
		return &types.Ring{
			RingPk:       "pk",
			PeerNodeKeys: []string{"b", "a"},
			Threshold:    2,
			PssInterval:  types.MinPSSIntervalSeconds,
			UpgradeInfo: types.UpgradeInfo{
				CurrentVersion: 0,
			},
			Reporting: reporting,
		}
	}

	hashA, err := reportRingStateSHA256(baseRing(1))
	require.NoError(t, err)
	hashB, err := reportRingStateSHA256(baseRing(2))
	require.NoError(t, err)
	require.NotEqual(t, hashA, hashB)
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

func TestInvalidCryptoResponsePayloadDecodeRejectsMalformedPayloads(t *testing.T) {
	signature := bytes.Repeat([]byte{42}, 64)
	validStatement := rustGoldenPreInvalidStatement()
	valid := preInvalidProofPayloadForTest(validStatement.encode(), signature)

	decoded, err := decodeInvalidCryptoResponsePayload(valid)
	require.NoError(t, err)
	require.Equal(t, reportTestChainID, decoded.chainID)
	require.Equal(t, "ring-1", decoded.ringID)
	require.Equal(t, "aabb", decoded.ringPk)
	require.Equal(t, strings.Repeat("11", 32), decoded.ringStateSha256)
	require.Equal(t, uint64(7), decoded.protocolVersion)
	require.Equal(t, "pre-request-1", decoded.requestID)
	require.Equal(t, reportTestObservedAt+reportObservedAtGraceSecs, decoded.signedAt)
	require.Equal(t, "accused", decoded.responderNodeKey)
	require.Equal(t, offlineOriginProtocolPRE, decoded.originProtocol)
	require.Equal(t, committeeScopeCurrent, decoded.accusedCommitteeScope)
	require.Equal(t, committeeScopeCurrent, decoded.signingCommitteeScope)

	rejected := []struct {
		name    string
		payload []byte
	}{
		{"truncated outer", valid[:10]},
		{"trailing outer bytes", append(append([]byte{}, valid...), 0)},
		{"unsupported evidence kind", invalidCryptoResponsePayloadForTest("other", validStatement.encode(), signature)},
		{"trailing inner bytes", preInvalidProofPayloadForTest(append(validStatement.encode(), 0), signature)},
		{"short signature", preInvalidProofPayloadForTest(validStatement.encode(), bytes.Repeat([]byte{42}, 63))},
		{"wrong domain", preInvalidProofPayloadForTest(validStatement.with(func(s *preInvalidProofStatementFields) { s.domain = "other" }).encode(), signature)},
		{"wrong origin protocol", preInvalidProofPayloadForTest(validStatement.with(func(s *preInvalidProofStatementFields) { s.originProtocol = offlineOriginProtocolSign }).encode(), signature)},
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
			_, err := decodeInvalidCryptoResponsePayload(tc.payload)
			require.ErrorIs(t, err, types.ErrInvalidReport)
		})
	}
}

func TestInvalidCryptoResponseSignPayloadDecodeNormalizesMetadata(t *testing.T) {
	signature := bytes.Repeat([]byte{42}, 64)
	validStatement := signResponseStatementFields{
		domain:                SignResponseDomain,
		chainID:               reportTestChainID,
		ringID:                "ring-1",
		ringPk:                "aabb",
		ringStateSha256:       strings.Repeat("11", 32),
		protocolVersion:       7,
		requestID:             "sign-request-1",
		signedAt:              reportTestObservedAt + reportObservedAtGraceSecs,
		responderKey:          "accused",
		originProtocol:        offlineOriginProtocolSign,
		accusedCommitteeScope: committeeScopeCurrent,
		signingCommitteeScope: committeeScopeCurrent,
		fromNodeID:            2,
		message:               []byte{1, 2, 3},
		signingCommitments:    []byte{4, 5, 6},
		derivation:            []byte{7, 8},
		metadata:              []byte{9, 10},
		sigShare:              []byte{11, 12},
		cryptoBackend:         "sign/test",
	}

	decoded, err := decodeInvalidCryptoResponsePayload(signResponsePayloadForTest(validStatement.encode(), signature))
	require.NoError(t, err)
	require.Equal(t, reportTestChainID, decoded.chainID)
	require.Equal(t, "ring-1", decoded.ringID)
	require.Equal(t, "aabb", decoded.ringPk)
	require.Equal(t, strings.Repeat("11", 32), decoded.ringStateSha256)
	require.Equal(t, uint64(7), decoded.protocolVersion)
	require.Equal(t, "sign-request-1", decoded.requestID)
	require.Equal(t, reportTestObservedAt+reportObservedAtGraceSecs, decoded.signedAt)
	require.Equal(t, "accused", decoded.responderNodeKey)
	require.Equal(t, offlineOriginProtocolSign, decoded.originProtocol)
	require.Equal(t, committeeScopeCurrent, decoded.accusedCommitteeScope)
	require.Equal(t, committeeScopeCurrent, decoded.signingCommitteeScope)

	rejected := []struct {
		name    string
		payload []byte
	}{
		{"wrong domain", signResponsePayloadForTest(validStatement.with(func(s *signResponseStatementFields) { s.domain = "other" }).encode(), signature)},
		{"empty request_id", signResponsePayloadForTest(validStatement.with(func(s *signResponseStatementFields) { s.requestID = "" }).encode(), signature)},
		{"empty responder", signResponsePayloadForTest(validStatement.with(func(s *signResponseStatementFields) { s.responderKey = "" }).encode(), signature)},
		{"unknown origin protocol", signResponsePayloadForTest(validStatement.with(func(s *signResponseStatementFields) { s.originProtocol = "policy" }).encode(), signature)},
		// "pre" is only valid for the dedicated PRE evidence kind, never as a Sign
		// statement origin — matching the node-side sign-origin allowlist.
		{"pre origin rejected for sign", signResponsePayloadForTest(validStatement.with(func(s *signResponseStatementFields) { s.originProtocol = offlineOriginProtocolPRE }).encode(), signature)},
		{"unknown accused scope", signResponsePayloadForTest(validStatement.with(func(s *signResponseStatementFields) { s.accusedCommitteeScope = 99 }).encode(), signature)},
		{"unknown signing scope", signResponsePayloadForTest(validStatement.with(func(s *signResponseStatementFields) { s.signingCommitteeScope = 99 }).encode(), signature)},
		{"empty message", signResponsePayloadForTest(validStatement.with(func(s *signResponseStatementFields) { s.message = nil }).encode(), signature)},
		{"empty sig_share", signResponsePayloadForTest(validStatement.with(func(s *signResponseStatementFields) { s.sigShare = nil }).encode(), signature)},
		{"empty crypto backend", signResponsePayloadForTest(validStatement.with(func(s *signResponseStatementFields) { s.cryptoBackend = "" }).encode(), signature)},
		{"oversize message", signResponsePayloadForTest(validStatement.with(func(s *signResponseStatementFields) { s.message = make([]byte, signResponseMaxMessageLen+1) }).encode(), signature)},
		{"oversize commitments", signResponsePayloadForTest(validStatement.with(func(s *signResponseStatementFields) {
			s.signingCommitments = make([]byte, signResponseMaxCommitmentsLen+1)
		}).encode(), signature)},
		{"oversize sig_share", signResponsePayloadForTest(validStatement.with(func(s *signResponseStatementFields) { s.sigShare = make([]byte, signResponseMaxElementLen+1) }).encode(), signature)},
	}
	for _, tc := range rejected {
		t.Run(tc.name, func(t *testing.T) {
			_, err := decodeInvalidCryptoResponsePayload(tc.payload)
			require.ErrorIs(t, err, types.ErrInvalidReport)
		})
	}
}

func TestInvalidCryptoResponseDkgSharePayloadDecodeBindsNestedCommitment(t *testing.T) {
	signature := bytes.Repeat([]byte{42}, 64)
	validStatement := dkgShareStatementForTest()

	decoded, err := decodeInvalidCryptoResponsePayload(dkgSharePayloadForTest(validStatement.encode(), signature))
	require.NoError(t, err)
	require.Equal(t, reportTestChainID, decoded.chainID)
	require.Equal(t, "ring-1", decoded.ringID)
	require.Equal(t, "aabb", decoded.ringPk)
	require.Equal(t, strings.Repeat("11", 32), decoded.ringStateSha256)
	require.Equal(t, uint64(7), decoded.protocolVersion)
	require.Equal(t, "dkg-session-1", decoded.requestID)
	require.Equal(t, reportTestObservedAt+reportObservedAtGraceSecs, decoded.signedAt)
	require.Equal(t, "accused", decoded.responderNodeKey)
	require.Equal(t, offlineOriginProtocolPSSRefresh, decoded.originProtocol)
	require.Equal(t, committeeScopeCurrent, decoded.accusedCommitteeScope)
	require.Equal(t, committeeScopeCurrent, decoded.signingCommitteeScope)

	reshareStatement := validStatement.with(func(s *dkgShareStatementFields) {
		s.originProtocol = offlineOriginProtocolPSSReshare
		s.commitmentStatement.originProtocol = offlineOriginProtocolPSSReshare
	})
	decoded, err = decodeInvalidCryptoResponsePayload(dkgSharePayloadForTest(reshareStatement.encode(), signature))
	require.NoError(t, err)
	require.Equal(t, offlineOriginProtocolPSSReshare, decoded.originProtocol)

	rejected := []struct {
		name    string
		payload []byte
	}{
		{"wrong domain", dkgSharePayloadForTest(validStatement.with(func(s *dkgShareStatementFields) { s.domain = "other" }).encode(), signature)},
		{"empty request_id", dkgSharePayloadForTest(validStatement.with(func(s *dkgShareStatementFields) {
			s.requestID = ""
			s.commitmentStatement.requestID = ""
		}).encode(), signature)},
		{"empty responder", dkgSharePayloadForTest(validStatement.with(func(s *dkgShareStatementFields) {
			s.responderKey = ""
			s.commitmentStatement.responderKey = ""
		}).encode(), signature)},
		{"empty receiver", dkgSharePayloadForTest(validStatement.with(func(s *dkgShareStatementFields) { s.receiverKey = "" }).encode(), signature)},
		{"unsupported origin", dkgSharePayloadForTest(validStatement.with(func(s *dkgShareStatementFields) {
			s.originProtocol = offlineOriginProtocolSign
			s.commitmentStatement.originProtocol = offlineOriginProtocolSign
		}).encode(), signature)},
		{"unknown accused scope", dkgSharePayloadForTest(validStatement.with(func(s *dkgShareStatementFields) { s.accusedCommitteeScope = 99 }).encode(), signature)},
		{"pending accused scope", dkgSharePayloadForTest(validStatement.with(func(s *dkgShareStatementFields) {
			s.accusedCommitteeScope = committeeScopePendingNew
			s.commitmentStatement.accusedCommitteeScope = committeeScopePendingNew
		}).encode(), signature)},
		{"unknown signing scope", dkgSharePayloadForTest(validStatement.with(func(s *dkgShareStatementFields) { s.signingCommitteeScope = 99 }).encode(), signature)},
		{"from node zero", dkgSharePayloadForTest(validStatement.with(func(s *dkgShareStatementFields) {
			s.fromNodeID = 0
			s.commitmentStatement.fromNodeID = 0
		}).encode(), signature)},
		{"to node zero", dkgSharePayloadForTest(validStatement.with(func(s *dkgShareStatementFields) { s.toNodeID = 0 }).encode(), signature)},
		{"short commitment signature", dkgSharePayloadForTest(validStatement.with(func(s *dkgShareStatementFields) { s.commitmentSignature = bytes.Repeat([]byte{1}, 63) }).encode(), signature)},
		{"empty share value", dkgSharePayloadForTest(validStatement.with(func(s *dkgShareStatementFields) { s.shareValue = nil }).encode(), signature)},
		{"oversize share value", dkgSharePayloadForTest(validStatement.with(func(s *dkgShareStatementFields) { s.shareValue = make([]byte, dkgShareMaxElementLen+1) }).encode(), signature)},
		{"bad nonce length", dkgSharePayloadForTest(validStatement.with(func(s *dkgShareStatementFields) { s.nonce = bytes.Repeat([]byte{3}, dkgNonceLen-1) }).encode(), signature)},
		{"empty crypto backend", dkgSharePayloadForTest(validStatement.with(func(s *dkgShareStatementFields) {
			s.cryptoBackend = ""
			s.commitmentStatement.cryptoBackend = ""
		}).encode(), signature)},
		{"nested wrong domain", dkgSharePayloadForTest(validStatement.with(func(s *dkgShareStatementFields) { s.commitmentStatement.domain = "other" }).encode(), signature)},
		{"nested ring mismatch", dkgSharePayloadForTest(validStatement.with(func(s *dkgShareStatementFields) { s.commitmentStatement.ringID = "other" }).encode(), signature)},
		{"nested signed after share", dkgSharePayloadForTest(validStatement.with(func(s *dkgShareStatementFields) { s.commitmentStatement.signedAt = s.signedAt + 1 }).encode(), signature)},
		{"empty nested commitment", dkgSharePayloadForTest(validStatement.with(func(s *dkgShareStatementFields) { s.commitmentStatement.commitment = nil }).encode(), signature)},
		{"oversize nested commitment", dkgSharePayloadForTest(validStatement.with(func(s *dkgShareStatementFields) {
			s.commitmentStatement.commitment = make([]byte, dkgCommitmentMaxLen+1)
		}).encode(), signature)},
		{"trailing inner bytes", dkgSharePayloadForTest(append(validStatement.encode(), 0), signature)},
	}
	for _, tc := range rejected {
		t.Run(tc.name, func(t *testing.T) {
			_, err := decodeInvalidCryptoResponsePayload(tc.payload)
			require.ErrorIs(t, err, types.ErrInvalidReport)
		})
	}
}

func TestInvalidCryptoResponseDkgInvalidRefreshCommitmentPayloadDecodeBindsCommitment(t *testing.T) {
	signature := bytes.Repeat([]byte{42}, 64)
	validStatement, _ := dkgEquivocationCommitmentsForTest()

	decoded, err := decodeInvalidCryptoResponsePayload(dkgInvalidRefreshCommitmentPayloadForTest(validStatement.encode(), signature))
	require.NoError(t, err)
	require.Equal(t, reportTestChainID, decoded.chainID)
	require.Equal(t, "ring-1", decoded.ringID)
	require.Equal(t, "aabb", decoded.ringPk)
	require.Equal(t, strings.Repeat("11", 32), decoded.ringStateSha256)
	require.Equal(t, uint64(7), decoded.protocolVersion)
	require.Equal(t, "dkg-session-1", decoded.requestID)
	require.Equal(t, reportTestObservedAt+reportObservedAtGraceSecs, decoded.signedAt)
	require.Equal(t, "accused", decoded.responderNodeKey)
	require.Equal(t, offlineOriginProtocolPSSRefresh, decoded.originProtocol)
	require.Equal(t, committeeScopeCurrent, decoded.accusedCommitteeScope)
	require.Equal(t, committeeScopeCurrent, decoded.signingCommitteeScope)

	rejected := []struct {
		name    string
		payload []byte
	}{
		{"short signature", dkgInvalidRefreshCommitmentPayloadForTest(validStatement.encode(), bytes.Repeat([]byte{42}, 63))},
		{"wrong domain", dkgInvalidRefreshCommitmentPayloadForTest(validStatement.with(func(s *dkgCommitmentStatementFields) { s.domain = "other" }).encode(), signature)},
		{"empty request_id", dkgInvalidRefreshCommitmentPayloadForTest(validStatement.with(func(s *dkgCommitmentStatementFields) { s.requestID = "" }).encode(), signature)},
		{"empty responder", dkgInvalidRefreshCommitmentPayloadForTest(validStatement.with(func(s *dkgCommitmentStatementFields) { s.responderKey = "" }).encode(), signature)},
		{"unsupported origin", dkgInvalidRefreshCommitmentPayloadForTest(validStatement.with(func(s *dkgCommitmentStatementFields) { s.originProtocol = offlineOriginProtocolSign }).encode(), signature)},
		{"reshare origin", dkgInvalidRefreshCommitmentPayloadForTest(validStatement.with(func(s *dkgCommitmentStatementFields) { s.originProtocol = offlineOriginProtocolPSSReshare }).encode(), signature)},
		{"unknown accused scope", dkgInvalidRefreshCommitmentPayloadForTest(validStatement.with(func(s *dkgCommitmentStatementFields) { s.accusedCommitteeScope = 99 }).encode(), signature)},
		{"pending accused scope", dkgInvalidRefreshCommitmentPayloadForTest(validStatement.with(func(s *dkgCommitmentStatementFields) { s.accusedCommitteeScope = committeeScopePendingNew }).encode(), signature)},
		{"unknown signing scope", dkgInvalidRefreshCommitmentPayloadForTest(validStatement.with(func(s *dkgCommitmentStatementFields) { s.signingCommitteeScope = 99 }).encode(), signature)},
		{"pending signing scope", dkgInvalidRefreshCommitmentPayloadForTest(validStatement.with(func(s *dkgCommitmentStatementFields) { s.signingCommitteeScope = committeeScopePendingNew }).encode(), signature)},
		{"from node zero", dkgInvalidRefreshCommitmentPayloadForTest(validStatement.with(func(s *dkgCommitmentStatementFields) { s.fromNodeID = 0 }).encode(), signature)},
		{"empty commitment", dkgInvalidRefreshCommitmentPayloadForTest(validStatement.with(func(s *dkgCommitmentStatementFields) { s.commitment = nil }).encode(), signature)},
		{"oversize commitment", dkgInvalidRefreshCommitmentPayloadForTest(validStatement.with(func(s *dkgCommitmentStatementFields) { s.commitment = make([]byte, dkgCommitmentMaxLen+1) }).encode(), signature)},
		{"bad session nonce length", dkgInvalidRefreshCommitmentPayloadForTest(validStatement.with(func(s *dkgCommitmentStatementFields) { s.sessionNonce = bytes.Repeat([]byte{12}, dkgNonceLen-1) }).encode(), signature)},
		{"empty crypto backend", dkgInvalidRefreshCommitmentPayloadForTest(validStatement.with(func(s *dkgCommitmentStatementFields) { s.cryptoBackend = "" }).encode(), signature)},
		{"trailing inner bytes", dkgInvalidRefreshCommitmentPayloadForTest(append(validStatement.encode(), 0), signature)},
		{"trailing outer bytes", append(dkgInvalidRefreshCommitmentPayloadForTest(validStatement.encode(), signature), 0)},
	}
	for _, tc := range rejected {
		t.Run(tc.name, func(t *testing.T) {
			_, err := decodeInvalidCryptoResponsePayload(tc.payload)
			require.ErrorIs(t, err, types.ErrInvalidReport)
		})
	}
}

func TestInvalidCryptoResponseDkgEquivocationPayloadDecodeBindsCommitments(t *testing.T) {
	signatureA := bytes.Repeat([]byte{42}, 64)
	signatureB := bytes.Repeat([]byte{43}, 64)
	commitmentA, commitmentB := dkgEquivocationCommitmentsForTest()

	decoded, err := decodeInvalidCryptoResponsePayload(dkgEquivocationPayloadForTest(
		commitmentA.encode(),
		signatureA,
		commitmentB.encode(),
		signatureB,
	))
	require.NoError(t, err)
	require.Equal(t, reportTestChainID, decoded.chainID)
	require.Equal(t, "ring-1", decoded.ringID)
	require.Equal(t, "aabb", decoded.ringPk)
	require.Equal(t, strings.Repeat("11", 32), decoded.ringStateSha256)
	require.Equal(t, uint64(7), decoded.protocolVersion)
	require.Equal(t, "dkg-session-1", decoded.requestID)
	require.Equal(t, reportTestObservedAt+reportObservedAtGraceSecs, decoded.signedAt)
	require.Equal(t, "accused", decoded.responderNodeKey)
	require.Equal(t, offlineOriginProtocolPSSRefresh, decoded.originProtocol)
	require.Equal(t, committeeScopeCurrent, decoded.accusedCommitteeScope)
	require.Equal(t, committeeScopeCurrent, decoded.signingCommitteeScope)

	reshareA := commitmentA.with(func(s *dkgCommitmentStatementFields) {
		s.originProtocol = offlineOriginProtocolPSSReshare
	})
	reshareB := commitmentB.with(func(s *dkgCommitmentStatementFields) {
		s.originProtocol = offlineOriginProtocolPSSReshare
	})
	decoded, err = decodeInvalidCryptoResponsePayload(dkgEquivocationPayloadForTest(
		reshareA.encode(),
		signatureA,
		reshareB.encode(),
		signatureB,
	))
	require.NoError(t, err)
	require.Equal(t, offlineOriginProtocolPSSReshare, decoded.originProtocol)

	rejected := []struct {
		name    string
		payload []byte
	}{
		{"short commitment_a signature", dkgEquivocationPayloadForTest(commitmentA.encode(), bytes.Repeat([]byte{1}, 63), commitmentB.encode(), signatureB)},
		{"short commitment_b signature", dkgEquivocationPayloadForTest(commitmentA.encode(), signatureA, commitmentB.encode(), bytes.Repeat([]byte{1}, 63))},
		{"wrong domain", dkgEquivocationPayloadForTest(commitmentA.with(func(s *dkgCommitmentStatementFields) { s.domain = "other" }).encode(), signatureA, commitmentB.encode(), signatureB)},
		{"empty request_id", dkgEquivocationPayloadForTest(commitmentA.with(func(s *dkgCommitmentStatementFields) { s.requestID = "" }).encode(), signatureA, commitmentB.encode(), signatureB)},
		{"empty responder", dkgEquivocationPayloadForTest(commitmentA.with(func(s *dkgCommitmentStatementFields) { s.responderKey = "" }).encode(), signatureA, commitmentB.encode(), signatureB)},
		{"unsupported origin", dkgEquivocationPayloadForTest(commitmentA.with(func(s *dkgCommitmentStatementFields) { s.originProtocol = offlineOriginProtocolSign }).encode(), signatureA, commitmentB.encode(), signatureB)},
		{"unknown accused scope", dkgEquivocationPayloadForTest(commitmentA.with(func(s *dkgCommitmentStatementFields) { s.accusedCommitteeScope = 99 }).encode(), signatureA, commitmentB.encode(), signatureB)},
		{"pending accused scope", dkgEquivocationPayloadForTest(commitmentA.with(func(s *dkgCommitmentStatementFields) { s.accusedCommitteeScope = committeeScopePendingNew }).encode(), signatureA, commitmentB.encode(), signatureB)},
		{"from node zero", dkgEquivocationPayloadForTest(commitmentA.with(func(s *dkgCommitmentStatementFields) { s.fromNodeID = 0 }).encode(), signatureA, commitmentB.encode(), signatureB)},
		{"empty commitment", dkgEquivocationPayloadForTest(commitmentA.with(func(s *dkgCommitmentStatementFields) { s.commitment = nil }).encode(), signatureA, commitmentB.encode(), signatureB)},
		{"oversize commitment", dkgEquivocationPayloadForTest(commitmentA.with(func(s *dkgCommitmentStatementFields) { s.commitment = make([]byte, dkgCommitmentMaxLen+1) }).encode(), signatureA, commitmentB.encode(), signatureB)},
		{"bad session nonce length", dkgEquivocationPayloadForTest(commitmentA.with(func(s *dkgCommitmentStatementFields) { s.sessionNonce = bytes.Repeat([]byte{12}, dkgNonceLen-1) }).encode(), signatureA, commitmentB.encode(), signatureB)},
		{"empty crypto backend", dkgEquivocationPayloadForTest(commitmentA.with(func(s *dkgCommitmentStatementFields) { s.cryptoBackend = "" }).encode(), signatureA, commitmentB.encode(), signatureB)},
		{"ring mismatch", dkgEquivocationPayloadForTest(commitmentA.encode(), signatureA, commitmentB.with(func(s *dkgCommitmentStatementFields) { s.ringID = "other" }).encode(), signatureB)},
		{"session mismatch", dkgEquivocationPayloadForTest(commitmentA.encode(), signatureA, commitmentB.with(func(s *dkgCommitmentStatementFields) { s.requestID = "other" }).encode(), signatureB)},
		{"responder mismatch", dkgEquivocationPayloadForTest(commitmentA.encode(), signatureA, commitmentB.with(func(s *dkgCommitmentStatementFields) { s.responderKey = "other" }).encode(), signatureB)},
		{"origin mismatch", dkgEquivocationPayloadForTest(commitmentA.encode(), signatureA, commitmentB.with(func(s *dkgCommitmentStatementFields) { s.originProtocol = offlineOriginProtocolPSSReshare }).encode(), signatureB)},
		{"from node mismatch", dkgEquivocationPayloadForTest(commitmentA.encode(), signatureA, commitmentB.with(func(s *dkgCommitmentStatementFields) { s.fromNodeID = 3 }).encode(), signatureB)},
		{"crypto backend mismatch", dkgEquivocationPayloadForTest(commitmentA.encode(), signatureA, commitmentB.with(func(s *dkgCommitmentStatementFields) { s.cryptoBackend = "dkg/other" }).encode(), signatureB)},
		{"different session nonce", dkgEquivocationPayloadForTest(commitmentA.encode(), signatureA, commitmentB.with(func(s *dkgCommitmentStatementFields) { s.sessionNonce = bytes.Repeat([]byte{13}, dkgNonceLen) }).encode(), signatureB)},
		{"identical commitments", dkgEquivocationPayloadForTest(commitmentA.encode(), signatureA, commitmentB.with(func(s *dkgCommitmentStatementFields) { s.commitment = commitmentA.commitment }).encode(), signatureB)},
		{"trailing outer bytes", append(dkgEquivocationPayloadForTest(commitmentA.encode(), signatureA, commitmentB.encode(), signatureB), 0)},
		{"trailing commitment_a bytes", dkgEquivocationPayloadForTest(append(commitmentA.encode(), 0), signatureA, commitmentB.encode(), signatureB)},
	}
	for _, tc := range rejected {
		t.Run(tc.name, func(t *testing.T) {
			_, err := decodeInvalidCryptoResponsePayload(tc.payload)
			require.ErrorIs(t, err, types.ErrInvalidReport)
		})
	}
}

func TestValidateInvalidCryptoResponseStatementBindingAndAnchor(t *testing.T) {
	report := &types.ReportEnvelope{
		ChainId:         "chain",
		RingId:          "ring",
		RingPk:          "pk",
		RingStateSha256: "digest",
		SessionId:       "req",
		AccusedNodeKey:  "accused",
		ObservedAt:      reportTestObservedAt,
	}
	base := invalidCryptoResponseStatement{
		chainID:          "chain",
		ringID:           "ring",
		ringPk:           "pk",
		ringStateSha256:  "digest",
		requestID:        "req",
		signedAt:         reportTestObservedAt + reportObservedAtGraceSecs,
		responderNodeKey: "accused",
	}
	require.NoError(t, validateInvalidCryptoResponseStatement(report, base))

	rejected := []struct {
		name   string
		mutate func(*invalidCryptoResponseStatement)
	}{
		{"chain mismatch", func(s *invalidCryptoResponseStatement) { s.chainID = "other" }},
		{"ring id mismatch", func(s *invalidCryptoResponseStatement) { s.ringID = "other" }},
		{"ring pk mismatch", func(s *invalidCryptoResponseStatement) { s.ringPk = "other" }},
		{"ring state mismatch", func(s *invalidCryptoResponseStatement) { s.ringStateSha256 = "other" }},
		{"session mismatch", func(s *invalidCryptoResponseStatement) { s.requestID = "other" }},
		{"responder mismatch", func(s *invalidCryptoResponseStatement) { s.responderNodeKey = "other" }},
		{"anchor off by one late", func(s *invalidCryptoResponseStatement) { s.signedAt++ }},
		{"anchor off by one early", func(s *invalidCryptoResponseStatement) { s.signedAt-- }},
		{"signed_at below grace", func(s *invalidCryptoResponseStatement) { s.signedAt = reportObservedAtGraceSecs - 1 }},
	}
	for _, tc := range rejected {
		t.Run(tc.name, func(t *testing.T) {
			statement := base
			tc.mutate(&statement)
			err := validateInvalidCryptoResponseStatement(report, statement)
			require.ErrorIs(t, err, types.ErrInvalidReport)
		})
	}
}

func TestUnauthorizedRequestPayloadDecodeAcceptsAndRejectsMalformed(t *testing.T) {
	validSig := make([]byte, 64)
	valid := validRelayRequestStatementFields()

	statement, err := decodeUnauthorizedRequestPayload(
		unauthorizedRequestPayloadForTest(valid.encode(), validSig, "42"),
	)
	require.NoError(t, err)
	require.Equal(t, offlineOriginProtocolPRE, statement.originProtocol)
	require.Equal(t, "accused", statement.relayerNodeKey)
	require.Equal(t, uint64(0), statement.protocolVersion)

	// The sign origin is equally valid.
	_, err = decodeUnauthorizedRequestPayload(unauthorizedRequestPayloadForTest(
		valid.with(func(s *relayRequestStatementFields) { s.originProtocol = offlineOriginProtocolSign }).encode(),
		validSig, "42",
	))
	require.NoError(t, err)

	rejected := []struct {
		name    string
		payload []byte
	}{
		{"short signature", unauthorizedRequestPayloadForTest(valid.encode(), make([]byte, 63), "42")},
		{"empty anchor", unauthorizedRequestPayloadForTest(valid.encode(), validSig, "")},
		{"wrong domain", unauthorizedRequestPayloadForTest(valid.with(func(s *relayRequestStatementFields) { s.domain = "other" }).encode(), validSig, "42")},
		{"bad origin", unauthorizedRequestPayloadForTest(valid.with(func(s *relayRequestStatementFields) { s.originProtocol = "dkg" }).encode(), validSig, "42")},
		{"non-current accused scope", unauthorizedRequestPayloadForTest(valid.with(func(s *relayRequestStatementFields) { s.accusedScope = committeeScopePendingNew }).encode(), validSig, "42")},
		{"non-current signing scope", unauthorizedRequestPayloadForTest(valid.with(func(s *relayRequestStatementFields) { s.signingScope = committeeScopePendingNew }).encode(), validSig, "42")},
		{"zero from_node_id", unauthorizedRequestPayloadForTest(valid.with(func(s *relayRequestStatementFields) { s.fromNodeID = 0 }).encode(), validSig, "42")},
		{"empty actor", unauthorizedRequestPayloadForTest(valid.with(func(s *relayRequestStatementFields) { s.actorID = "" }).encode(), validSig, "42")},
		{"empty object", unauthorizedRequestPayloadForTest(valid.with(func(s *relayRequestStatementFields) { s.objectID = "" }).encode(), validSig, "42")},
		{"half-set window", unauthorizedRequestPayloadForTest(valid.with(func(s *relayRequestStatementFields) { v := uint64(1); s.validWindowStart = &v }).encode(), validSig, "42")},
		{"drift exceeds max", unauthorizedRequestPayloadForTest(valid.with(func(s *relayRequestStatementFields) { s.userSignedAt = s.signedAt - relayCheckMaxDriftSecs - 1 }).encode(), validSig, "42")},
	}
	for _, tc := range rejected {
		t.Run(tc.name, func(t *testing.T) {
			_, err := decodeUnauthorizedRequestPayload(tc.payload)
			require.ErrorIs(t, err, types.ErrInvalidReport)
		})
	}
}

func TestValidateRelayRequestStatementBindingAndAnchor(t *testing.T) {
	report := &types.ReportEnvelope{
		ChainId:         "chain",
		RingId:          "ring",
		RingPk:          "pk",
		RingStateSha256: "digest",
		SessionId:       "req",
		AccusedNodeKey:  "accused",
		ObservedAt:      reportTestObservedAt,
	}
	base := relayRequestStatement{
		chainID:         "chain",
		ringID:          "ring",
		ringPk:          "pk",
		ringStateSha256: "digest",
		requestID:       "req",
		signedAt:        reportTestObservedAt + reportObservedAtGraceSecs,
		relayerNodeKey:  "accused",
	}
	require.NoError(t, validateRelayRequestStatement(report, base))

	rejected := []struct {
		name   string
		mutate func(*relayRequestStatement)
	}{
		{"chain mismatch", func(s *relayRequestStatement) { s.chainID = "other" }},
		{"ring id mismatch", func(s *relayRequestStatement) { s.ringID = "other" }},
		{"ring pk mismatch", func(s *relayRequestStatement) { s.ringPk = "other" }},
		{"ring state mismatch", func(s *relayRequestStatement) { s.ringStateSha256 = "other" }},
		{"session mismatch", func(s *relayRequestStatement) { s.requestID = "other" }},
		{"relayer mismatch", func(s *relayRequestStatement) { s.relayerNodeKey = "other" }},
		{"anchor off by one late", func(s *relayRequestStatement) { s.signedAt++ }},
		{"anchor off by one early", func(s *relayRequestStatement) { s.signedAt-- }},
		{"signed_at below grace", func(s *relayRequestStatement) { s.signedAt = reportObservedAtGraceSecs - 1 }},
	}
	for _, tc := range rejected {
		t.Run(tc.name, func(t *testing.T) {
			statement := base
			tc.mutate(&statement)
			require.ErrorIs(t, validateRelayRequestStatement(report, statement), types.ErrInvalidReport)
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
				NodeOfflineDemerits:           5,
				InvalidCryptoResponseDemerits: types.DefaultInvalidCryptoResponseDemerits,
				UnauthorizedRequestDemerits:   types.DefaultUnauthorizedRequestDemerits,
				ResetIntervalSeconds:          types.DefaultDemeritResetIntervalSecs,
			},
			KickThreshold: types.DefaultReportingKickThreshold,
		},
	}, NodeOfflineReportType)
	require.NoError(t, err)
	require.Equal(t, uint64(5), amount)

	amount, err = DemeritAmountForReportType(&types.Ring{
		Reporting: types.ReportingConfig{
			DemeritConfig: types.DemeritConfig{
				NodeOfflineDemerits:           5,
				InvalidCryptoResponseDemerits: 7,
				UnauthorizedRequestDemerits:   types.DefaultUnauthorizedRequestDemerits,
				ResetIntervalSeconds:          types.DefaultDemeritResetIntervalSecs,
			},
			KickThreshold: types.DefaultReportingKickThreshold,
		},
	}, InvalidCryptoResponseReportType)
	require.NoError(t, err)
	require.Equal(t, uint64(7), amount)
}

func TestMsgServer_SubmitReport_InvalidCryptoPREAcceptsRetainsAndDedupes(t *testing.T) {
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
		ReportType:      InvalidCryptoResponseReportType,
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

func TestMsgServer_SubmitReport_InvalidCryptoDKGShareAcceptsAndDedupes(t *testing.T) {
	fixture, sk := setupBLSReportFixture(t, "orbis-report-dkgshare-ikm-00", 2)

	report := fixture.validDkgInvalidShareReport(t)
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
		ReportType:      InvalidCryptoResponseReportType,
		ReporterNodeKey: fixture.reporterKey,
		AccusedNodeKey:  fixture.accusedKey,
	})

	_, err = fixture.k.SubmitReport(fixture.ctx, msg)
	require.ErrorIs(t, err, types.ErrReportAlreadyAccepted)

	secondReport := fixture.validDkgInvalidShareReport(t)
	secondReport.ReporterNodeKey = fixture.validatorKey
	second := fixture.signBLSReport(t, sk, secondReport)
	require.NotEqual(t, msg.ReportId, second.ReportId)
	_, err = fixture.k.SubmitReport(fixture.ctx, second)
	require.ErrorIs(t, err, types.ErrReportAlreadyAccepted)
	require.Equal(t, uint64(1), fixture.k.GetNodeDemerits(fixture.ctx, fixture.ringID, fixture.accusedKey))

	sessionID, err := reportSessionDedupeID(&msg.Report, reportPayload{
		originProtocol:        offlineOriginProtocolPSSRefresh,
		originProtocolVersion: 0,
		accusedCommitteeScope: committeeScopeCurrent,
		signingCommitteeScope: committeeScopeCurrent,
	})
	require.NoError(t, err)
	require.True(t, fixture.k.HasAcceptedReportSession(fixture.ctx, sessionID))
}

func TestMsgServer_SubmitReport_InvalidCryptoDKGInvalidRefreshCommitmentAcceptsAndDedupes(t *testing.T) {
	fixture, sk := setupBLSReportFixture(t, "orbis-report-dkgcommit-ikm", 2)

	report := fixture.validDkgInvalidRefreshCommitmentReport(t)
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
		ReportType:      InvalidCryptoResponseReportType,
		ReporterNodeKey: fixture.reporterKey,
		AccusedNodeKey:  fixture.accusedKey,
	})

	_, err = fixture.k.SubmitReport(fixture.ctx, msg)
	require.ErrorIs(t, err, types.ErrReportAlreadyAccepted)

	secondReport := fixture.validDkgInvalidRefreshCommitmentReport(t)
	secondReport.ReporterNodeKey = fixture.validatorKey
	second := fixture.signBLSReport(t, sk, secondReport)
	require.NotEqual(t, msg.ReportId, second.ReportId)
	_, err = fixture.k.SubmitReport(fixture.ctx, second)
	require.ErrorIs(t, err, types.ErrReportAlreadyAccepted)
	require.Equal(t, uint64(1), fixture.k.GetNodeDemerits(fixture.ctx, fixture.ringID, fixture.accusedKey))

	sessionID, err := reportSessionDedupeID(&msg.Report, reportPayload{
		originProtocol:        offlineOriginProtocolPSSRefresh,
		originProtocolVersion: 0,
		accusedCommitteeScope: committeeScopeCurrent,
		signingCommitteeScope: committeeScopeCurrent,
	})
	require.NoError(t, err)
	require.True(t, fixture.k.HasAcceptedReportSession(fixture.ctx, sessionID))
}

func TestMsgServer_SubmitReport_InvalidCryptoDKGEquivocationAcceptsAndDedupes(t *testing.T) {
	fixture, sk := setupBLSReportFixture(t, "orbis-report-dkgequiv-ikm0", 2)

	report := fixture.validDkgEquivocationReport(t)
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
		ReportType:      InvalidCryptoResponseReportType,
		ReporterNodeKey: fixture.reporterKey,
		AccusedNodeKey:  fixture.accusedKey,
	})

	_, err = fixture.k.SubmitReport(fixture.ctx, msg)
	require.ErrorIs(t, err, types.ErrReportAlreadyAccepted)

	secondReport := fixture.validDkgEquivocationReport(t)
	secondReport.ReporterNodeKey = fixture.validatorKey
	second := fixture.signBLSReport(t, sk, secondReport)
	require.NotEqual(t, msg.ReportId, second.ReportId)
	_, err = fixture.k.SubmitReport(fixture.ctx, second)
	require.ErrorIs(t, err, types.ErrReportAlreadyAccepted)
	require.Equal(t, uint64(1), fixture.k.GetNodeDemerits(fixture.ctx, fixture.ringID, fixture.accusedKey))

	sessionID, err := reportSessionDedupeID(&msg.Report, reportPayload{
		originProtocol:        offlineOriginProtocolPSSRefresh,
		originProtocolVersion: 0,
		accusedCommitteeScope: committeeScopeCurrent,
		signingCommitteeScope: committeeScopeCurrent,
	})
	require.NoError(t, err)
	require.True(t, fixture.k.HasAcceptedReportSession(fixture.ctx, sessionID))
}

func TestMsgServer_SubmitReport_InvalidCryptoPRERejectsTamperedStatements(t *testing.T) {
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

func TestMsgServer_SubmitReport_InvalidCryptoPREDecaf377FROSTAccepts(t *testing.T) {
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
		NodeOfflineDemerits:           3,
		InvalidCryptoResponseDemerits: types.DefaultInvalidCryptoResponseDemerits,
		UnauthorizedRequestDemerits:   types.DefaultUnauthorizedRequestDemerits,
		ResetIntervalSeconds:          types.DefaultDemeritResetIntervalSecs,
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
			NodeOfflineDemerits:           3,
			InvalidCryptoResponseDemerits: types.DefaultInvalidCryptoResponseDemerits,
			UnauthorizedRequestDemerits:   types.DefaultUnauthorizedRequestDemerits,
			ResetIntervalSeconds:          types.DefaultDemeritResetIntervalSecs,
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
			NodeOfflineDemerits:           1,
			InvalidCryptoResponseDemerits: types.DefaultInvalidCryptoResponseDemerits,
			UnauthorizedRequestDemerits:   types.DefaultUnauthorizedRequestDemerits,
			ResetIntervalSeconds:          types.DefaultDemeritResetIntervalSecs,
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
			NodeOfflineDemerits:           3,
			InvalidCryptoResponseDemerits: types.DefaultInvalidCryptoResponseDemerits,
			UnauthorizedRequestDemerits:   types.DefaultUnauthorizedRequestDemerits,
			ResetIntervalSeconds:          types.DefaultDemeritResetIntervalSecs,
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
			NodeOfflineDemerits:           3,
			InvalidCryptoResponseDemerits: types.DefaultInvalidCryptoResponseDemerits,
			UnauthorizedRequestDemerits:   types.DefaultUnauthorizedRequestDemerits,
			ResetIntervalSeconds:          types.DefaultDemeritResetIntervalSecs,
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
		NodeOfflineDemerits:           3,
		InvalidCryptoResponseDemerits: types.DefaultInvalidCryptoResponseDemerits,
		UnauthorizedRequestDemerits:   types.DefaultUnauthorizedRequestDemerits,
		ResetIntervalSeconds:          ReportTTLSeconds * 2,
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
		NodeOfflineDemerits:           3,
		InvalidCryptoResponseDemerits: types.DefaultInvalidCryptoResponseDemerits,
		UnauthorizedRequestDemerits:   types.DefaultUnauthorizedRequestDemerits,
		ResetIntervalSeconds:          10,
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
		NodeOfflineDemerits:           3,
		InvalidCryptoResponseDemerits: types.DefaultInvalidCryptoResponseDemerits,
		UnauthorizedRequestDemerits:   types.DefaultUnauthorizedRequestDemerits,
		ResetIntervalSeconds:          10,
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
		if msg.ReportId[0] == '0' {
			msg.ReportId = "1" + msg.ReportId[1:]
		} else {
			msg.ReportId = "0" + msg.ReportId[1:]
		}
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
	originProtocol        string
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
	w.writeString(s.originProtocol)
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
	return invalidCryptoResponsePayloadForTest(invalidCryptoEvidenceKindPRE, statement, signature)
}

type signResponseStatementFields struct {
	domain                string
	chainID               string
	ringID                string
	ringPk                string
	ringStateSha256       string
	protocolVersion       uint64
	requestID             string
	signedAt              uint64
	responderKey          string
	originProtocol        string
	accusedCommitteeScope byte
	signingCommitteeScope byte
	fromNodeID            uint32
	message               []byte
	signingCommitments    []byte
	derivation            []byte
	metadata              []byte
	sigShare              []byte
	cryptoBackend         string
}

func (s signResponseStatementFields) with(mutate func(*signResponseStatementFields)) signResponseStatementFields {
	mutate(&s)
	return s
}

func (s signResponseStatementFields) encode() []byte {
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
	w.writeString(s.originProtocol)
	w.bytes = append(w.bytes, s.accusedCommitteeScope, s.signingCommitteeScope)
	w.writeU32(s.fromNodeID)
	w.writeBytes(s.message)
	w.writeBytes(s.signingCommitments)
	if s.derivation == nil {
		w.bytes = append(w.bytes, 0)
	} else {
		w.bytes = append(w.bytes, 1)
		w.writeBytes(s.derivation)
	}
	if s.metadata == nil {
		w.bytes = append(w.bytes, 0)
	} else {
		w.bytes = append(w.bytes, 1)
		w.writeBytes(s.metadata)
	}
	w.writeBytes(s.sigShare)
	w.writeString(s.cryptoBackend)
	statement, err := w.finish()
	if err != nil {
		panic(err)
	}
	return statement
}

func signResponsePayloadForTest(statement []byte, signature []byte) []byte {
	return invalidCryptoResponsePayloadForTest(invalidCryptoEvidenceKindSign, statement, signature)
}

type dkgCommitmentStatementFields struct {
	domain                string
	chainID               string
	ringID                string
	ringPk                string
	ringStateSha256       string
	protocolVersion       uint64
	requestID             string
	signedAt              uint64
	responderKey          string
	originProtocol        string
	accusedCommitteeScope byte
	signingCommitteeScope byte
	fromNodeID            uint32
	commitment            []byte
	sessionNonce          []byte
	cryptoBackend         string
}

func (s dkgCommitmentStatementFields) with(mutate func(*dkgCommitmentStatementFields)) dkgCommitmentStatementFields {
	mutate(&s)
	return s
}

func (s dkgCommitmentStatementFields) encode() []byte {
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
	w.writeString(s.originProtocol)
	w.bytes = append(w.bytes, s.accusedCommitteeScope, s.signingCommitteeScope)
	w.writeU32(s.fromNodeID)
	w.writeBytes(s.commitment)
	w.writeBytes(s.sessionNonce)
	w.writeString(s.cryptoBackend)
	statement, err := w.finish()
	if err != nil {
		panic(err)
	}
	return statement
}

type dkgShareStatementFields struct {
	domain                string
	chainID               string
	ringID                string
	ringPk                string
	ringStateSha256       string
	protocolVersion       uint64
	requestID             string
	signedAt              uint64
	responderKey          string
	receiverKey           string
	originProtocol        string
	accusedCommitteeScope byte
	signingCommitteeScope byte
	fromNodeID            uint32
	toNodeID              uint32
	commitmentStatement   dkgCommitmentStatementFields
	commitmentSignature   []byte
	shareValue            []byte
	nonce                 []byte
	cryptoBackend         string
}

func (s dkgShareStatementFields) with(mutate func(*dkgShareStatementFields)) dkgShareStatementFields {
	mutate(&s)
	return s
}

func (s dkgShareStatementFields) encode() []byte {
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
	w.writeString(s.receiverKey)
	w.writeString(s.originProtocol)
	w.bytes = append(w.bytes, s.accusedCommitteeScope, s.signingCommitteeScope)
	w.writeU32(s.fromNodeID)
	w.writeU32(s.toNodeID)
	w.writeBytes(s.commitmentStatement.encode())
	w.writeBytes(s.commitmentSignature)
	w.writeBytes(s.shareValue)
	w.writeBytes(s.nonce)
	w.writeString(s.cryptoBackend)
	statement, err := w.finish()
	if err != nil {
		panic(err)
	}
	return statement
}

func dkgShareStatementForTest() dkgShareStatementFields {
	commitment := dkgCommitmentStatementFields{
		domain:                DkgCommitmentDomain,
		chainID:               reportTestChainID,
		ringID:                "ring-1",
		ringPk:                "aabb",
		ringStateSha256:       strings.Repeat("11", 32),
		protocolVersion:       7,
		requestID:             "dkg-session-1",
		signedAt:              reportTestObservedAt + reportObservedAtGraceSecs - 1,
		responderKey:          "accused",
		originProtocol:        offlineOriginProtocolPSSRefresh,
		accusedCommitteeScope: committeeScopeCurrent,
		signingCommitteeScope: committeeScopeCurrent,
		fromNodeID:            2,
		commitment:            []byte{1, 2, 3, 4},
		sessionNonce:          bytes.Repeat([]byte{12}, dkgNonceLen),
		cryptoBackend:         "dkg/test",
	}
	return dkgShareStatementFields{
		domain:                DkgShareDomain,
		chainID:               reportTestChainID,
		ringID:                "ring-1",
		ringPk:                "aabb",
		ringStateSha256:       strings.Repeat("11", 32),
		protocolVersion:       7,
		requestID:             "dkg-session-1",
		signedAt:              reportTestObservedAt + reportObservedAtGraceSecs,
		responderKey:          "accused",
		receiverKey:           "reporter",
		originProtocol:        offlineOriginProtocolPSSRefresh,
		accusedCommitteeScope: committeeScopeCurrent,
		signingCommitteeScope: committeeScopeCurrent,
		fromNodeID:            2,
		toNodeID:              1,
		commitmentStatement:   commitment,
		commitmentSignature:   bytes.Repeat([]byte{7}, 64),
		shareValue:            []byte{8, 9, 10},
		nonce:                 bytes.Repeat([]byte{11}, dkgNonceLen),
		cryptoBackend:         "dkg/test",
	}
}

func dkgSharePayloadForTest(statement []byte, signature []byte) []byte {
	return invalidCryptoResponsePayloadForTest(invalidCryptoEvidenceKindDkgShare, statement, signature)
}

func dkgInvalidRefreshCommitmentPayloadForTest(statement []byte, signature []byte) []byte {
	return invalidCryptoResponsePayloadForTest(invalidCryptoEvidenceKindDkgInvalidRefreshCommitment, statement, signature)
}

func dkgEquivocationCommitmentsForTest() (dkgCommitmentStatementFields, dkgCommitmentStatementFields) {
	commitmentA := dkgShareStatementForTest().commitmentStatement
	commitmentA.signedAt = reportTestObservedAt + reportObservedAtGraceSecs
	commitmentB := commitmentA
	commitmentB.signedAt = commitmentA.signedAt + 5
	commitmentB.commitment = []byte{4, 3, 2, 1}
	return commitmentA, commitmentB
}

func dkgEquivocationPayloadForTest(
	commitmentAStatement []byte,
	commitmentASignature []byte,
	commitmentBStatement []byte,
	commitmentBSignature []byte,
) []byte {
	w := newReportCanonicalWriter()
	w.writeString(invalidCryptoEvidenceKindDkgEquivocation)
	w.writeBytes(commitmentAStatement)
	w.writeBytes(commitmentASignature)
	w.writeBytes(commitmentBStatement)
	w.writeBytes(commitmentBSignature)
	payload, err := w.finish()
	if err != nil {
		panic(err)
	}
	return payload
}

func invalidCryptoResponsePayloadForTest(evidenceKind string, statement []byte, signature []byte) []byte {
	w := newReportCanonicalWriter()
	w.writeString(evidenceKind)
	w.writeBytes(statement)
	w.writeBytes(signature)
	payload, err := w.finish()
	if err != nil {
		panic(err)
	}
	return payload
}

// relayRequestStatementFields mirrors the orbis-rs RelayRequestStatement canonical
// encoding for tests.
type relayRequestStatementFields struct {
	domain           string
	chainID          string
	ringID           string
	ringPk           string
	ringStateSha256  string
	protocolVersion  uint64
	requestID        string
	signedAt         uint64
	userSignedAt     uint64
	relayerNodeKey   string
	originProtocol   string
	accusedScope     byte
	signingScope     byte
	fromNodeID       uint32
	actorID          string
	objectID         string
	validWindowStart *uint64
	validWindowEnd   *uint64
	timestamp        *uint64
}

func validRelayRequestStatementFields() relayRequestStatementFields {
	return relayRequestStatementFields{
		domain:          RelayRequestDomain,
		chainID:         reportTestChainID,
		ringID:          "ring",
		ringPk:          "pk",
		ringStateSha256: "digest",
		protocolVersion: 0,
		requestID:       "relay-req",
		signedAt:        reportTestObservedAt + reportObservedAtGraceSecs,
		userSignedAt:    reportTestObservedAt + reportObservedAtGraceSecs,
		relayerNodeKey:  "accused",
		originProtocol:  offlineOriginProtocolPRE,
		accusedScope:    committeeScopeCurrent,
		signingScope:    committeeScopeCurrent,
		fromNodeID:      2,
		actorID:         "did:key:actor",
		objectID:        "object",
	}
}

func (s relayRequestStatementFields) with(mutate func(*relayRequestStatementFields)) relayRequestStatementFields {
	mutate(&s)
	return s
}

func (s relayRequestStatementFields) encode() []byte {
	w := newReportCanonicalWriter()
	w.writeString(s.domain)
	w.writeString(s.chainID)
	w.writeString(s.ringID)
	w.writeString(s.ringPk)
	w.writeString(s.ringStateSha256)
	w.writeU64(s.protocolVersion)
	w.writeString(s.requestID)
	w.writeU64(s.signedAt)
	w.writeU64(s.userSignedAt)
	w.writeString(s.relayerNodeKey)
	w.writeString(s.originProtocol)
	w.bytes = append(w.bytes, s.accusedScope, s.signingScope)
	w.writeU32(s.fromNodeID)
	w.writeString(s.actorID)
	w.writeString(s.objectID)
	w.writeOptionalU64(s.validWindowStart)
	w.writeOptionalU64(s.validWindowEnd)
	w.writeOptionalU64(s.timestamp)
	statement, err := w.finish()
	if err != nil {
		panic(err)
	}
	return statement
}

func unauthorizedRequestPayloadForTest(statement []byte, signature []byte, anchor string) []byte {
	w := newReportCanonicalWriter()
	w.writeBytes(statement)
	w.writeBytes(signature)
	w.writeString(anchor)
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
		originProtocol:  offlineOriginProtocolPRE,
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
		originProtocol:  offlineOriginProtocolPRE,
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
	report.ReportType = InvalidCryptoResponseReportType
	report.Payload = preInvalidProofPayloadForTest(
		f.preInvalidProofStatementFields(t).encode(),
		bytes.Repeat([]byte{42}, 64),
	)
	return report
}

func (f reportTestFixture) dkgShareStatementFields(t *testing.T) dkgShareStatementFields {
	t.Helper()
	require.NotNil(t, f.originalRing)
	ringDigest, err := reportRingStateSHA256(f.originalRing)
	require.NoError(t, err)
	statement := dkgShareStatementForTest()
	statement.chainID = f.ctx.ChainID()
	statement.ringID = f.ringID
	statement.ringPk = f.originalRing.RingPk
	statement.ringStateSha256 = ringDigest
	statement.protocolVersion = 0
	statement.responderKey = f.accusedKey
	statement.receiverKey = f.reporterKey
	statement.commitmentStatement.chainID = statement.chainID
	statement.commitmentStatement.ringID = statement.ringID
	statement.commitmentStatement.ringPk = statement.ringPk
	statement.commitmentStatement.ringStateSha256 = statement.ringStateSha256
	statement.commitmentStatement.protocolVersion = statement.protocolVersion
	statement.commitmentStatement.requestID = statement.requestID
	statement.commitmentStatement.responderKey = statement.responderKey
	statement.commitmentStatement.originProtocol = statement.originProtocol
	statement.commitmentStatement.accusedCommitteeScope = statement.accusedCommitteeScope
	statement.commitmentStatement.signingCommitteeScope = statement.signingCommitteeScope
	statement.commitmentStatement.fromNodeID = statement.fromNodeID
	statement.commitmentStatement.cryptoBackend = statement.cryptoBackend
	return statement
}

func (f reportTestFixture) dkgInvalidRefreshCommitmentStatementFields(t *testing.T) dkgCommitmentStatementFields {
	t.Helper()
	commitment, _ := f.dkgEquivocationCommitments(t)
	return commitment
}

func (f reportTestFixture) dkgEquivocationCommitments(t *testing.T) (dkgCommitmentStatementFields, dkgCommitmentStatementFields) {
	t.Helper()
	require.NotNil(t, f.originalRing)
	ringDigest, err := reportRingStateSHA256(f.originalRing)
	require.NoError(t, err)
	commitmentA, commitmentB := dkgEquivocationCommitmentsForTest()
	for _, commitment := range []*dkgCommitmentStatementFields{&commitmentA, &commitmentB} {
		commitment.chainID = f.ctx.ChainID()
		commitment.ringID = f.ringID
		commitment.ringPk = f.originalRing.RingPk
		commitment.ringStateSha256 = ringDigest
		commitment.protocolVersion = 0
		commitment.responderKey = f.accusedKey
	}
	return commitmentA, commitmentB
}

func (f reportTestFixture) validDkgInvalidShareReport(t *testing.T) types.ReportEnvelope {
	t.Helper()
	statement := f.dkgShareStatementFields(t)
	report := f.validReport(t, committeeScopeCurrent, committeeScopeCurrent, 0)
	report.ReportType = InvalidCryptoResponseReportType
	report.SessionId = statement.requestID
	report.Payload = dkgSharePayloadForTest(
		statement.encode(),
		bytes.Repeat([]byte{42}, 64),
	)
	return report
}

func (f reportTestFixture) validDkgInvalidRefreshCommitmentReport(t *testing.T) types.ReportEnvelope {
	t.Helper()
	statement := f.dkgInvalidRefreshCommitmentStatementFields(t)
	report := f.validReport(t, committeeScopeCurrent, committeeScopeCurrent, 0)
	report.ReportType = InvalidCryptoResponseReportType
	report.SessionId = statement.requestID
	report.Payload = dkgInvalidRefreshCommitmentPayloadForTest(
		statement.encode(),
		bytes.Repeat([]byte{42}, 64),
	)
	return report
}

func (f reportTestFixture) validDkgEquivocationReport(t *testing.T) types.ReportEnvelope {
	t.Helper()
	commitmentA, commitmentB := f.dkgEquivocationCommitments(t)
	report := f.validReport(t, committeeScopeCurrent, committeeScopeCurrent, 0)
	report.ReportType = InvalidCryptoResponseReportType
	report.SessionId = commitmentA.requestID
	report.Payload = dkgEquivocationPayloadForTest(
		commitmentA.encode(),
		bytes.Repeat([]byte{42}, 64),
		commitmentB.encode(),
		bytes.Repeat([]byte{43}, 64),
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
