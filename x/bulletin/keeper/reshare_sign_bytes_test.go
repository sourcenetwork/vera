package keeper

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sourcenetwork/sourcehub/x/bulletin/types"
)

type ringReshareFinalizeVector struct {
	Domain                    string `json:"domain"`
	ChainID                   string `json:"chain_id"`
	Namespace                 string `json:"namespace"`
	PostID                    string `json:"post_id"`
	SignatureScheme           string `json:"signature_scheme"`
	BLSDST                    string `json:"bls_dst"`
	SignerKeygenSeed          string `json:"signer_keygen_seed"`
	CurrentPayload            string `json:"current_payload"`
	FinalizedPayload          string `json:"finalized_payload"`
	CurrentPayloadSHA256Hex   string `json:"current_payload_sha256_hex"`
	FinalizedPayloadSHA256Hex string `json:"finalized_payload_sha256_hex"`
	CanonicalSignBytesHex     string `json:"canonical_sign_bytes_hex"`
	RingPublicKeyHex          string `json:"ring_public_key_hex"`
	BlockNumberNonce          uint64 `json:"block_number_nonce"`
	SignatureHex              string `json:"signature_hex"`
}

func TestRingReshareFinalizeSignBytesVector(t *testing.T) {
	vectorBytes, err := os.ReadFile("testdata/ring_reshare_finalize_vector.json")
	require.NoError(t, err)

	var vector ringReshareFinalizeVector
	require.NoError(t, json.Unmarshal(vectorBytes, &vector))
	require.Equal(t, RingReshareFinalizeSignDocDomain, vector.Domain)
	require.Equal(t, ThresholdSignatureSchemeBLS12381G1PKG2SigNUL, vector.SignatureScheme)
	require.Equal(t, bls12381G2SignatureDST, vector.BLSDST)

	currentPayload := []byte(vector.CurrentPayload)
	finalizedPayload, err := deriveFinalizedRingPayloadReshare(currentPayload)
	require.NoError(t, err)
	require.Equal(t, vector.FinalizedPayload, string(finalizedPayload))

	currentPayloadHash := sha256.Sum256(currentPayload)
	finalizedPayloadHash := sha256.Sum256(finalizedPayload)
	require.Equal(t, vector.CurrentPayloadSHA256Hex, hex.EncodeToString(currentPayloadHash[:]))
	require.Equal(t, vector.FinalizedPayloadSHA256Hex, hex.EncodeToString(finalizedPayloadHash[:]))

	signBytes, err := ringReshareFinalizeSignBytes(
		vector.ChainID,
		vector.Namespace,
		vector.PostID,
		currentPayload,
		finalizedPayload,
	)
	require.NoError(t, err)
	require.Equal(t, vector.CanonicalSignBytesHex, hex.EncodeToString(signBytes))

	var signDoc types.RingReshareFinalizeSignDoc
	require.NoError(t, signDoc.Unmarshal(signBytes))
	require.Equal(t, vector.Domain, signDoc.Domain)
	require.Equal(t, vector.ChainID, signDoc.ChainId)
	require.Equal(t, vector.Namespace, signDoc.Namespace)
	require.Equal(t, vector.PostID, signDoc.PostId)
	require.Equal(t, vector.RingPublicKeyHex, signDoc.RingPk)
	require.Equal(t, currentPayloadHash[:], signDoc.CurrentPayloadSha256)
	require.Equal(t, finalizedPayloadHash[:], signDoc.FinalizedPayloadSha256)
	require.Equal(t, vector.BlockNumberNonce, signDoc.BlockNumberNonce)

	ringPayload, err := parseRingPayloadJSON(currentPayload)
	require.NoError(t, err)
	require.Equal(t, vector.RingPublicKeyHex, *ringPayload.RingPK)
	require.Equal(t, vector.BlockNumberNonce, *ringPayload.BlockNumberNonce)

	signature, err := hex.DecodeString(vector.SignatureHex)
	require.NoError(t, err)
	require.Equal(t, vector.SignatureHex, hex.EncodeToString(signPayloadWithSeed(t, vector.SignerKeygenSeed, signBytes)))
	require.NoError(t, verifyThresholdSignature(vector.SignatureScheme, vector.RingPublicKeyHex, signBytes, signature))
}
