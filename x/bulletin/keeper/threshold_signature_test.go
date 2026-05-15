package keeper

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sourcenetwork/sourcehub/x/bulletin/types"
)

func TestDecodeHexBytes(t *testing.T) {
	decoded, err := decodeHexBytes("0aFF")
	require.NoError(t, err)
	require.Equal(t, []byte{0x0a, 0xff}, decoded)

	decoded, err = decodeHexBytes(" 0aff ")
	require.NoError(t, err)
	require.Equal(t, []byte{0x0a, 0xff}, decoded)

	_, err = decodeHexBytes("0x0aff")
	require.Error(t, err)

	_, err = decodeHexBytes("AQIDBA==")
	require.Error(t, err)

	_, err = decodeHexBytes("")
	require.Error(t, err)
}

func TestRingPayloadPublicKeyEncodingMatchesOrbisHex(t *testing.T) {
	payload := ringPayloadWithSeed(
		t,
		"sourcehub-bulletin-bls-test-seed-0001",
		[]string{"peer1"},
		1,
		nil,
		nil,
	)
	ringPayload, err := parseRingPayloadJSON(payload)
	require.NoError(t, err)

	publicKey, err := decodeHexBytes(*ringPayload.RingPK)
	require.NoError(t, err)
	require.Len(t, publicKey, bls12381PublicKeySize)
	require.Equal(t, *ringPayload.RingPK, hex.EncodeToString(publicKey))
}

func TestVerifyDecaf377FROSTThresholdSignature(t *testing.T) {
	publicKeyHex := "48b01e513dd37d94c3b48940dc133b92ccba7f546e99d3fc2e602d284f609f00"
	message, err := hex.DecodeString("6f7262697320646563616633373720696e7465726f70206d657373616765")
	require.NoError(t, err)
	signature, err := hex.DecodeString("588125a8f4e2bab8d16affc4ca60c5f64b50d38d2bb053148021631f72e99b0626e73e709ee9e725af65a80d824a8207f11e3fe8a293f40828ad365a6d9e2200")
	require.NoError(t, err)

	require.NoError(t, verifyThresholdSignature(ThresholdSignatureSchemeDecaf377FROST, publicKeyHex, message, signature))

	tamperedSignature := append([]byte(nil), signature...)
	tamperedSignature[4] ^= 0x01
	require.ErrorIs(
		t,
		verifyThresholdSignature(ThresholdSignatureSchemeDecaf377FROST, publicKeyHex, message, tamperedSignature),
		types.ErrInvalidThresholdSignature,
	)

	wrongMessage := []byte("wrong message")
	require.ErrorIs(
		t,
		verifyThresholdSignature(ThresholdSignatureSchemeDecaf377FROST, publicKeyHex, wrongMessage, signature),
		types.ErrInvalidThresholdSignature,
	)

	require.ErrorIs(
		t,
		verifyThresholdSignature(ThresholdSignatureSchemeDecaf377FROST, publicKeyHex, message, signature[:len(signature)-1]),
		types.ErrInvalidThresholdSignature,
	)
}
