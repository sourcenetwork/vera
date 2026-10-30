package keeper

import (
	"crypto/sha512"
	"encoding/hex"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	blst "github.com/supranational/blst/bindings/go"

	appparams "github.com/sourcenetwork/vera/app/params"
	jubjub "github.com/sourcenetwork/vera/x/orbis/jubjub"
	"github.com/sourcenetwork/vera/x/orbis/jubjub/jubjubfrost"
	"github.com/sourcenetwork/vera/x/orbis/jubjub/jubjubtest"
	"github.com/sourcenetwork/vera/x/orbis/types"
)

func TestRingReshareSignStateHashIncludesTrustedAuthRelays(t *testing.T) {
	ring := &types.Ring{
		RingPk:                 "ring-pk",
		PeerNodeKeys:           []string{"node-1"},
		Threshold:              1,
		AllowTrustedAuthRelays: true,
		TrustedAuthRelayDids:   []string{testRelayDID},
	}

	withRelay, err := ringReshareSignStateHash(ring)
	require.NoError(t, err)
	ring.TrustedAuthRelayDids = nil
	withoutRelay, err := ringReshareSignStateHash(ring)
	require.NoError(t, err)
	require.NotEqual(t, withRelay, withoutRelay)
	ring.AllowTrustedAuthRelays = false
	directOnly, err := ringReshareSignStateHash(ring)
	require.NoError(t, err)
	require.NotEqual(t, withoutRelay, directOnly)
}

func TestJubjubIdentityPublicKeyForgeryRejected(t *testing.T) {
	identityBytes := jubjub.IdentityBytes[:]

	// The underlying Schnorr equation would accept this construction for
	// Y = 0: choose z, then publish R = z*G. No signing secret is needed.
	//
	// Unlike decaf377's bare orbisfrost.Verify, jubjubfrost.Verify's
	// underlying verify_core already rejects the identity public key by
	// itself — it mirrors orbis-rs's real ThresholdJubjubSigner::verify,
	// which has this check built in (decaf377's primitive didn't, which is
	// why that backend needed the check pushed up to this layer). So this
	// test demonstrates the safety property holds end-to-end, at every
	// layer, rather than proving a lower layer alone is forgeable.
	z := big.NewInt(42)
	rPoint, err := jubjubtest.DerivePublicKey(scalarToLittleEndian32(z)) // R = z*G
	require.NoError(t, err)
	forgedSignature := append(append([]byte{}, rPoint...), scalarToLittleEndian32(z)...)

	ok, err := jubjubfrost.Verify(identityBytes, []byte("identity-key forgery"), forgedSignature)
	require.NoError(t, err)
	require.False(t, ok, "verify_core already rejects the identity public key by itself")

	err = verifyJubjubFROSTThresholdSignature(
		hex.EncodeToString(identityBytes),
		[]byte("identity-key forgery"),
		forgedSignature,
	)
	require.ErrorIs(t, err, types.ErrInvalidThresholdSignature)
	require.ErrorIs(t, rejectIdentityRingPublicKey(hex.EncodeToString(identityBytes)), types.ErrInvalidRing)
}

