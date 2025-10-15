package feegrant_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/sourcenetwork/sourcehub/x/feegrant"

	codecaddress "github.com/cosmos/cosmos-sdk/codec/address"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestMarshalAndUnmarshalFeegrantKey(t *testing.T) {
	addressCodec := codecaddress.NewBech32Codec("source")
	grantee, err := addressCodec.StringToBytes("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)
	granter, err := addressCodec.StringToBytes("source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et")
	require.NoError(t, err)

	key := feegrant.FeeAllowanceKey(granter, grantee)
	require.Len(t, key, len(grantee)+len(granter)+3)
	require.Equal(t, feegrant.FeeAllowancePrefixByGrantee(grantee), key[:len(grantee)+2])

	g1, g2 := feegrant.ParseAddressesFromFeeAllowanceKey(key)
	require.Equal(t, granter, g1)
	require.Equal(t, grantee, g2)
}

func TestMarshalAndUnmarshalFeegrantKeyQueueKey(t *testing.T) {
	addressCodec := codecaddress.NewBech32Codec("source")
	grantee, err := addressCodec.StringToBytes("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)
	granter, err := addressCodec.StringToBytes("source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et")
	require.NoError(t, err)

	exp := time.Now()
	expBytes := sdk.FormatTimeBytes(exp)

	key := feegrant.FeeAllowancePrefixQueue(&exp, feegrant.FeeAllowanceKey(granter, grantee)[1:])
	require.Len(t, key, len(grantee)+len(granter)+3+len(expBytes))

	granter1, grantee1 := feegrant.ParseAddressesFromFeeAllowanceQueueKey(key)
	require.Equal(t, granter, granter1)
	require.Equal(t, grantee, grantee1)
}

func TestMarshalAndUnmarshalDIDFeegrantKey(t *testing.T) {
	addressCodec := codecaddress.NewBech32Codec("source")
	granter, err := addressCodec.StringToBytes("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)

	testDID := "did:example:alice123"

	key := feegrant.FeeAllowanceByDIDKey(granter, testDID)
	require.Len(t, key, len(testDID)+len(granter)+3)
	require.Equal(t, feegrant.FeeAllowancePrefixByDID(testDID), key[:len(testDID)+2])

	parsedGranter, parsedDID := feegrant.ParseGranterDIDFromFeeAllowanceKey(key)
	require.Equal(t, granter, parsedGranter)
	require.Equal(t, testDID, parsedDID)
}

func TestMarshalAndUnmarshalDIDFeegrantQueueKey(t *testing.T) {
	addressCodec := codecaddress.NewBech32Codec("source")
	granter, err := addressCodec.StringToBytes("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)

	testDID := "did:example:bob456"
	exp := time.Now()
	expBytes := sdk.FormatTimeBytes(exp)

	// Create the DID allowance key (without the prefix, as used in queue)
	didKey := feegrant.FeeAllowanceByDIDKey(granter, testDID)[1:] // Remove the 0x02 prefix

	queueKey := feegrant.DIDFeeAllowancePrefixQueue(&exp, didKey)
	require.Len(t, queueKey, len(testDID)+len(granter)+3+len(expBytes)) // 0x03 + exp + did_len + did + granter_len + granter

	parsedGranter, parsedDID := feegrant.ParseGranterDIDFromDIDAllowanceQueueKey(queueKey)
	require.Equal(t, granter, parsedGranter)
	require.Equal(t, testDID, parsedDID)
}

func TestDIDKeyPrefixes(t *testing.T) {
	require.Equal(t, []byte{0x00}, feegrant.FeeAllowanceKeyPrefix)
	require.Equal(t, []byte{0x01}, feegrant.FeeAllowanceQueueKeyPrefix)
	require.Equal(t, []byte{0x02}, feegrant.DIDFeeAllowanceKeyPrefix)
	require.Equal(t, []byte{0x03}, feegrant.DIDFeeAllowanceQueueKeyPrefix)

	prefixes := [][]byte{
		feegrant.FeeAllowanceKeyPrefix,
		feegrant.FeeAllowanceQueueKeyPrefix,
		feegrant.DIDFeeAllowanceKeyPrefix,
		feegrant.DIDFeeAllowanceQueueKeyPrefix,
	}

	for i := 0; i < len(prefixes); i++ {
		for j := i + 1; j < len(prefixes); j++ {
			require.NotEqual(t, prefixes[i], prefixes[j], "Prefixes %d and %d should be different", i, j)
		}
	}
}

func TestDIDPrefixByDID(t *testing.T) {
	testDID := "did:example:bob"
	prefix := feegrant.FeeAllowancePrefixByDID(testDID)

	require.Equal(t, feegrant.DIDFeeAllowanceKeyPrefix[0], prefix[0])

	require.Len(t, prefix, len(testDID)+2)

	didBytes := prefix[2:]
	require.Equal(t, testDID, string(didBytes))
}

func TestDIDAllowanceByExpTimeKey(t *testing.T) {
	exp := time.Now()
	expKey := feegrant.DIDAllowanceByExpTimeKey(&exp)

	require.Equal(t, feegrant.DIDFeeAllowanceQueueKeyPrefix[0], expKey[0])

	expBytes := sdk.FormatTimeBytes(exp)
	require.Len(t, expKey, 1+len(expBytes))
}

func TestKeyFormatConsistency(t *testing.T) {
	addressCodec := codecaddress.NewBech32Codec("source")
	granter, err := addressCodec.StringToBytes("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)
	grantee, err := addressCodec.StringToBytes("source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et")
	require.NoError(t, err)

	testDID := "did:example:alice"

	regularKey := feegrant.FeeAllowanceKey(granter, grantee)
	require.Equal(t, feegrant.FeeAllowanceKeyPrefix[0], regularKey[0])

	didKey := feegrant.FeeAllowanceByDIDKey(granter, testDID)
	require.Equal(t, feegrant.DIDFeeAllowanceKeyPrefix[0], didKey[0])

	require.NotEqual(t, regularKey, didKey)

	parsedGranter1, parsedGrantee := feegrant.ParseAddressesFromFeeAllowanceKey(regularKey)
	require.Equal(t, granter, parsedGranter1)
	require.Equal(t, grantee, parsedGrantee)

	parsedGranter2, parsedDID := feegrant.ParseGranterDIDFromFeeAllowanceKey(didKey)
	require.Equal(t, granter, parsedGranter2)
	require.Equal(t, testDID, parsedDID)
}
