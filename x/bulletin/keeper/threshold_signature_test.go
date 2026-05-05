package keeper

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
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
