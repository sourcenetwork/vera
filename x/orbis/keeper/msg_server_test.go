package keeper_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	appparams "github.com/sourcenetwork/sourcehub/app/params"
	keepertestutil "github.com/sourcenetwork/sourcehub/testutil/keeper"
	"github.com/sourcenetwork/sourcehub/x/orbis/types"
)

const testDID = "did:example:orbis-creator"

func TestMsgServer_CreateRingStoreDocumentAndKeyDerivation(t *testing.T) {
	k, ctx := keepertestutil.OrbisKeeper(t)
	ctx = ctx.WithValue(appparams.ExtractedDIDContextKey, testDID)

	creator := testAddress()
	namespace := "vault"
	namespaceID := types.GetNamespaceID(namespace)
	pssInterval := uint64(600)

	createRingResp, err := k.CreateRing(ctx, &types.MsgCreateRing{
		Creator:   creator,
		Namespace: namespace,
		RingPk:    "ring-pk",
		PeerIds:   []string{"peer-1", "peer-2", "peer-3"},
		Threshold: 2,
		XPssInterval: &types.MsgCreateRing_PssInterval{
			PssInterval: pssInterval,
		},
		PolicyId: "policy-ring",
		Artifact: "ring-artifact",
	})
	require.NoError(t, err)

	ring := k.GetRing(ctx, createRingResp.RingId)
	require.NotNil(t, ring)
	require.Equal(t, namespaceID, ring.Namespace)
	require.Equal(t, testDID, ring.CreatorDid)
	require.Equal(t, "ring-pk", ring.RingPk)
	require.Equal(t, []string{"peer-1", "peer-2", "peer-3"}, ring.PeerIds)
	require.Equal(t, uint32(2), ring.Threshold)
	require.NotNil(t, ring.XPssInterval)
	require.Equal(t, pssInterval, ring.GetPssInterval())

	_, err = k.CreateRing(ctx, &types.MsgCreateRing{
		Creator:   creator,
		Namespace: namespace,
		RingPk:    "ring-pk",
		PeerIds:   []string{"peer-1", "peer-2", "peer-3"},
		Threshold: 2,
		XPssInterval: &types.MsgCreateRing_PssInterval{
			PssInterval: pssInterval,
		},
		PolicyId: "policy-ring",
	})
	require.ErrorIs(t, err, types.ErrRingAlreadyExists)

	tier := "gold"
	timestamp := uint64(42)
	storeDocumentMsg := &types.MsgStoreDocument{
		Creator:    creator,
		Namespace:  namespace,
		RingId:     createRingResp.RingId,
		Document:   "ciphertext",
		Proof:      "proof",
		PolicyId:   "policy-doc",
		Resource:   "secret",
		Permission: "decrypt",
		XTier: &types.MsgStoreDocument_Tier{
			Tier: tier,
		},
		XTimestamp: &types.MsgStoreDocument_Timestamp{
			Timestamp: timestamp,
		},
	}
	storeDocumentResp, err := k.StoreDocument(ctx, storeDocumentMsg)
	require.NoError(t, err)
	require.Equal(
		t,
		types.GenerateDocumentID(namespaceID, createRingResp.RingId, "ciphertext", "proof", "policy-doc", "secret", "decrypt", &tier, &timestamp),
		storeDocumentResp.DocumentId,
	)

	document := k.GetDocument(ctx, namespaceID, storeDocumentResp.DocumentId)
	require.NotNil(t, document)
	require.Equal(t, testDID, document.CreatorDid)
	require.NotNil(t, document.XTier)
	require.NotNil(t, document.XTimestamp)
	require.Equal(t, tier, document.GetTier())
	require.Equal(t, timestamp, document.GetTimestamp())

	_, err = k.StoreDocument(ctx, storeDocumentMsg)
	require.ErrorIs(t, err, types.ErrDocumentAlreadyExists)

	storeKeyDerivationMsg := &types.MsgStoreKeyDerivation{
		Creator:    creator,
		Namespace:  namespace,
		RingId:     createRingResp.RingId,
		Derivation: "m/0/1",
		PolicyId:   "policy-derivation",
		Resource:   "derived-key",
		Permission: "derive",
	}
	storeKeyDerivationResp, err := k.StoreKeyDerivation(ctx, storeKeyDerivationMsg)
	require.NoError(t, err)
	require.Equal(
		t,
		types.GenerateKeyDerivationID(namespaceID, createRingResp.RingId, "m/0/1", "policy-derivation", "derived-key", "derive"),
		storeKeyDerivationResp.KeyDerivationId,
	)

	keyDerivation := k.GetKeyDerivation(ctx, namespaceID, storeKeyDerivationResp.KeyDerivationId)
	require.NotNil(t, keyDerivation)
	require.Equal(t, testDID, keyDerivation.CreatorDid)
	require.Equal(t, "m/0/1", keyDerivation.Derivation)

	_, err = k.StoreKeyDerivation(ctx, storeKeyDerivationMsg)
	require.ErrorIs(t, err, types.ErrKeyDerivationAlreadyExists)
}