func TestMsgServer_FinalizeRingReshareByThresholdSignature_BLS12381(t *testing.T) {
	k, authKeeper, ctx := setupOrbisKeeper(t)
	ctx = ctx.
		WithValue(appparams.ExtractedDIDContextKey, testDID).
		WithBlockTime(time.Unix(int64(ringUpgradeBaseTime), 0))

	// BLS key pair — G1 public key, G2 signature scheme.
	ikm := make([]byte, 32)
	copy(ikm, "orbis-test-bls-ikm-000000000000")
	sk := blst.KeyGen(ikm)
	pk := new(blst.P1Affine).From(sk)
	ringPk := hex.EncodeToString(pk.Compress())

	creatorAddr, _ := testAccountWithPubKey(t, ctx, authKeeper)
	peer1Addr, peer1Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWBLSPeer1")
	peer2Addr, peer2Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWBLSPeer2")
	policyID := createOrbisRingPolicy(t, k, ctx, creatorAddr)

	createResp, err := k.CreateRing(ctx, &types.MsgCreateRing{
		Creator:      creatorAddr,
		PeerNodeKeys: []string{peer1Key, peer2Key},
		Threshold:    2,
		PssInterval:  types.MinPSSIntervalSeconds,
		PolicyId:     policyID,
	})
	require.NoError(t, err)
	ringID := createResp.RingId

	_, err = k.FinalizeRing(ctx, &types.MsgFinalizeRing{Creator: peer1Addr, RingId: ringID, RingPk: ringPk})
	require.NoError(t, err)
	_, err = k.FinalizeRing(ctx, &types.MsgFinalizeRing{Creator: peer2Addr, RingId: ringID, RingPk: ringPk})
	require.NoError(t, err)
	require.Equal(t, ringPk, k.GetRing(ctx, ringID).RingPk)

	peer3Addr, peer3Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWBLSPeer3")
	updatePeerNodeWhitelists(t, k, ctx, peer3Addr, peer3Key, []string{policyID}, nil)

	_, err = k.StartRingReshareByAcp(ctx, &types.MsgStartRingReshareByAcp{
		Creator:         creatorAddr,
		RingId:          ringID,
		NewPeerNodeKeys: []string{peer3Key},
		XNewThreshold: &types.MsgStartRingReshareByAcp_NewThreshold{
			NewThreshold: 1,
		},
	})
	require.NoError(t, err)

	_, err = k.ScheduleRingUpgradeByAcp(ctx, &types.MsgScheduleRingUpgradeByAcp{
		Creator:        creatorAddr,
		RingId:         ringID,
		NextVersion:    1,
		ActivationTime: ringUpgradeBaseTime + MinRingUpgradeLeadSeconds,
	})
	require.NoError(t, err)

	ring := k.GetRing(ctx, ringID)
	require.Equal(t, uint64(0), ring.UpgradeInfo.CurrentVersion)
	require.Equal(t, uint64(1), ring.UpgradeInfo.GetNextVersion())
	require.Equal(t, ringUpgradeBaseTime+MinRingUpgradeLeadSeconds, ring.UpgradeInfo.GetActivationTime())
	finalizedRing, err := ringForReshareFinalization(ring)
	require.NoError(t, err)
	signBytes, err := ringReshareFinalizeSignBytes(ctx.ChainID(), ring, finalizedRing)
	require.NoError(t, err)

	// Augmented BLS: the ring public key is prepended to the message.
	dst := []byte(bls12381G2SignatureAugDST)
	sig := new(blst.P2Affine).Sign(sk, signBytes, dst, pk.Compress())
	require.NotNil(t, sig)

	_, err = k.FinalizeRingReshareByThresholdSignature(ctx, &types.MsgFinalizeRingReshareByThresholdSignature{
		Creator:         creatorAddr,
		RingId:          ringID,
		SignatureScheme: ThresholdSignatureSchemeBLS12381G1PKG2SigAugV1,
		Signature:       sig.Compress(),
	})
	require.NoError(t, err)

	updated := k.GetRing(ctx, ringID)
	require.Equal(t, []string{peer3Key}, updated.PeerNodeKeys)
	require.Equal(t, uint32(1), updated.Threshold)
	require.Empty(t, updated.NewPeerNodeKeys)
	require.Nil(t, updated.XNewThreshold)
	require.Equal(t, uint64(ctx.BlockHeight()), updated.BlockNumberNonce)
	require.Equal(t, uint64(0), updated.UpgradeInfo.CurrentVersion)
	require.Equal(t, uint64(1), updated.UpgradeInfo.GetNextVersion())
	require.Equal(t, ringUpgradeBaseTime+MinRingUpgradeLeadSeconds, updated.UpgradeInfo.GetActivationTime())
}

