package keeper_test

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	"github.com/stretchr/testify/require"

	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	appparams "github.com/sourcenetwork/sourcehub/app/params"
	keepertestutil "github.com/sourcenetwork/sourcehub/testutil/keeper"
	"github.com/sourcenetwork/sourcehub/x/acp/capability"
	acptypes "github.com/sourcenetwork/sourcehub/x/acp/types"
	"github.com/sourcenetwork/sourcehub/x/orbis/keeper"
	"github.com/sourcenetwork/sourcehub/x/orbis/types"
)

const testDID = "did:example:orbis-creator"

func TestMsgServer_CreateRingStoreDocumentAndKeyDerivation(t *testing.T) {
	k, authKeeper, ctx := keepertestutil.OrbisKeeperFull(t)
	ctx = ctx.WithValue(appparams.ExtractedDIDContextKey, testDID)

	creatorAddr, _ := testAccountWithPubKey(t, ctx, authKeeper)
	namespace := "vault"
	namespaceID := types.GetNamespaceID(namespace)
	setupNamespaceWithMember(t, k, ctx, namespaceID, testDID, creatorAddr)

	peer1Addr, peer1Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer1")
	peer2Addr, peer2Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer2")
	_, peer3Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer3")

	pssInterval := uint64(600)
	createRingResp, err := k.CreateRing(ctx, &types.MsgCreateRing{
		Creator:      creatorAddr,
		Namespace:    namespace,
		PeerNodeKeys: []string{peer1Key, peer2Key, peer3Key},
		Threshold:    2,
		XPssInterval: &types.MsgCreateRing_PssInterval{
			PssInterval: pssInterval,
		},
		Artifact: "ring-artifact",
	})
	require.NoError(t, err)

	ring := k.GetRing(ctx, createRingResp.RingId)
	require.NotNil(t, ring)
	require.Equal(t, namespaceID, ring.Namespace)
	require.Equal(t, testDID, ring.CreatorDid)
	require.Empty(t, ring.RingPk)
	require.Equal(t, []string{peer1Key, peer2Key, peer3Key}, ring.PeerNodeKeys)
	require.Equal(t, uint32(2), ring.Threshold)
	require.NotNil(t, ring.XPssInterval)
	require.Equal(t, pssInterval, ring.GetPssInterval())

	// peer1 submits first confirmation; threshold=2 so not finalized yet
	_, err = k.FinalizeRing(ctx, &types.MsgFinalizeRing{
		Creator: peer1Addr,
		RingId:  createRingResp.RingId,
		RingPk:  "ring-pk",
	})
	require.NoError(t, err)
	require.Empty(t, k.GetRing(ctx, createRingResp.RingId).RingPk)

	// peer2 submits second confirmation; threshold met → finalized
	_, err = k.FinalizeRing(ctx, &types.MsgFinalizeRing{
		Creator: peer2Addr,
		RingId:  createRingResp.RingId,
		RingPk:  "ring-pk",
	})
	require.NoError(t, err)
	require.Equal(t, "ring-pk", k.GetRing(ctx, createRingResp.RingId).RingPk)

	_, err = k.FinalizeRing(ctx, &types.MsgFinalizeRing{
		Creator: peer1Addr,
		RingId:  createRingResp.RingId,
		RingPk:  "ring-pk-2",
	})
	require.ErrorIs(t, err, types.ErrRingAlreadyFinalized)

	_, err = k.CreateRing(ctx, &types.MsgCreateRing{
		Creator:      creatorAddr,
		Namespace:    namespace,
		PeerNodeKeys: []string{peer1Key, peer2Key, peer3Key},
		Threshold:    2,
		XPssInterval: &types.MsgCreateRing_PssInterval{
			PssInterval: pssInterval,
		},
	})
	require.ErrorIs(t, err, types.ErrRingAlreadyExists)

	tier := "gold"
	timestamp := uint64(42)
	storeDocumentMsg := &types.MsgStoreDocument{
		Creator:    creatorAddr,
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
		Creator:    creatorAddr,
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

func TestMsgServer_FinalizeRing_UnauthorizedNonPeer(t *testing.T) {
	k, authKeeper, ctx := keepertestutil.OrbisKeeperFull(t)
	ctx = ctx.WithValue(appparams.ExtractedDIDContextKey, testDID)

	creatorAddr, _ := testAccountWithPubKey(t, ctx, authKeeper)
	namespaceID := types.GetNamespaceID("vault")
	setupNamespaceWithMember(t, k, ctx, namespaceID, testDID, creatorAddr)

	_, peer1Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer1")
	_, peer2Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer2")

	createRingResp, err := k.CreateRing(ctx, &types.MsgCreateRing{
		Creator:      creatorAddr,
		Namespace:    "vault",
		PeerNodeKeys: []string{peer1Key, peer2Key},
		Threshold:    1,
	})
	require.NoError(t, err)

	outsiderAddr, _ := testAccountWithPubKey(t, ctx, authKeeper)
	_, err = k.FinalizeRing(ctx, &types.MsgFinalizeRing{
		Creator: outsiderAddr,
		RingId:  createRingResp.RingId,
		RingPk:  "ring-pk",
	})
	require.ErrorIs(t, err, types.ErrInvalidRingFinalizer)
}

func TestMsgServer_FinalizeRing_ThresholdRequiresMultiple(t *testing.T) {
	k, authKeeper, ctx := keepertestutil.OrbisKeeperFull(t)
	ctx = ctx.WithValue(appparams.ExtractedDIDContextKey, testDID)

	creatorAddr, _ := testAccountWithPubKey(t, ctx, authKeeper)
	namespaceID := types.GetNamespaceID("vault")
	setupNamespaceWithMember(t, k, ctx, namespaceID, testDID, creatorAddr)

	peer1Addr, peer1Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer1")
	peer2Addr, peer2Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer2")
	peer3Addr, peer3Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer3")

	createRingResp, err := k.CreateRing(ctx, &types.MsgCreateRing{
		Creator:      creatorAddr,
		Namespace:    "vault",
		PeerNodeKeys: []string{peer1Key, peer2Key, peer3Key},
		Threshold:    2,
	})
	require.NoError(t, err)
	ringID := createRingResp.RingId

	// first confirmation — not finalized yet
	_, err = k.FinalizeRing(ctx, &types.MsgFinalizeRing{Creator: peer1Addr, RingId: ringID, RingPk: "agreed-pk"})
	require.NoError(t, err)
	require.Empty(t, k.GetRing(ctx, ringID).RingPk)
	require.Len(t, k.GetRing(ctx, ringID).Confirmations, 1)

	// second confirmation — threshold=2 met → finalized, confirmations cleared
	_, err = k.FinalizeRing(ctx, &types.MsgFinalizeRing{Creator: peer2Addr, RingId: ringID, RingPk: "agreed-pk"})
	require.NoError(t, err)
	ring := k.GetRing(ctx, ringID)
	require.Equal(t, "agreed-pk", ring.RingPk)
	require.Empty(t, ring.Confirmations)

	// third peer cannot confirm a finalized ring
	_, err = k.FinalizeRing(ctx, &types.MsgFinalizeRing{Creator: peer3Addr, RingId: ringID, RingPk: "agreed-pk"})
	require.ErrorIs(t, err, types.ErrRingAlreadyFinalized)
}

func TestMsgServer_FinalizeRing_PkConflictDeletesRing(t *testing.T) {
	k, authKeeper, ctx := keepertestutil.OrbisKeeperFull(t)
	ctx = ctx.WithValue(appparams.ExtractedDIDContextKey, testDID)

	creatorAddr, _ := testAccountWithPubKey(t, ctx, authKeeper)
	namespaceID := types.GetNamespaceID("vault")
	setupNamespaceWithMember(t, k, ctx, namespaceID, testDID, creatorAddr)

	peer1Addr, peer1Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer1")
	peer2Addr, peer2Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer2")

	createRingResp, err := k.CreateRing(ctx, &types.MsgCreateRing{
		Creator:      creatorAddr,
		Namespace:    "vault",
		PeerNodeKeys: []string{peer1Key, peer2Key},
		Threshold:    2,
	})
	require.NoError(t, err)
	ringID := createRingResp.RingId

	_, err = k.FinalizeRing(ctx, &types.MsgFinalizeRing{Creator: peer1Addr, RingId: ringID, RingPk: "pk-version-A"})
	require.NoError(t, err)

	// peer2 disagrees on the ring_pk → conflict, ring deleted
	_, err = k.FinalizeRing(ctx, &types.MsgFinalizeRing{Creator: peer2Addr, RingId: ringID, RingPk: "pk-version-B"})
	require.ErrorIs(t, err, types.ErrRingPkConflict)
	require.Nil(t, k.GetRing(ctx, ringID))
}

func TestMsgServer_FinalizeRing_DuplicateConfirmationRejected(t *testing.T) {
	k, authKeeper, ctx := keepertestutil.OrbisKeeperFull(t)
	ctx = ctx.WithValue(appparams.ExtractedDIDContextKey, testDID)

	creatorAddr, _ := testAccountWithPubKey(t, ctx, authKeeper)
	namespaceID := types.GetNamespaceID("vault")
	setupNamespaceWithMember(t, k, ctx, namespaceID, testDID, creatorAddr)

	peer1Addr, peer1Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer1")
	_, peer2Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer2")

	createRingResp, err := k.CreateRing(ctx, &types.MsgCreateRing{
		Creator:      creatorAddr,
		Namespace:    "vault",
		PeerNodeKeys: []string{peer1Key, peer2Key},
		Threshold:    2,
	})
	require.NoError(t, err)
	ringID := createRingResp.RingId

	_, err = k.FinalizeRing(ctx, &types.MsgFinalizeRing{Creator: peer1Addr, RingId: ringID, RingPk: "ring-pk"})
	require.NoError(t, err)

	_, err = k.FinalizeRing(ctx, &types.MsgFinalizeRing{Creator: peer1Addr, RingId: ringID, RingPk: "ring-pk"})
	require.ErrorIs(t, err, types.ErrDuplicateConfirmation)
}

func TestMsgServer_AbsentOptionalFieldsAreTreatedAsNone(t *testing.T) {
	k, authKeeper, ctx := keepertestutil.OrbisKeeperFull(t)
	ctx = ctx.WithValue(appparams.ExtractedDIDContextKey, testDID)

	creatorAddr, _ := testAccountWithPubKey(t, ctx, authKeeper)
	namespace := "vault"
	namespaceID := types.GetNamespaceID(namespace)
	setupNamespaceWithMember(t, k, ctx, namespaceID, testDID, creatorAddr)

	_, peer1Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer1")
	_, peer2Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer2")

	createRingResp, err := k.CreateRing(ctx, &types.MsgCreateRing{
		Creator:      creatorAddr,
		Namespace:    namespace,
		PeerNodeKeys: []string{peer1Key, peer2Key},
		Threshold:    1,
	})
	require.NoError(t, err)
	require.Equal(
		t,
		types.GenerateRingID(namespaceID, []string{peer1Key, peer2Key}, 1, nil, ""),
		createRingResp.RingId,
	)

	storeDocumentResp, err := k.StoreDocument(ctx, &types.MsgStoreDocument{
		Creator:    creatorAddr,
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
	k, authKeeper, ctx := keepertestutil.OrbisKeeperFull(t)
	ctx = ctx.WithValue(appparams.ExtractedDIDContextKey, testDID)

	creatorAddr, _ := testAccountWithPubKey(t, ctx, authKeeper)
	namespace := "vault"
	namespaceID := types.GetNamespaceID(namespace)
	setupNamespaceWithMember(t, k, ctx, namespaceID, testDID, creatorAddr)
	pssInterval := uint64(0)

	_, peer1Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer1")
	_, peer2Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer2")

	createRingResp, err := k.CreateRing(ctx, &types.MsgCreateRing{
		Creator:      creatorAddr,
		Namespace:    namespace,
		PeerNodeKeys: []string{peer1Key, peer2Key},
		Threshold:    1,
		XPssInterval: &types.MsgCreateRing_PssInterval{
			PssInterval: pssInterval,
		},
	})
	require.NoError(t, err)
	require.Equal(
		t,
		types.GenerateRingID(namespaceID, []string{peer1Key, peer2Key}, 1, &pssInterval, ""),
		createRingResp.RingId,
	)

	ring := k.GetRing(ctx, createRingResp.RingId)
	require.NotNil(t, ring)
	require.NotNil(t, ring.XPssInterval)
	require.Equal(t, pssInterval, ring.GetPssInterval())
}

func TestMsgServer_UpdateRingByAcpRequiresPolicy(t *testing.T) {
	k, authKeeper, ctx := keepertestutil.OrbisKeeperFull(t)
	ctx = ctx.WithValue(appparams.ExtractedDIDContextKey, testDID)

	creatorAddr, _ := testAccountWithPubKey(t, ctx, authKeeper)
	setupNamespaceWithMember(t, k, ctx, types.GetNamespaceID("vault"), testDID, creatorAddr)

	_, peer1Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer1")
	_, peer2Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer2")

	createRingResp, err := k.CreateRing(ctx, &types.MsgCreateRing{
		Creator:      creatorAddr,
		Namespace:    "vault",
		PeerNodeKeys: []string{peer1Key, peer2Key},
		Threshold:    1,
	})
	require.NoError(t, err)

	_, peer3Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer3")
	_, peer4Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer4")
	outsiderAddr, _ := testAccountWithPubKey(t, ctx, authKeeper)
	_, err = k.UpdateRingByAcp(ctx, &types.MsgUpdateRingByAcp{
		Creator:        outsiderAddr,
		RingId:         createRingResp.RingId,
		NewPeerNodeKeys: []string{peer3Key, peer4Key},
		XNewThreshold: &types.MsgUpdateRingByAcp_NewThreshold{
			NewThreshold: 1,
		},
	})
	require.ErrorIs(t, err, types.ErrRingMissingPolicyId)
}

func TestMsgServer_FinalizeRingReshareRequiresPendingUpdate(t *testing.T) {
	k, authKeeper, ctx := keepertestutil.OrbisKeeperFull(t)
	ctx = ctx.WithValue(appparams.ExtractedDIDContextKey, testDID)

	creatorAddr, _ := testAccountWithPubKey(t, ctx, authKeeper)
	setupNamespaceWithMember(t, k, ctx, types.GetNamespaceID("vault"), testDID, creatorAddr)

	_, peer1Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer1")
	_, peer2Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer2")

	createRingResp, err := k.CreateRing(ctx, &types.MsgCreateRing{
		Creator:      creatorAddr,
		Namespace:    "vault",
		PeerNodeKeys: []string{peer1Key, peer2Key},
		Threshold:    1,
	})
	require.NoError(t, err)

	outsiderAddr, _ := testAccountWithPubKey(t, ctx, authKeeper)
	_, err = k.FinalizeRingReshareByThresholdSignature(ctx, &types.MsgFinalizeRingReshareByThresholdSignature{
		Creator:         outsiderAddr,
		RingId:          createRingResp.RingId,
		SignatureScheme: "bls12_381",
		Signature:       []byte("signature"),
	})
	require.ErrorIs(t, err, types.ErrInvalidRing)
	require.ErrorContains(t, err, "missing new_peer_node_keys or new_threshold")
}

func TestMsgServer_CreateNodeInfo(t *testing.T) {
	k, authKeeper, ctx := keepertestutil.OrbisKeeperFull(t)
	ctx = ctx.WithValue(appparams.ExtractedDIDContextKey, testDID)

	nodeAddr, nodePubKeyHex := testAccountWithPubKey(t, ctx, authKeeper)
	_, controllerPubKeyHex := testAccountWithPubKey(t, ctx, authKeeper)

	_, err := k.CreateNodeInfo(ctx, &types.MsgCreateNodeInfo{
		Creator:       nodeAddr,
		PeerId:        "12D3KooWExamplePeerID",
		ControllerKey: controllerPubKeyHex,
	})
	require.NoError(t, err)

	nodeInfo := k.GetNodeInfo(ctx, nodePubKeyHex)
	require.NotNil(t, nodeInfo)
	require.Equal(t, "12D3KooWExamplePeerID", nodeInfo.PeerId)
	require.Equal(t, controllerPubKeyHex, nodeInfo.ControllerKey)
	require.Empty(t, nodeInfo.WhitelistedNamespaces)
	require.Empty(t, nodeInfo.WhitelistedRingIds)
}

func TestMsgServer_CreateNodeInfo_WithWhitelists(t *testing.T) {
	k, authKeeper, ctx := keepertestutil.OrbisKeeperFull(t)
	ctx = ctx.WithValue(appparams.ExtractedDIDContextKey, testDID)

	nodeAddr, nodePubKeyHex := testAccountWithPubKey(t, ctx, authKeeper)
	_, controllerPubKeyHex := testAccountWithPubKey(t, ctx, authKeeper)

	_, err := k.CreateNodeInfo(ctx, &types.MsgCreateNodeInfo{
		Creator:               nodeAddr,
		PeerId:                "peer-1",
		ControllerKey:         controllerPubKeyHex,
		WhitelistedNamespaces: []string{"orbis/ns-a", "orbis/ns-b"},
		WhitelistedRingIds:    []string{"ring-1"},
	})
	require.NoError(t, err)

	nodeInfo := k.GetNodeInfo(ctx, nodePubKeyHex)
	require.NotNil(t, nodeInfo)
	require.Equal(t, []string{"orbis/ns-a", "orbis/ns-b"}, nodeInfo.WhitelistedNamespaces)
	require.Equal(t, []string{"ring-1"}, nodeInfo.WhitelistedRingIds)
}

func TestMsgServer_CreateNodeInfo_AlreadyExists(t *testing.T) {
	k, authKeeper, ctx := keepertestutil.OrbisKeeperFull(t)
	ctx = ctx.WithValue(appparams.ExtractedDIDContextKey, testDID)

	nodeAddr, _ := testAccountWithPubKey(t, ctx, authKeeper)
	_, controllerPubKeyHex := testAccountWithPubKey(t, ctx, authKeeper)

	msg := &types.MsgCreateNodeInfo{
		Creator:       nodeAddr,
		PeerId:        "peer-1",
		ControllerKey: controllerPubKeyHex,
	}

	_, err := k.CreateNodeInfo(ctx, msg)
	require.NoError(t, err)

	_, err = k.CreateNodeInfo(ctx, msg)
	require.ErrorIs(t, err, types.ErrNodeInfoAlreadyExists)
}

func TestMsgServer_UpdateNodeInfo(t *testing.T) {
	k, authKeeper, ctx := keepertestutil.OrbisKeeperFull(t)
	ctx = ctx.WithValue(appparams.ExtractedDIDContextKey, testDID)

	nodeAddr, nodePubKeyHex := testAccountWithPubKey(t, ctx, authKeeper)
	controllerAddr, controllerPubKeyHex := testAccountWithPubKey(t, ctx, authKeeper)

	_, err := k.CreateNodeInfo(ctx, &types.MsgCreateNodeInfo{
		Creator:               nodeAddr,
		PeerId:                "peer-original",
		ControllerKey:         controllerPubKeyHex,
		WhitelistedNamespaces: []string{"orbis/ns-a"},
	})
	require.NoError(t, err)

	_, err = k.UpdateNodeInfo(ctx, &types.MsgUpdateNodeInfo{
		Creator:               controllerAddr,
		NodeKey:               nodePubKeyHex,
		WhitelistedNamespaces: []string{"orbis/ns-b", "orbis/ns-c"},
		WhitelistedRingIds:    []string{"ring-1", "ring-2"},
	})
	require.NoError(t, err)

	nodeInfo := k.GetNodeInfo(ctx, nodePubKeyHex)
	require.NotNil(t, nodeInfo)
	require.Equal(t, "peer-original", nodeInfo.PeerId)
	require.Equal(t, []string{"orbis/ns-b", "orbis/ns-c"}, nodeInfo.WhitelistedNamespaces)
	require.Equal(t, []string{"ring-1", "ring-2"}, nodeInfo.WhitelistedRingIds)
}

func TestMsgServer_UpdateNodeInfo_UpdatePeerId(t *testing.T) {
	k, authKeeper, ctx := keepertestutil.OrbisKeeperFull(t)
	ctx = ctx.WithValue(appparams.ExtractedDIDContextKey, testDID)

	nodeAddr, nodePubKeyHex := testAccountWithPubKey(t, ctx, authKeeper)
	controllerAddr, controllerPubKeyHex := testAccountWithPubKey(t, ctx, authKeeper)

	_, err := k.CreateNodeInfo(ctx, &types.MsgCreateNodeInfo{
		Creator:       nodeAddr,
		PeerId:        "peer-original",
		ControllerKey: controllerPubKeyHex,
	})
	require.NoError(t, err)

	_, err = k.UpdateNodeInfo(ctx, &types.MsgUpdateNodeInfo{
		Creator: controllerAddr,
		NodeKey: nodePubKeyHex,
		XPeerId: &types.MsgUpdateNodeInfo_PeerId{PeerId: "peer-updated"},
	})
	require.NoError(t, err)

	nodeInfo := k.GetNodeInfo(ctx, nodePubKeyHex)
	require.NotNil(t, nodeInfo)
	require.Equal(t, "peer-updated", nodeInfo.PeerId)
}

func TestMsgServer_UpdateNodeInfo_AbsentPeerIdIsNotCleared(t *testing.T) {
	k, authKeeper, ctx := keepertestutil.OrbisKeeperFull(t)
	ctx = ctx.WithValue(appparams.ExtractedDIDContextKey, testDID)

	nodeAddr, nodePubKeyHex := testAccountWithPubKey(t, ctx, authKeeper)
	controllerAddr, controllerPubKeyHex := testAccountWithPubKey(t, ctx, authKeeper)

	_, err := k.CreateNodeInfo(ctx, &types.MsgCreateNodeInfo{
		Creator:       nodeAddr,
		PeerId:        "peer-original",
		ControllerKey: controllerPubKeyHex,
	})
	require.NoError(t, err)

	_, err = k.UpdateNodeInfo(ctx, &types.MsgUpdateNodeInfo{
		Creator:            controllerAddr,
		NodeKey:            nodePubKeyHex,
		WhitelistedRingIds: []string{"ring-1"},
	})
	require.NoError(t, err)

	nodeInfo := k.GetNodeInfo(ctx, nodePubKeyHex)
	require.Equal(t, "peer-original", nodeInfo.PeerId)
}

func TestMsgServer_UpdateNodeInfo_NotFound(t *testing.T) {
	k, authKeeper, ctx := keepertestutil.OrbisKeeperFull(t)
	ctx = ctx.WithValue(appparams.ExtractedDIDContextKey, testDID)

	controllerAddr, _ := testAccountWithPubKey(t, ctx, authKeeper)

	_, err := k.UpdateNodeInfo(ctx, &types.MsgUpdateNodeInfo{
		Creator: controllerAddr,
		NodeKey: "nonexistent-key",
	})
	require.ErrorIs(t, err, types.ErrNodeInfoNotFound)
}

func TestMsgServer_UpdateNodeInfo_Unauthorized(t *testing.T) {
	k, authKeeper, ctx := keepertestutil.OrbisKeeperFull(t)
	ctx = ctx.WithValue(appparams.ExtractedDIDContextKey, testDID)

	nodeAddr, nodePubKeyHex := testAccountWithPubKey(t, ctx, authKeeper)
	_, controllerPubKeyHex := testAccountWithPubKey(t, ctx, authKeeper)
	wrongAddr, _ := testAccountWithPubKey(t, ctx, authKeeper)

	_, err := k.CreateNodeInfo(ctx, &types.MsgCreateNodeInfo{
		Creator:       nodeAddr,
		PeerId:        "peer-1",
		ControllerKey: controllerPubKeyHex,
	})
	require.NoError(t, err)

	_, err = k.UpdateNodeInfo(ctx, &types.MsgUpdateNodeInfo{
		Creator: wrongAddr,
		NodeKey: nodePubKeyHex,
	})
	require.ErrorIs(t, err, types.ErrUnauthorizedNodeInfoUpdate)
}

// setupPeerWithNodeInfo registers an account and creates a NodeInfo entry for it.
// Returns the account bech32 address and its hex-encoded public key (the node_key used in rings).
func setupPeerWithNodeInfo(t *testing.T, k keeper.Keeper, ak authkeeper.AccountKeeper, ctx sdk.Context, networkID string) (addr string, nodeKey string) {
	t.Helper()
	addr, nodeKey = testAccountWithPubKey(t, ctx, ak)
	_, err := k.CreateNodeInfo(ctx, &types.MsgCreateNodeInfo{
		Creator:       addr,
		PeerId:        networkID,
		ControllerKey: nodeKey,
	})
	require.NoError(t, err)
	return addr, nodeKey
}

// setupNamespaceWithMember creates the orbis module policy, registers namespaceID as an object,
// and grants actorDID the member relation so CreateRing passes the ACP check.
func setupNamespaceWithMember(t *testing.T, k keeper.Keeper, ctx sdk.Context, namespaceID, actorDID, signerAddr string) {
	t.Helper()

	policyId, err := k.EnsurePolicy(ctx)
	require.NoError(t, err)

	manager := capability.NewPolicyCapabilityManager(k.GetScopedKeeper())
	polCap, err := manager.Fetch(ctx, policyId)
	require.NoError(t, err)

	registerCmd := acptypes.NewRegisterObjectCmd(coretypes.NewObject(types.NamespaceResource, namespaceID))
	_, err = k.GetAcpKeeper().ModulePolicyCmdForActorDID(ctx, polCap, registerCmd, actorDID, signerAddr)
	require.NoError(t, err)

	memberRel := coretypes.NewActorRelationship(types.NamespaceResource, namespaceID, "member", actorDID)
	setRelCmd := acptypes.NewSetRelationshipCmd(memberRel)
	_, err = k.GetAcpKeeper().ModulePolicyCmdForActorDID(ctx, polCap, setRelCmd, actorDID, signerAddr)
	require.NoError(t, err)
}

// testAccountWithPubKey creates a secp256k1 keypair, registers the account in the auth keeper,
// and returns the bech32 address and hex-encoded public key bytes.
func testAccountWithPubKey(t *testing.T, ctx sdk.Context, ak authkeeper.AccountKeeper) (addr string, pubKeyHex string) {
	t.Helper()
	privKey := secp256k1.GenPrivKey()
	pubKey := privKey.PubKey()
	accAddr := sdk.AccAddress(pubKey.Address())
	account := ak.NewAccountWithAddress(ctx, accAddr)
	if err := account.SetPubKey(pubKey); err != nil {
		t.Fatal(err)
	}
	ak.SetAccount(ctx, account)
	return accAddr.String(), hex.EncodeToString(pubKey.Bytes())
}

// testPeerID returns a unique libp2p-style peer ID string for use in tests.
func testPeerID(n int) string {
	return fmt.Sprintf("12D3KooWTestPeer%d", n)
}