func TestMsgServer_AbsentOptionalFieldsAreTreatedAsNone(t *testing.T) {
	k, ctx := keepertestutil.OrbisKeeper(t)
	ctx = ctx.WithValue(appparams.ExtractedDIDContextKey, testDID)

	creator := testAddress()
	namespace := "vault"
	namespaceID := types.GetNamespaceID(namespace)

	createRingResp, err := k.CreateRing(ctx, &types.MsgCreateRing{
		Creator:   creator,
		Namespace: namespace,
		RingPk:    "ring-pk",
		PeerIds:   []string{"peer-1", "peer-2"},
		Threshold: 1,
	})
	require.NoError(t, err)
	require.Equal(
		t,
		types.GenerateRingID(namespaceID, "ring-pk", []string{"peer-1", "peer-2"}, 1, nil, ""),
		createRingResp.RingId,
	)

	storeDocumentResp, err := k.StoreDocument(ctx, &types.MsgStoreDocument{
		Creator:    creator,
		Namespace:  namespace,
		RingId:     createRingResp.RingId,
		Document:   "ciphertext",
		Proof:      "proof",
		PolicyId:   "policy-doc",
		Resource:   "secret",
		Permission: "decrypt",
	})
	require.NoError(t, err)
	require.Equal(
		t,
		types.GenerateDocumentID(namespaceID, createRingResp.RingId, "ciphertext", "proof", "policy-doc", "secret", "decrypt", nil, nil),
		storeDocumentResp.DocumentId,
	)
}

func TestMsgServer_ZeroPSSIntervalIsPreservedWhenPresent(t *testing.T) {
	k, ctx := keepertestutil.OrbisKeeper(t)
	ctx = ctx.WithValue(appparams.ExtractedDIDContextKey, testDID)

	creator := testAddress()
	namespace := "vault"
	namespaceID := types.GetNamespaceID(namespace)
	pssInterval := uint64(0)

	createRingResp, err := k.CreateRing(ctx, &types.MsgCreateRing{
		Creator:   creator,
		Namespace: namespace,
		RingPk:    "ring-pk",
		PeerIds:   []string{"peer-1", "peer-2"},
		Threshold: 1,
		XPssInterval: &types.MsgCreateRing_PssInterval{
			PssInterval: pssInterval,
		},
	})
	require.NoError(t, err)
	require.Equal(
		t,
		types.GenerateRingID(namespaceID, "ring-pk", []string{"peer-1", "peer-2"}, 1, &pssInterval, ""),
		createRingResp.RingId,
	)

	ring := k.GetRing(ctx, createRingResp.RingId)
	require.NotNil(t, ring)
	require.NotNil(t, ring.XPssInterval)
	require.Equal(t, pssInterval, ring.GetPssInterval())
}

func TestMsgServer_UpdateRingByAcpRequiresPolicy(t *testing.T) {
	k, ctx := keepertestutil.OrbisKeeper(t)
	ctx = ctx.WithValue(appparams.ExtractedDIDContextKey, testDID)

	createRingResp, err := k.CreateRing(ctx, &types.MsgCreateRing{
		Creator:   testAddress(),
		Namespace: "vault",
		RingPk:    "ring-pk",
		PeerIds:   []string{"peer-1", "peer-2"},
		Threshold: 1,
	})
	require.NoError(t, err)

	_, err = k.UpdateRingByAcp(ctx, &types.MsgUpdateRingByAcp{
		Creator:    testAddress(),
		RingId:     createRingResp.RingId,
		NewPeerIds: []string{"peer-3", "peer-4"},
		XNewThreshold: &types.MsgUpdateRingByAcp_NewThreshold{
			NewThreshold: 1,
		},
	})
	require.ErrorIs(t, err, types.ErrRingMissingPolicyId)
}

func TestMsgServer_FinalizeRingReshareRequiresPendingUpdate(t *testing.T) {
	k, ctx := keepertestutil.OrbisKeeper(t)
	ctx = ctx.WithValue(appparams.ExtractedDIDContextKey, testDID)

	createRingResp, err := k.CreateRing(ctx, &types.MsgCreateRing{
		Creator:   testAddress(),
		Namespace: "vault",
		RingPk:    "ring-pk",
		PeerIds:   []string{"peer-1", "peer-2"},
		Threshold: 1,
	})
	require.NoError(t, err)

	_, err = k.FinalizeRingReshareByThresholdSignature(ctx, &types.MsgFinalizeRingReshareByThresholdSignature{
		Creator:         testAddress(),
		RingId:          createRingResp.RingId,
		SignatureScheme: "bls12_381",
		Signature:       []byte("signature"),
	})
	require.ErrorIs(t, err, types.ErrInvalidRing)
	require.ErrorContains(t, err, "missing new_peer_ids or new_threshold")
}

func testAddress() string {
	return sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()
}
