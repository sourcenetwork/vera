package keeper

import (
	"crypto/sha256"
	"encoding/hex"
	"math/big"
	"testing"
	"time"

	decaf377 "github.com/mizufinance/decaf377-go"
	"github.com/mizufinance/decaf377-go/orbisfrost"
	"github.com/stretchr/testify/require"
	blst "github.com/supranational/blst/bindings/go"

	appparams "github.com/sourcenetwork/sourcehub/app/params"
	"github.com/sourcenetwork/sourcehub/x/orbis/types"
)

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

	_, err = k.UpdateRingByAcp(ctx, &types.MsgUpdateRingByAcp{
		Creator:         creatorAddr,
		RingId:          ringID,
		NewPeerNodeKeys: []string{peer3Key},
		XNewThreshold: &types.MsgUpdateRingByAcp_NewThreshold{
			NewThreshold: 1,
		},
	})
	require.NoError(t, err)

	_, err = k.UpdateRingByAcp(ctx, &types.MsgUpdateRingByAcp{
		Creator: creatorAddr,
		RingId:  ringID,
		XNextVersion: &types.MsgUpdateRingByAcp_NextVersion{
			NextVersion: 1,
		},
		XActivationTime: &types.MsgUpdateRingByAcp_ActivationTime{
			ActivationTime: ringUpgradeBaseTime + MinRingUpgradeLeadSeconds,
		},
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

	dst := []byte("BLS_SIG_BLS12381G2_XMD:SHA-256_SSWU_RO_NUL_")
	sig := new(blst.P2Affine).Sign(sk, signBytes, dst)
	require.NotNil(t, sig)

	_, err = k.FinalizeRingReshareByThresholdSignature(ctx, &types.MsgFinalizeRingReshareByThresholdSignature{
		Creator:         creatorAddr,
		RingId:          ringID,
		SignatureScheme: ThresholdSignatureSchemeBLS12381G1PKG2SigNUL,
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

func TestMsgServer_FinalizeRingReshareByThresholdSignature_Decaf377FROST(t *testing.T) {
	k, authKeeper, ctx := setupOrbisKeeper(t)
	ctx = ctx.WithValue(appparams.ExtractedDIDContextKey, testDID)

	// Decaf377-FROST key pair — secret scalar, public key = x·G.
	secretScalar := new(big.Int).SetBytes([]byte("orbis-test-decaf377-secret-key00"))
	secretScalar.Mod(secretScalar, decaf377.ScalarOrder())

	ringPkBytes, err := decaf377PublicKeyBytes(secretScalar)
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

	_, err = k.UpdateRingByAcp(ctx, &types.MsgUpdateRingByAcp{
		Creator:         creatorAddr,
		RingId:          ringID,
		NewPeerNodeKeys: []string{peer3Key},
		XNewThreshold: &types.MsgUpdateRingByAcp_NewThreshold{
			NewThreshold: 1,
		},
	})
	require.NoError(t, err)

	ring := k.GetRing(ctx, ringID)
	finalizedRing, err := ringForReshareFinalization(ring)
	require.NoError(t, err)
	signBytes, err := ringReshareFinalizeSignBytes(ctx.ChainID(), ring, finalizedRing)
	require.NoError(t, err)

	sigBytes, err := decaf377SchnorrSign(secretScalar, ringPkBytes, signBytes)
	require.NoError(t, err)

	// Sanity-check our signing helper before submitting.
	ok, err := orbisfrost.Verify(ringPkBytes, signBytes, sigBytes)
	require.NoError(t, err)
	require.True(t, ok)

	_, err = k.FinalizeRingReshareByThresholdSignature(ctx, &types.MsgFinalizeRingReshareByThresholdSignature{
		Creator:         creatorAddr,
		RingId:          ringID,
		SignatureScheme: ThresholdSignatureSchemeDecaf377FROST,
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

// decaf377PublicKeyBytes returns the encoded public key point x·G for the given secret scalar.
func decaf377PublicKeyBytes(x *big.Int) ([]byte, error) {
	g, err := decaf377.Generator()
	if err != nil {
		return nil, err
	}
	pub, err := decaf377.ScalarMul(g, x)
	if err != nil {
		return nil, err
	}
	return decaf377.Encode(pub)
}

// decaf377SchnorrSign produces a signature (R || z) compatible with orbisfrost.Verify.
// It uses a deterministic nonce derived from the secret and message.
func decaf377SchnorrSign(x *big.Int, pubKeyBytes, msg []byte) ([]byte, error) {
	g, err := decaf377.Generator()
	if err != nil {
		return nil, err
	}

	// Deterministic nonce: k = H(x || msg) mod order
	nonceInput := append(scalarToLittleEndian32(x), msg...)
	nonceHash := sha256.Sum256(nonceInput)
	k := decaf377.ScalarFromUniformBytes(nonceHash[:])

	// R = k·G
	rPoint, err := decaf377.ScalarMul(g, k)
	if err != nil {
		return nil, err
	}
	rBytes, err := decaf377.Encode(rPoint)
	if err != nil {
		return nil, err
	}

	// c = H(domain || R || pubKey || msg)
	h := sha256.New()
	h.Write([]byte(orbisfrost.ChallengeDomain))
	h.Write(rBytes)
	h.Write(pubKeyBytes)
	h.Write(msg)
	c := decaf377.ScalarFromUniformBytes(h.Sum(nil))

	// z = (k + c·x) mod order
	order := decaf377.ScalarOrder()
	z := new(big.Int).Mul(c, x)
	z.Add(z, k)
	z.Mod(z, order)

	return append(rBytes, scalarToLittleEndian32(z)...), nil
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