func TestMsgServer_FinalizeRingReshareByThresholdSignature_JubjubFROST(t *testing.T) {
	k, authKeeper, ctx := setupOrbisKeeper(t)
	ctx = ctx.WithValue(appparams.ExtractedDIDContextKey, testDID)

	// Jubjub-FROST key pair — secret scalar, public key = x·G. The seed
	// string is wide-reduced (via the same Fr::from_bytes_wide operation
	// production code uses) rather than hand-reduced against a hardcoded
	// modulus constant, so this test never needs to know the scalar order.
	seedHash := sha512.Sum512([]byte("orbis-test-jubjub-secret-key0000"))
	secretBytes, err := jubjubtest.ReduceScalarWide(seedHash[:])
	require.NoError(t, err)
	secretScalar := bigIntFromLittleEndianScalar(secretBytes)

	ringPkBytes, err := jubjubPublicKeyBytes(secretScalar)
	require.NoError(t, err)
	ringPk := hex.EncodeToString(ringPkBytes)

	creatorAddr, _ := testAccountWithPubKey(t, ctx, authKeeper)
	peer1Addr, peer1Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWFROSTPeer1")
	peer2Addr, peer2Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWFROSTPeer2")
	policyID := createOrbisRingPolicy(t, k, ctx, creatorAddr)

	createResp, err := k.CreateRing(ctx, &types.MsgCreateRing{
		Creator:      creatorAddr,
		PeerNodeKeys: []string{peer1Key, peer2Key},
		Threshold:    2,
		PssInterval:  types.MinPSSIntervalSeconds,
		PolicyId:     policyID,
	})
	require.NoError(t, err)
	ringID := createResp.RingId

	_, err = k.FinalizeRing(ctx, &types.MsgFinalizeRing{Creator: peer1Addr, RingId: ringID, RingPk: ringPk})
	require.NoError(t, err)
	_, err = k.FinalizeRing(ctx, &types.MsgFinalizeRing{Creator: peer2Addr, RingId: ringID, RingPk: ringPk})
	require.NoError(t, err)
	require.Equal(t, ringPk, k.GetRing(ctx, ringID).RingPk)

	peer3Addr, peer3Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWFROSTPeer3")
	updatePeerNodeWhitelists(t, k, ctx, peer3Addr, peer3Key, []string{policyID}, nil)

	_, err = k.StartRingReshareByAcp(ctx, &types.MsgStartRingReshareByAcp{
		Creator:         creatorAddr,
		RingId:          ringID,
		NewPeerNodeKeys: []string{peer3Key},
		XNewThreshold: &types.MsgStartRingReshareByAcp_NewThreshold{
			NewThreshold: 1,
		},
	})
	require.NoError(t, err)

	ring := k.GetRing(ctx, ringID)
	finalizedRing, err := ringForReshareFinalization(ring)
	require.NoError(t, err)
	signBytes, err := ringReshareFinalizeSignBytes(ctx.ChainID(), ring, finalizedRing)
	require.NoError(t, err)

	sigBytes, err := jubjubSchnorrSign(secretScalar, ringPkBytes, signBytes)
	require.NoError(t, err)

	// Sanity-check our signing helper before submitting.
	ok, err := jubjubfrost.Verify(ringPkBytes, signBytes, sigBytes)
	require.NoError(t, err)
	require.True(t, ok)

	_, err = k.FinalizeRingReshareByThresholdSignature(ctx, &types.MsgFinalizeRingReshareByThresholdSignature{
		Creator:         creatorAddr,
		RingId:          ringID,
		SignatureScheme: ThresholdSignatureSchemeJubjubFROST,
		Signature:       sigBytes,
	})
	require.NoError(t, err)

	updated := k.GetRing(ctx, ringID)
	require.Equal(t, []string{peer3Key}, updated.PeerNodeKeys)
	require.Equal(t, uint32(1), updated.Threshold)
	require.Empty(t, updated.NewPeerNodeKeys)
	require.Nil(t, updated.XNewThreshold)
	require.Equal(t, uint64(ctx.BlockHeight()), updated.BlockNumberNonce)
}

// jubjubPublicKeyBytes returns the encoded public key point x·G for the
// given secret scalar. x must already be reduced mod the Jubjub scalar
// order (see jubjubtest.ReduceScalarWide).
func jubjubPublicKeyBytes(x *big.Int) ([]byte, error) {
	return jubjubtest.DerivePublicKey(scalarToLittleEndian32(x))
}

// jubjubSchnorrSign produces a signature (R || z) compatible with
// jubjubfrost.Verify, via a deterministic nonce derived from the secret and
// message (see jubjub-wasm's test_sign_core doc comment for why).
func jubjubSchnorrSign(x *big.Int, pubKeyBytes, msg []byte) ([]byte, error) {
	return jubjubtest.Sign(scalarToLittleEndian32(x), pubKeyBytes, msg)
}

// scalarToLittleEndian32 encodes a big.Int as a 32-byte little-endian scalar.
func scalarToLittleEndian32(x *big.Int) []byte {
	be := x.Bytes()
	le := make([]byte, 32)
	for i, b := range be {
		le[len(be)-1-i] = b
	}
	return le
}

// bigIntFromLittleEndianScalar interprets a fixed-width 32-byte
// little-endian scalar as a big.Int, for test-only modular arithmetic.
func bigIntFromLittleEndianScalar(s []byte) *big.Int {
	be := make([]byte, len(s))
	for i, b := range s {
		be[len(s)-1-i] = b
	}
	return new(big.Int).SetBytes(be)
}
