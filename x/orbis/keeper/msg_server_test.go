package keeper

import (
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	"github.com/cosmos/gogoproto/proto"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/sourcenetwork/immutable"
	"github.com/stretchr/testify/require"

	appparams "github.com/sourcenetwork/vera/app/params"
	acptypes "github.com/sourcenetwork/vera/x/acp/types"
	"github.com/sourcenetwork/vera/x/orbis/types"
)

const testDID = "did:example:orbis-creator"
const testPolicyOwnerDID = "did:example:orbis-policy-owner"
const testOperatorDID = "did:example:orbis-operator"
const testOutsiderDID = "did:example:orbis-outsider"
const testPeerDID = "did:example:orbis-peer"
const testRelayDID = "did:key:z6MkpTHR8VNsBxYAAWHut2Geadd9jSwuBV8xRoAnwWsdvktH"
const testSecondRelayDID = "did:key:z6MkhaXgBZDvotDkL5257faiztiGiC2QtKLGpbnnEGta2doK"

const testOrbisRingPolicy = `
name: orbis ring policy
resources:
- name: ring_policy
  permissions:
  - name: create_ring
    expr: ring_creator
  relations:
  - name: ring_creator
    types:
    - actor
- name: ring
  permissions:
  - name: update_ring
    expr: operator
  relations:
  - name: operator
    types:
    - actor
`

const testNonRingPolicy = `
name: non-ring policy
resources:
- name: file
`

func TestMsgServer_CreateRingStoreDocumentAndKeyDerivation(t *testing.T) {
	k, authKeeper, ctx := setupOrbisKeeper(t)
	ctx = ctx.WithValue(appparams.ExtractedDIDContextKey, testDID)

	creatorAddr, _ := testAccountWithPubKey(t, ctx, authKeeper)

	peer1Addr, peer1Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer1")
	peer2Addr, peer2Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer2")
	peer3Addr, peer3Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer3")
	policyID := createOrbisRingPolicy(t, k, ctx, creatorAddr)

	pssInterval := types.MinPSSIntervalSeconds + 1
	createRingResp, err := k.CreateRing(ctx, &types.MsgCreateRing{
		Creator:      creatorAddr,
		PeerNodeKeys: []string{peer1Key, peer2Key, peer3Key},
		Threshold:    2,
		PolicyId:     policyID,
		PssInterval:  pssInterval,
	})
	require.NoError(t, err)

	ring := k.GetRing(ctx, createRingResp.RingId)
	require.NotNil(t, ring)
	require.Equal(t, testDID, ring.CreatorDid)
	require.Empty(t, ring.RingPk)
	require.Equal(t, canonicalStrings([]string{peer1Key, peer2Key, peer3Key}), ring.PeerNodeKeys)
	require.Equal(t, uint32(2), ring.Threshold)
	require.Equal(t, policyID, ring.PolicyId)
	require.Equal(t, pssInterval, ring.GetPssInterval())
	require.Equal(t, types.DefaultReportingConfig(), ring.Reporting)

	// all three nodes must confirm; ring not finalized until the last one
	_, err = k.FinalizeRing(ctx, &types.MsgFinalizeRing{Creator: peer1Addr, RingId: createRingResp.RingId, RingPk: "ring-pk"})
	require.NoError(t, err)
	require.Empty(t, k.GetRing(ctx, createRingResp.RingId).RingPk)

	_, err = k.FinalizeRing(ctx, &types.MsgFinalizeRing{Creator: peer2Addr, RingId: createRingResp.RingId, RingPk: "ring-pk"})
	require.NoError(t, err)
	require.Empty(t, k.GetRing(ctx, createRingResp.RingId).RingPk)

	_, err = k.FinalizeRing(ctx, &types.MsgFinalizeRing{Creator: peer3Addr, RingId: createRingResp.RingId, RingPk: "ring-pk"})
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
		PeerNodeKeys: []string{peer1Key, peer2Key, peer3Key},
		Threshold:    2,
		PolicyId:     policyID,
		PssInterval:  pssInterval,
	})
	require.ErrorIs(t, err, types.ErrRingAlreadyExists)

	tier := "gold"
	timestamp := uint64(42)
	storeDocumentMsg := &types.MsgStoreDocument{
		Creator:    creatorAddr,
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
		types.GenerateDocumentID(createRingResp.RingId, "ciphertext", "proof", "policy-doc", "secret", "decrypt", immutable.Some(tier), immutable.Some(timestamp)),
		storeDocumentResp.DocumentId,
	)

	document := k.GetDocument(ctx, storeDocumentResp.DocumentId)
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
		types.GenerateKeyDerivationID(createRingResp.RingId, "m/0/1", "policy-derivation", "derived-key", "derive"),
		storeKeyDerivationResp.KeyDerivationId,
	)

	keyDerivation := k.GetKeyDerivation(ctx, storeKeyDerivationResp.KeyDerivationId)
	require.NotNil(t, keyDerivation)
	require.Equal(t, testDID, keyDerivation.CreatorDid)
	require.Equal(t, "m/0/1", keyDerivation.Derivation)

	_, err = k.StoreKeyDerivation(ctx, storeKeyDerivationMsg)
	require.ErrorIs(t, err, types.ErrKeyDerivationAlreadyExists)
}

func TestMsgServer_CreateRingSnapshotsReportingConfig(t *testing.T) {
	k, authKeeper, ctx := setupOrbisKeeper(t)
	ctx = ctx.WithValue(appparams.ExtractedDIDContextKey, testDID)

	creatorAddr, _ := testAccountWithPubKey(t, ctx, authKeeper)
	_, peer1Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer1")
	policyID := createOrbisRingPolicy(t, k, ctx, creatorAddr)

	defaultConfig := types.DemeritConfig{
		NodeOfflineDemerits:           4,
		InvalidCryptoResponseDemerits: types.DefaultInvalidCryptoResponseDemerits,
		UnauthorizedRequestDemerits:   types.DefaultUnauthorizedRequestDemerits,
		ResetIntervalSeconds:          12,
	}
	defaultReporting := types.ReportingDefaults{
		DemeritConfig: defaultConfig,
		KickThreshold: 5,
	}
	require.NoError(t, k.SetParams(ctx, types.NewParams(defaultReporting)))

	defaultResp, err := k.CreateRing(ctx, &types.MsgCreateRing{
		Creator:      creatorAddr,
		PeerNodeKeys: []string{peer1Key},
		Threshold:    1,
		PssInterval:  types.MinPSSIntervalSeconds,
		PolicyId:     policyID,
	})
	require.NoError(t, err)
	require.Equal(t, types.ReportingConfigFromDefaults(defaultReporting), k.GetRing(ctx, defaultResp.RingId).Reporting)

	updatedReporting := types.ReportingDefaults{
		DemeritConfig: types.DemeritConfig{
			NodeOfflineDemerits:           8,
			InvalidCryptoResponseDemerits: types.DefaultInvalidCryptoResponseDemerits,
			UnauthorizedRequestDemerits:   types.DefaultUnauthorizedRequestDemerits,
			ResetIntervalSeconds:          24,
		},
		KickThreshold: 6,
	}
	require.NoError(t, k.SetParams(ctx, types.NewParams(updatedReporting)))
	require.Equal(t, types.ReportingConfigFromDefaults(defaultReporting), k.GetRing(ctx, defaultResp.RingId).Reporting)

	explicitConfig := types.DemeritConfig{
		NodeOfflineDemerits:           7,
		InvalidCryptoResponseDemerits: types.DefaultInvalidCryptoResponseDemerits,
		UnauthorizedRequestDemerits:   types.DefaultUnauthorizedRequestDemerits,
		ResetIntervalSeconds:          60,
	}
	explicitReporting := types.ReportingConfig{
		DemeritConfig: explicitConfig,
		KickThreshold: 2,
	}
	explicitResp, err := k.CreateRing(ctx, &types.MsgCreateRing{
		Creator:      creatorAddr,
		PeerNodeKeys: []string{peer1Key},
		Threshold:    1,
		PssInterval:  types.MinPSSIntervalSeconds,
		PolicyId:     policyID,
		XNonce:       &types.MsgCreateRing_Nonce{Nonce: "explicit-reporting-config"},
		Reporting:    &explicitReporting,
	})
	require.NoError(t, err)
	require.Equal(t, explicitReporting, k.GetRing(ctx, explicitResp.RingId).Reporting)
}

// TestMsgServer_CreateRingParamsValidation exercises the params validation/defaulting
// paths in params.go that TestMsgServer_CreateRingSnapshotsReportingConfig does not reach:
// (1) ring with no Reporting inherits the module-default Reporting via GetParams,
// (2) SetParams rejects zero-value fields and leaves existing params intact.
func TestMsgServer_CreateRingParamsValidation(t *testing.T) {
	k, authKeeper, ctx := setupOrbisKeeper(t)
	ctx = ctx.WithValue(appparams.ExtractedDIDContextKey, testDID)

	creatorAddr, _ := testAccountWithPubKey(t, ctx, authKeeper)
	_, peer1Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer1")
	policyID := createOrbisRingPolicy(t, k, ctx, creatorAddr)

	// setupOrbisKeeper seeds the store with DefaultParams; a ring with no explicit
	// Reporting must inherit exactly DefaultReportingConfig via GetParams.
	resp, err := k.CreateRing(ctx, &types.MsgCreateRing{
		Creator:      creatorAddr,
		PeerNodeKeys: []string{peer1Key},
		Threshold:    1,
		PssInterval:  types.MinPSSIntervalSeconds,
		PolicyId:     policyID,
	})
	require.NoError(t, err)
	require.Equal(t, types.DefaultReportingConfig(), k.GetRing(ctx, resp.RingId).Reporting)

	// params.go SetParams runs Validate(); zero NodeOfflineDemerits must be rejected.
	require.ErrorContains(t,
		k.SetParams(ctx, types.NewParams(types.ReportingDefaults{
			DemeritConfig: types.DemeritConfig{
				NodeOfflineDemerits:           0,
				InvalidCryptoResponseDemerits: types.DefaultInvalidCryptoResponseDemerits,
				UnauthorizedRequestDemerits:   types.DefaultUnauthorizedRequestDemerits,
				ResetIntervalSeconds:          types.DefaultDemeritResetIntervalSecs,
			},
			KickThreshold: types.DefaultReportingKickThreshold,
		})),
		"node_offline_demerits must be at least 1",
	)

	// params.go SetParams runs Validate(); zero ResetIntervalSeconds must be rejected.
	require.ErrorContains(t,
		k.SetParams(ctx, types.NewParams(types.ReportingDefaults{
			DemeritConfig: types.DemeritConfig{
				NodeOfflineDemerits:           types.DefaultNodeOfflineDemerits,
				InvalidCryptoResponseDemerits: types.DefaultInvalidCryptoResponseDemerits,
				UnauthorizedRequestDemerits:   types.DefaultUnauthorizedRequestDemerits,
				ResetIntervalSeconds:          0,
			},
			KickThreshold: types.DefaultReportingKickThreshold,
		})),
		"reset_interval_seconds must be at least 1",
	)

	// params.go SetParams runs Validate(); zero KickThreshold must be rejected.
	require.ErrorContains(t,
		k.SetParams(ctx, types.NewParams(types.ReportingDefaults{
			DemeritConfig: types.DefaultDemeritConfig(),
		})),
		"kick_threshold must be at least 1",
	)

	// After both failed SetParams calls params remain DefaultParams; a second ring
	// (disambiguated by nonce) must still inherit DefaultReportingConfig.
	resp2, err := k.CreateRing(ctx, &types.MsgCreateRing{
		Creator:      creatorAddr,
		PeerNodeKeys: []string{peer1Key},
		Threshold:    1,
		PssInterval:  types.MinPSSIntervalSeconds,
		PolicyId:     policyID,
		XNonce:       &types.MsgCreateRing_Nonce{Nonce: "after-invalid-params"},
	})
	require.NoError(t, err)
	require.Equal(t, types.DefaultReportingConfig(), k.GetRing(ctx, resp2.RingId).Reporting)
}

func TestMsgServer_CreateRing_NonceDisambiguatesIdenticalSettings(t *testing.T) {
	k, authKeeper, ctx := setupOrbisKeeper(t)
	ctx = ctx.WithValue(appparams.ExtractedDIDContextKey, testDID)

	creatorAddr, _ := testAccountWithPubKey(t, ctx, authKeeper)
	_, peer1Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer1")
	policyID := createOrbisRingPolicy(t, k, ctx, creatorAddr)

	base := &types.MsgCreateRing{
		Creator:      creatorAddr,
		PeerNodeKeys: []string{peer1Key},
		Threshold:    1,
		PssInterval:  types.MinPSSIntervalSeconds,
		PolicyId:     policyID,
	}

	// first ring — no nonce
	resp1, err := k.CreateRing(ctx, base)
	require.NoError(t, err)

	// identical settings without nonce clash
	_, err = k.CreateRing(ctx, base)
	require.ErrorIs(t, err, types.ErrRingAlreadyExists)

	// same settings but with a nonce produces a different ring_id
	resp2, err := k.CreateRing(ctx, &types.MsgCreateRing{
		Creator:      creatorAddr,
		PeerNodeKeys: []string{peer1Key},
		Threshold:    1,
		PssInterval:  types.MinPSSIntervalSeconds,
		PolicyId:     policyID,
		XNonce:       &types.MsgCreateRing_Nonce{Nonce: "attempt-2"},
	})
	require.NoError(t, err)
	require.NotEqual(t, resp1.RingId, resp2.RingId)

	// different nonce values produce different ring_ids
	resp3, err := k.CreateRing(ctx, &types.MsgCreateRing{
		Creator:      creatorAddr,
		PeerNodeKeys: []string{peer1Key},
		Threshold:    1,
		PssInterval:  types.MinPSSIntervalSeconds,
		PolicyId:     policyID,
		XNonce:       &types.MsgCreateRing_Nonce{Nonce: "attempt-3"},
	})
	require.NoError(t, err)
	require.NotEqual(t, resp2.RingId, resp3.RingId)
}

func TestMsgServer_CreateRing_PeerKeyOrderDoesNotAffectRingID(t *testing.T) {
	k, authKeeper, ctx := setupOrbisKeeper(t)
	ctx = ctx.WithValue(appparams.ExtractedDIDContextKey, testDID)

	creatorAddr, _ := testAccountWithPubKey(t, ctx, authKeeper)
	_, peer1Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer1")
	_, peer2Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer2")
	policyID := createOrbisRingPolicy(t, k, ctx, creatorAddr)

	canonicalCommittee := canonicalStrings([]string{peer1Key, peer2Key})
	submittedCommittee := []string{canonicalCommittee[1], canonicalCommittee[0]}
	resp, err := k.CreateRing(ctx, &types.MsgCreateRing{
		Creator:      creatorAddr,
		PeerNodeKeys: submittedCommittee,
		Threshold:    1,
		PssInterval:  types.MinPSSIntervalSeconds,
		PolicyId:     policyID,
	})
	require.NoError(t, err)

	// GenerateRingID with keys in the opposite order must produce the same ID
	require.Equal(t, types.GenerateRingID(canonicalCommittee, 1, types.MinPSSIntervalSeconds, policyID, immutable.None[string](), 0, false, nil), resp.RingId)
	require.Equal(t, []string{canonicalCommittee[1], canonicalCommittee[0]}, submittedCommittee)
	require.Equal(t, canonicalCommittee, k.GetRing(ctx, resp.RingId).PeerNodeKeys)
}

func TestMsgServer_CreateRingRequiresPolicyID(t *testing.T) {
	k, authKeeper, ctx := setupOrbisKeeper(t)
	ctx = ctx.WithValue(appparams.ExtractedDIDContextKey, testDID)

	creatorAddr, _ := testAccountWithPubKey(t, ctx, authKeeper)
	_, peer1Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer1")

	_, err := k.CreateRing(ctx, &types.MsgCreateRing{
		Creator:      creatorAddr,
		PeerNodeKeys: []string{peer1Key},
		Threshold:    1,
		PssInterval:  types.MinPSSIntervalSeconds,
	})
	require.ErrorIs(t, err, types.ErrInvalidRing)
	require.ErrorContains(t, err, "missing policy_id")
}

func TestMsgServer_UpdateRingTrustedAuthRelaysByAcp(t *testing.T) {
	k, authKeeper, ctx := setupOrbisKeeper(t)
	creatorCtx := ctxWithDID(ctx, testDID)
	outsiderCtx := ctxWithDID(ctx, testOutsiderDID)

	creatorAddr, _ := testAccountWithPubKey(t, creatorCtx, authKeeper)
	_, peerKey := setupPeerWithNodeInfo(t, k, authKeeper, creatorCtx, "12D3KooWPeer1")
	policyID := createOrbisRingPolicy(t, k, creatorCtx, creatorAddr)

	createResp, err := k.CreateRing(creatorCtx, &types.MsgCreateRing{
		Creator:                creatorAddr,
		PeerNodeKeys:           []string{peerKey},
		Threshold:              1,
		PssInterval:            types.MinPSSIntervalSeconds,
		PolicyId:               policyID,
		AllowTrustedAuthRelays: true,
		TrustedAuthRelayDids:   []string{testSecondRelayDID},
	})
	require.NoError(t, err)
	require.Equal(
		t,
		types.GenerateRingID([]string{peerKey}, 1, types.MinPSSIntervalSeconds, policyID, immutable.None[string](), 0, true, []string{testSecondRelayDID}),
		createResp.RingId,
	)

	outsiderAddr, _ := testAccountWithPubKey(t, outsiderCtx, authKeeper)
	_, err = k.AddRingTrustedAuthRelayByAcp(outsiderCtx, &types.MsgAddRingTrustedAuthRelayByAcp{
		Creator:  outsiderAddr,
		RingId:   createResp.RingId,
		RelayDid: testRelayDID,
	})
	require.ErrorIs(t, err, types.ErrUnauthorizedRingUpdate)

	_, err = k.AddRingTrustedAuthRelayByAcp(creatorCtx, &types.MsgAddRingTrustedAuthRelayByAcp{
		Creator:  creatorAddr,
		RingId:   createResp.RingId,
		RelayDid: testRelayDID,
	})
	require.NoError(t, err)
	require.Equal(
		t,
		[]string{testSecondRelayDID, testRelayDID},
		k.GetRing(creatorCtx, createResp.RingId).TrustedAuthRelayDids,
	)

	_, err = k.AddRingTrustedAuthRelayByAcp(creatorCtx, &types.MsgAddRingTrustedAuthRelayByAcp{
		Creator:  creatorAddr,
		RingId:   createResp.RingId,
		RelayDid: testRelayDID,
	})
	require.ErrorIs(t, err, types.ErrRingAuthRelayExists)

	_, err = k.RemoveRingTrustedAuthRelayByAcp(outsiderCtx, &types.MsgRemoveRingTrustedAuthRelayByAcp{
		Creator:  outsiderAddr,
		RingId:   createResp.RingId,
		RelayDid: testSecondRelayDID,
	})
	require.ErrorIs(t, err, types.ErrUnauthorizedRingUpdate)

	_, err = k.RemoveRingTrustedAuthRelayByAcp(creatorCtx, &types.MsgRemoveRingTrustedAuthRelayByAcp{
		Creator:  creatorAddr,
		RingId:   createResp.RingId,
		RelayDid: testSecondRelayDID,
	})
	require.NoError(t, err)
	require.Equal(
		t,
		[]string{testRelayDID},
		k.GetRing(creatorCtx, createResp.RingId).TrustedAuthRelayDids,
	)

	_, err = k.RemoveRingTrustedAuthRelayByAcp(creatorCtx, &types.MsgRemoveRingTrustedAuthRelayByAcp{
		Creator:  creatorAddr,
		RingId:   createResp.RingId,
		RelayDid: testSecondRelayDID,
	})
	require.ErrorIs(t, err, types.ErrRingTrustedAuthRelayNotFound)
}

func TestMsgServer_RingTrustedAuthRelayUpdatesRequireOptIn(t *testing.T) {
	k, authKeeper, ctx := setupOrbisKeeper(t)
	ctx = ctxWithDID(ctx, testDID)

	creatorAddr, _ := testAccountWithPubKey(t, ctx, authKeeper)
	_, peerKey := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer1")
	policyID := createOrbisRingPolicy(t, k, ctx, creatorAddr)
	createResp, err := k.CreateRing(ctx, &types.MsgCreateRing{
		Creator:      creatorAddr,
		PeerNodeKeys: []string{peerKey},
		Threshold:    1,
		PssInterval:  types.MinPSSIntervalSeconds,
		PolicyId:     policyID,
	})
	require.NoError(t, err)

	_, err = k.AddRingTrustedAuthRelayByAcp(ctx, &types.MsgAddRingTrustedAuthRelayByAcp{
		Creator:  creatorAddr,
		RingId:   createResp.RingId,
		RelayDid: testRelayDID,
	})
	require.ErrorIs(t, err, types.ErrRingAuthRelaysDisabled)

	_, err = k.RemoveRingTrustedAuthRelayByAcp(ctx, &types.MsgRemoveRingTrustedAuthRelayByAcp{
		Creator:  creatorAddr,
		RingId:   createResp.RingId,
		RelayDid: testRelayDID,
	})
	require.ErrorIs(t, err, types.ErrRingAuthRelaysDisabled)
}

func TestMsgServer_CreateRingRequiresExistingRingPolicy(t *testing.T) {
	k, authKeeper, ctx := setupOrbisKeeper(t)
	ctx = ctx.WithValue(appparams.ExtractedDIDContextKey, testDID)

	creatorAddr, _ := testAccountWithPubKey(t, ctx, authKeeper)
	_, peer1Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer1")

	ringID := types.GenerateRingID([]string{peer1Key}, 1, types.MinPSSIntervalSeconds, "missing-policy", immutable.None[string](), 0, false, nil)
	_, err := k.CreateRing(ctx, &types.MsgCreateRing{
		Creator:      creatorAddr,
		PeerNodeKeys: []string{peer1Key},
		Threshold:    1,
		PssInterval:  types.MinPSSIntervalSeconds,
		PolicyId:     "missing-policy",
	})
	require.Error(t, err)
	require.Nil(t, k.GetRing(ctx, ringID))
}

func TestMsgServer_CreateRingRequiresRegisteredRingPolicyControlObject(t *testing.T) {
	k, authKeeper, ctx := setupOrbisKeeper(t)
	ctx = ctx.WithValue(appparams.ExtractedDIDContextKey, testDID)

	creatorAddr, _ := testAccountWithPubKey(t, ctx, authKeeper)
	_, peer1Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer1")
	policyID := createACPPolicy(t, k, ctx, creatorAddr, testOrbisRingPolicy)

	ringID := types.GenerateRingID([]string{peer1Key}, 1, types.MinPSSIntervalSeconds, policyID, immutable.None[string](), 0, false, nil)
	_, err := k.CreateRing(ctx, &types.MsgCreateRing{
		Creator:      creatorAddr,
		PeerNodeKeys: []string{peer1Key},
		Threshold:    1,
		PssInterval:  types.MinPSSIntervalSeconds,
		PolicyId:     policyID,
	})
	require.ErrorIs(t, err, types.ErrUnauthorizedRingCreate)
	require.Nil(t, k.GetRing(ctx, ringID))
}

func TestMsgServer_CreateRingRequiresPolicyWithRingResource(t *testing.T) {
	k, authKeeper, ctx := setupOrbisKeeper(t)
	ctx = ctx.WithValue(appparams.ExtractedDIDContextKey, testDID)

	creatorAddr, _ := testAccountWithPubKey(t, ctx, authKeeper)
	_, peer1Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer1")
	policyID := createACPPolicy(t, k, ctx, creatorAddr, testNonRingPolicy)

	ringID := types.GenerateRingID([]string{peer1Key}, 1, types.MinPSSIntervalSeconds, policyID, immutable.None[string](), 0, false, nil)
	_, err := k.CreateRing(ctx, &types.MsgCreateRing{
		Creator:      creatorAddr,
		PeerNodeKeys: []string{peer1Key},
		Threshold:    1,
		PssInterval:  types.MinPSSIntervalSeconds,
		PolicyId:     policyID,
	})
	require.Error(t, err)
	require.Nil(t, k.GetRing(ctx, ringID))
}

func TestMsgServer_CreateRingRejectsActorWithoutCreatePermission(t *testing.T) {
	k, authKeeper, ctx := setupOrbisKeeper(t)
	policyOwnerCtx := ctxWithDID(ctx, testPolicyOwnerDID)
	creatorCtx := ctxWithDID(ctx, testDID)

	policyOwnerAddr, _ := testAccountWithPubKey(t, policyOwnerCtx, authKeeper)
	creatorAddr, _ := testAccountWithPubKey(t, creatorCtx, authKeeper)
	_, peer1Key := setupPeerWithNodeInfo(t, k, authKeeper, policyOwnerCtx, "12D3KooWPeer1")
	policyID := createOrbisRingPolicy(t, k, policyOwnerCtx, policyOwnerAddr)

	ringID := types.GenerateRingID([]string{peer1Key}, 1, types.MinPSSIntervalSeconds, policyID, immutable.None[string](), 0, false, nil)
	_, err := k.CreateRing(creatorCtx, &types.MsgCreateRing{
		Creator:      creatorAddr,
		PeerNodeKeys: []string{peer1Key},
		Threshold:    1,
		PssInterval:  types.MinPSSIntervalSeconds,
		PolicyId:     policyID,
	})
	require.ErrorIs(t, err, types.ErrUnauthorizedRingCreate)
	require.Nil(t, k.GetRing(creatorCtx, ringID))
}

func TestMsgServer_CreateRingAllowsRingCreatorRelation(t *testing.T) {
	k, authKeeper, ctx := setupOrbisKeeper(t)
	policyOwnerCtx := ctxWithDID(ctx, testPolicyOwnerDID)
	creatorCtx := ctxWithDID(ctx, testDID)

	policyOwnerAddr, _ := testAccountWithPubKey(t, policyOwnerCtx, authKeeper)
	creatorAddr, _ := testAccountWithPubKey(t, creatorCtx, authKeeper)
	_, peer1Key := setupPeerWithNodeInfo(t, k, authKeeper, policyOwnerCtx, "12D3KooWPeer1")
	policyID := createOrbisRingPolicy(t, k, policyOwnerCtx, policyOwnerAddr)
	grantRingCreator(t, k, policyOwnerCtx, policyOwnerAddr, policyID, testDID)

	createRingResp, err := k.CreateRing(creatorCtx, &types.MsgCreateRing{
		Creator:      creatorAddr,
		PeerNodeKeys: []string{peer1Key},
		Threshold:    1,
		PssInterval:  types.MinPSSIntervalSeconds,
		PolicyId:     policyID,
	})
	require.NoError(t, err)

	ring := k.GetRing(creatorCtx, createRingResp.RingId)
	require.NotNil(t, ring)
	require.Equal(t, testDID, ring.CreatorDid)

	ownerResp, err := k.GetAcpKeeper().ObjectOwner(creatorCtx, &acptypes.QueryObjectOwnerRequest{
		PolicyId: policyID,
		Object:   coretypes.NewObject(types.ACPResourceRing, createRingResp.RingId),
	})
	require.NoError(t, err)
	require.True(t, ownerResp.IsRegistered)
	require.Equal(t, testDID, ownerResp.Record.Metadata.OwnerDid)
}

func TestMsgServer_CreateRingRegistersACPObject(t *testing.T) {
	k, authKeeper, ctx := setupOrbisKeeper(t)
	ctx = ctx.WithValue(appparams.ExtractedDIDContextKey, testDID)

	creatorAddr, _ := testAccountWithPubKey(t, ctx, authKeeper)
	_, peer1Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer1")
	policyID := createOrbisRingPolicy(t, k, ctx, creatorAddr)

	createRingResp, err := k.CreateRing(ctx, &types.MsgCreateRing{
		Creator:      creatorAddr,
		PeerNodeKeys: []string{peer1Key},
		Threshold:    1,
		PssInterval:  types.MinPSSIntervalSeconds,
		PolicyId:     policyID,
	})
	require.NoError(t, err)

	ownerResp, err := k.GetAcpKeeper().ObjectOwner(ctx, &acptypes.QueryObjectOwnerRequest{
		PolicyId: policyID,
		Object:   coretypes.NewObject(types.ACPResourceRing, createRingResp.RingId),
	})
	require.NoError(t, err)
	require.True(t, ownerResp.IsRegistered)
	require.NotNil(t, ownerResp.Record)
}

func TestMsgServer_FinalizeRing_UnauthorizedNonPeer(t *testing.T) {
	k, authKeeper, ctx := setupOrbisKeeper(t)
	ctx = ctx.WithValue(appparams.ExtractedDIDContextKey, testDID)

	creatorAddr, _ := testAccountWithPubKey(t, ctx, authKeeper)

	_, peer1Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer1")
	_, peer2Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer2")
	policyID := createOrbisRingPolicy(t, k, ctx, creatorAddr)

	createRingResp, err := k.CreateRing(ctx, &types.MsgCreateRing{
		Creator:      creatorAddr,
		PeerNodeKeys: []string{peer1Key, peer2Key},
		Threshold:    1,
		PssInterval:  types.MinPSSIntervalSeconds,
		PolicyId:     policyID,
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

func TestMsgServer_FinalizeRing_RequiresAllNodes(t *testing.T) {
	k, authKeeper, ctx := setupOrbisKeeper(t)
	ctx = ctx.WithValue(appparams.ExtractedDIDContextKey, testDID)

	creatorAddr, _ := testAccountWithPubKey(t, ctx, authKeeper)

	peer1Addr, peer1Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer1")
	peer2Addr, peer2Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer2")
	peer3Addr, peer3Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer3")
	policyID := createOrbisRingPolicy(t, k, ctx, creatorAddr)

	createRingResp, err := k.CreateRing(ctx, &types.MsgCreateRing{
		Creator:      creatorAddr,
		PeerNodeKeys: []string{peer1Key, peer2Key, peer3Key},
		Threshold:    2,
		PssInterval:  types.MinPSSIntervalSeconds,
		PolicyId:     policyID,
	})
	require.NoError(t, err)
	ringID := createRingResp.RingId

	// first confirmation — not finalized
	finalizeResp, err := k.FinalizeRing(ctx, &types.MsgFinalizeRing{Creator: peer1Addr, RingId: ringID, RingPk: "agreed-pk"})
	require.NoError(t, err)
	require.Equal(t, types.FinalizeRingOutcome_CONFIRMATION_RECORDED, finalizeResp.Outcome)
	require.Empty(t, k.GetRing(ctx, ringID).RingPk)
	require.Len(t, k.GetRing(ctx, ringID).Confirmations, 1)

	// second confirmation — threshold=2 is met but not all nodes, so still not finalized
	finalizeResp, err = k.FinalizeRing(ctx, &types.MsgFinalizeRing{Creator: peer2Addr, RingId: ringID, RingPk: "agreed-pk"})
	require.NoError(t, err)
	require.Equal(t, types.FinalizeRingOutcome_CONFIRMATION_RECORDED, finalizeResp.Outcome)
	require.Empty(t, k.GetRing(ctx, ringID).RingPk)
	require.Len(t, k.GetRing(ctx, ringID).Confirmations, 2)

	// third confirmation — all nodes confirmed → finalized, confirmations cleared
	finalizeResp, err = k.FinalizeRing(ctx, &types.MsgFinalizeRing{Creator: peer3Addr, RingId: ringID, RingPk: "agreed-pk"})
	require.NoError(t, err)
	require.Equal(t, types.FinalizeRingOutcome_RING_FINALIZED, finalizeResp.Outcome)
	ring := k.GetRing(ctx, ringID)
	require.Equal(t, "agreed-pk", ring.RingPk)
	require.Empty(t, ring.Confirmations)

	// any further confirmation must fail
	_, err = k.FinalizeRing(ctx, &types.MsgFinalizeRing{Creator: peer1Addr, RingId: ringID, RingPk: "agreed-pk"})
	require.ErrorIs(t, err, types.ErrRingAlreadyFinalized)
}

func TestMsgServer_FinalizeRing_PkConflictDeletesRing(t *testing.T) {
	k, authKeeper, ctx := setupOrbisKeeper(t)
	ctx = ctx.WithValue(appparams.ExtractedDIDContextKey, testDID)

	creatorAddr, _ := testAccountWithPubKey(t, ctx, authKeeper)

	peer1Addr, peer1Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer1")
	peer2Addr, peer2Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer2")
	policyID := createOrbisRingPolicy(t, k, ctx, creatorAddr)

	createRingResp, err := k.CreateRing(ctx, &types.MsgCreateRing{
		Creator:      creatorAddr,
		PeerNodeKeys: []string{peer1Key, peer2Key},
		Threshold:    2,
		PssInterval:  types.MinPSSIntervalSeconds,
		PolicyId:     policyID,
	})
	require.NoError(t, err)
	ringID := createRingResp.RingId

	finalizeResp, err := k.FinalizeRing(ctx, &types.MsgFinalizeRing{Creator: peer1Addr, RingId: ringID, RingPk: "pk-version-A"})
	require.NoError(t, err)
	require.Equal(t, types.FinalizeRingOutcome_CONFIRMATION_RECORDED, finalizeResp.Outcome)

	// peer2 disagrees on the ring_pk -> BFT violation, ring is deleted
	conflictCtx := ctx.WithEventManager(sdk.NewEventManager())
	finalizeResp, err = k.FinalizeRing(conflictCtx, &types.MsgFinalizeRing{Creator: peer2Addr, RingId: ringID, RingPk: "pk-version-B"})
	require.NoError(t, err)
	require.Equal(t, types.FinalizeRingOutcome_CONFLICT_DELETED, finalizeResp.Outcome)
	require.Nil(t, k.GetRing(ctx, ringID))
	require.Equal(t, []proto.Message{
		&types.EventRingDeleted{
			RingId: ringID,
			Reason: "ring_pk_conflict",
		},
	}, parseTypedEvents(t, conflictCtx))
}

func TestMsgServer_FinalizeRing_DuplicateConfirmationRejected(t *testing.T) {
	k, authKeeper, ctx := setupOrbisKeeper(t)
	ctx = ctx.WithValue(appparams.ExtractedDIDContextKey, testDID)

	creatorAddr, _ := testAccountWithPubKey(t, ctx, authKeeper)

	peer1Addr, peer1Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer1")
	_, peer2Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer2")
	policyID := createOrbisRingPolicy(t, k, ctx, creatorAddr)

	createRingResp, err := k.CreateRing(ctx, &types.MsgCreateRing{
		Creator:      creatorAddr,
		PeerNodeKeys: []string{peer1Key, peer2Key},
		Threshold:    2,
		PssInterval:  types.MinPSSIntervalSeconds,
		PolicyId:     policyID,
	})
	require.NoError(t, err)
	ringID := createRingResp.RingId

	finalizeResp, err := k.FinalizeRing(ctx, &types.MsgFinalizeRing{Creator: peer1Addr, RingId: ringID, RingPk: "ring-pk"})
	require.NoError(t, err)
	require.Equal(t, types.FinalizeRingOutcome_CONFIRMATION_RECORDED, finalizeResp.Outcome)

	_, err = k.FinalizeRing(ctx, &types.MsgFinalizeRing{Creator: peer1Addr, RingId: ringID, RingPk: "different-ring-pk"})
	require.ErrorIs(t, err, types.ErrDuplicateConfirmation)
	ring := k.GetRing(ctx, ringID)
	require.NotNil(t, ring)
	require.Len(t, ring.Confirmations, 1)
	require.Equal(t, "ring-pk", ring.Confirmations[0].RingPk)
}

func TestMsgServer_RingNotFinalizedGuard(t *testing.T) {
	k, authKeeper, ctx := setupOrbisKeeper(t)
	ctx = ctx.WithValue(appparams.ExtractedDIDContextKey, testDID)

	creatorAddr, _ := testAccountWithPubKey(t, ctx, authKeeper)

	_, peer1Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer1")
	policyID := createOrbisRingPolicy(t, k, ctx, creatorAddr)

	createRingResp, err := k.CreateRing(ctx, &types.MsgCreateRing{
		Creator:      creatorAddr,
		PeerNodeKeys: []string{peer1Key},
		Threshold:    1,
		PssInterval:  types.MinPSSIntervalSeconds,
		PolicyId:     policyID,
	})
	require.NoError(t, err)
	ringID := createRingResp.RingId

	_, err = k.StoreDocument(ctx, &types.MsgStoreDocument{
		Creator:    creatorAddr,
		RingId:     ringID,
		Document:   "ciphertext",
		Proof:      "proof",
		PolicyId:   "p",
		Resource:   "r",
		Permission: "read",
	})
	require.ErrorIs(t, err, types.ErrRingNotFinalized)

	_, err = k.StoreKeyDerivation(ctx, &types.MsgStoreKeyDerivation{
		Creator:    creatorAddr,
		RingId:     ringID,
		Derivation: "m/0/1",
		PolicyId:   "p",
		Resource:   "r",
		Permission: "derive",
	})
	require.ErrorIs(t, err, types.ErrRingNotFinalized)
}

func TestMsgServer_AbsentOptionalFieldsAreTreatedAsNone(t *testing.T) {
	k, authKeeper, ctx := setupOrbisKeeper(t)
	ctx = ctx.WithValue(appparams.ExtractedDIDContextKey, testDID)

	creatorAddr, _ := testAccountWithPubKey(t, ctx, authKeeper)

	peer1Addr, peer1Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer1")
	peer2Addr, peer2Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer2")
	policyID := createOrbisRingPolicy(t, k, ctx, creatorAddr)

	createRingResp, err := k.CreateRing(ctx, &types.MsgCreateRing{
		Creator:      creatorAddr,
		PeerNodeKeys: []string{peer1Key, peer2Key},
		Threshold:    1,
		PssInterval:  types.MinPSSIntervalSeconds,
		PolicyId:     policyID,
	})
	require.NoError(t, err)
	require.Equal(
		t,
		types.GenerateRingID([]string{peer1Key, peer2Key}, 1, types.MinPSSIntervalSeconds, policyID, immutable.None[string](), 0, false, nil),
		createRingResp.RingId,
	)

	_, err = k.FinalizeRing(ctx, &types.MsgFinalizeRing{Creator: peer1Addr, RingId: createRingResp.RingId, RingPk: "ring-pk"})
	require.NoError(t, err)
	_, err = k.FinalizeRing(ctx, &types.MsgFinalizeRing{Creator: peer2Addr, RingId: createRingResp.RingId, RingPk: "ring-pk"})
	require.NoError(t, err)

	storeDocumentResp, err := k.StoreDocument(ctx, &types.MsgStoreDocument{
		Creator:    creatorAddr,
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
		types.GenerateDocumentID(createRingResp.RingId, "ciphertext", "proof", "policy-doc", "secret", "decrypt", immutable.None[string](), immutable.None[uint64]()),
		storeDocumentResp.DocumentId,
	)
}

func TestMsgServer_CreateRingRejectsPSSIntervalBelowMinimum(t *testing.T) {
	k, authKeeper, ctx := setupOrbisKeeper(t)
	ctx = ctx.WithValue(appparams.ExtractedDIDContextKey, testDID)

	creatorAddr, _ := testAccountWithPubKey(t, ctx, authKeeper)

	_, peer1Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer1")
	_, peer2Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer2")
	policyID := createOrbisRingPolicy(t, k, ctx, creatorAddr)

	for _, pssInterval := range []uint64{0, types.MinPSSIntervalSeconds - 1} {
		_, err := k.CreateRing(ctx, &types.MsgCreateRing{
			Creator:      creatorAddr,
			PeerNodeKeys: []string{peer1Key, peer2Key},
			Threshold:    1,
			PssInterval:  pssInterval,
			PolicyId:     policyID,
		})
		require.ErrorContains(t, err, "pss_interval must be at least 86400 seconds")
	}
}

func TestMsgServer_SetRingPssIntervalByAcpAllowsRingOwner(t *testing.T) {
	k, authKeeper, ctx := setupOrbisKeeper(t)
	ctx = ctxWithDID(ctx, testDID)

	creatorAddr, _ := testAccountWithPubKey(t, ctx, authKeeper)

	peer1Addr, peer1Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer1")
	peer2Addr, peer2Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer2")
	policyID := createOrbisRingPolicy(t, k, ctx, creatorAddr)

	createRingResp, err := k.CreateRing(ctx, &types.MsgCreateRing{
		Creator:      creatorAddr,
		PeerNodeKeys: []string{peer1Key, peer2Key},
		Threshold:    1,
		PssInterval:  types.MinPSSIntervalSeconds,
		PolicyId:     policyID,
	})
	require.NoError(t, err)

	_, err = k.FinalizeRing(ctx, &types.MsgFinalizeRing{Creator: peer1Addr, RingId: createRingResp.RingId, RingPk: "ring-pk"})
	require.NoError(t, err)
	_, err = k.FinalizeRing(ctx, &types.MsgFinalizeRing{Creator: peer2Addr, RingId: createRingResp.RingId, RingPk: "ring-pk"})
	require.NoError(t, err)

	pssInterval := types.MinPSSIntervalSeconds + 1
	_, err = k.SetRingPssIntervalByAcp(ctx, &types.MsgSetRingPssIntervalByAcp{
		Creator:     creatorAddr,
		RingId:      createRingResp.RingId,
		PssInterval: pssInterval,
	})
	require.NoError(t, err)

	ring := k.GetRing(ctx, createRingResp.RingId)
	require.NotNil(t, ring)
	require.Equal(t, policyID, ring.PolicyId)
	require.Equal(t, pssInterval, ring.GetPssInterval())

	_, err = k.SetRingPssIntervalByAcp(ctx, &types.MsgSetRingPssIntervalByAcp{
		Creator:     creatorAddr,
		RingId:      createRingResp.RingId,
		PssInterval: pssInterval,
	})
	require.ErrorIs(t, err, types.ErrRingPssIntervalUnchanged)

	_, err = k.SetRingPssIntervalByAcp(ctx, &types.MsgSetRingPssIntervalByAcp{
		Creator:     creatorAddr,
		RingId:      createRingResp.RingId,
		PssInterval: types.MinPSSIntervalSeconds + 2,
	})
	require.NoError(t, err)
	require.Equal(t, types.MinPSSIntervalSeconds+2, k.GetRing(ctx, createRingResp.RingId).GetPssInterval())

	_, err = k.SetRingPssIntervalByAcp(ctx, &types.MsgSetRingPssIntervalByAcp{
		Creator: creatorAddr,
		RingId:  createRingResp.RingId,
	})
	require.ErrorContains(t, err, "pss_interval must be at least 86400 seconds")
}

func TestMsgServer_SetRingReportingByAcpAllowsRingOwner(t *testing.T) {
	k, authKeeper, ctx := setupOrbisKeeper(t)
	ctx = ctxWithDID(ctx, testDID)

	creatorAddr, _ := testAccountWithPubKey(t, ctx, authKeeper)

	peer1Addr, peer1Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer1")
	peer2Addr, peer2Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer2")
	backup1Addr, backup1Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWBackup1")
	backup2Addr, backup2Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWBackup2")
	policyID := createOrbisRingPolicy(t, k, ctx, creatorAddr)

	createRingResp, err := k.CreateRing(ctx, &types.MsgCreateRing{
		Creator:      creatorAddr,
		PeerNodeKeys: []string{peer1Key, peer2Key},
		Threshold:    1,
		PssInterval:  types.MinPSSIntervalSeconds,
		PolicyId:     policyID,
	})
	require.NoError(t, err)

	_, err = k.FinalizeRing(ctx, &types.MsgFinalizeRing{Creator: peer1Addr, RingId: createRingResp.RingId, RingPk: "ring-pk"})
	require.NoError(t, err)
	_, err = k.FinalizeRing(ctx, &types.MsgFinalizeRing{Creator: peer2Addr, RingId: createRingResp.RingId, RingPk: "ring-pk"})
	require.NoError(t, err)

	updatePeerNodeWhitelists(t, k, ctx, backup1Addr, backup1Key, []string{policyID}, nil)
	updatePeerNodeWhitelists(t, k, ctx, backup2Addr, backup2Key, []string{policyID}, nil)

	reporting := types.ReportingConfig{
		DemeritConfig: types.DemeritConfig{
			NodeOfflineDemerits:           2,
			InvalidCryptoResponseDemerits: types.DefaultInvalidCryptoResponseDemerits,
			UnauthorizedRequestDemerits:   types.DefaultUnauthorizedRequestDemerits,
			ResetIntervalSeconds:          42,
		},
		BackupNodeKeys: []string{backup2Key, backup1Key},
		KickThreshold:  4,
	}
	updateCtx := ctx.WithEventManager(sdk.NewEventManager())
	_, err = k.SetRingReportingByAcp(updateCtx, &types.MsgSetRingReportingByAcp{
		Creator:   creatorAddr,
		RingId:    createRingResp.RingId,
		Reporting: reporting,
	})
	require.NoError(t, err)

	ring := k.GetRing(ctx, createRingResp.RingId)
	require.NotNil(t, ring)
	require.Equal(t, reporting, ring.Reporting)
	require.Equal(t, []proto.Message{
		&types.EventRingUpdated{
			RingId:     createRingResp.RingId,
			UpdaterDid: testDID,
		},
	}, parseTypedEvents(t, updateCtx))

	invalidReporting := reporting
	invalidReporting.KickThreshold = 0
	_, err = k.SetRingReportingByAcp(updateCtx, &types.MsgSetRingReportingByAcp{
		Creator:   creatorAddr,
		RingId:    createRingResp.RingId,
		Reporting: invalidReporting,
	})
	require.ErrorContains(t, err, "kick_threshold must be at least 1")
	require.Equal(t, reporting, k.GetRing(ctx, createRingResp.RingId).Reporting)
}

func TestMsgServer_CreateRingValidatesReportingBackupNodeInfo(t *testing.T) {
	k, authKeeper, ctx := setupOrbisKeeper(t)
	ctx = ctxWithDID(ctx, testDID)

	creatorAddr, _ := testAccountWithPubKey(t, ctx, authKeeper)
	_, peer1Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer1")
	_, backupKey := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWBackup")
	_, missingBackupKey := testAccountWithPubKey(t, ctx, authKeeper)
	policyID := createOrbisRingPolicy(t, k, ctx, creatorAddr)

	missingNodeInfoReporting := types.DefaultReportingConfig()
	missingNodeInfoReporting.BackupNodeKeys = []string{missingBackupKey}
	_, err := k.CreateRing(ctx, &types.MsgCreateRing{
		Creator:      creatorAddr,
		PeerNodeKeys: []string{peer1Key},
		Threshold:    1,
		PssInterval:  types.MinPSSIntervalSeconds,
		PolicyId:     policyID,
		Reporting:    &missingNodeInfoReporting,
	})
	require.ErrorIs(t, err, types.ErrInvalidRing)
	require.ErrorContains(t, err, "backup_node_key")
	require.ErrorContains(t, err, "has no registered node info")

	reporting := types.DefaultReportingConfig()
	reporting.BackupNodeKeys = []string{backupKey}
	resp, err := k.CreateRing(ctx, &types.MsgCreateRing{
		Creator:      creatorAddr,
		PeerNodeKeys: []string{peer1Key},
		Threshold:    1,
		PssInterval:  types.MinPSSIntervalSeconds,
		PolicyId:     policyID,
		Reporting:    &reporting,
	})
	require.NoError(t, err)
	require.Equal(t, reporting, k.GetRing(ctx, resp.RingId).Reporting)
}

func TestMsgServer_SetRingReportingByAcpValidatesBackupNodeInfoAndWhitelist(t *testing.T) {
	k, authKeeper, ctx := setupOrbisKeeper(t)
	ctx = ctxWithDID(ctx, testDID)

	creatorAddr, _ := testAccountWithPubKey(t, ctx, authKeeper)
	peer1Addr, peer1Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer1")
	peer2Addr, peer2Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer2")
	_, backupNoWhitelistKey := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWBackupNoWhitelist")
	backupPolicyAddr, backupPolicyKey := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWBackupPolicy")
	backupRingAddr, backupRingKey := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWBackupRing")
	_, missingBackupKey := testAccountWithPubKey(t, ctx, authKeeper)
	policyID := createOrbisRingPolicy(t, k, ctx, creatorAddr)

	createRingResp, err := k.CreateRing(ctx, &types.MsgCreateRing{
		Creator:      creatorAddr,
		PeerNodeKeys: []string{peer1Key, peer2Key},
		Threshold:    1,
		PssInterval:  types.MinPSSIntervalSeconds,
		PolicyId:     policyID,
	})
	require.NoError(t, err)
	_, err = k.FinalizeRing(ctx, &types.MsgFinalizeRing{Creator: peer1Addr, RingId: createRingResp.RingId, RingPk: "ring-pk"})
	require.NoError(t, err)
	_, err = k.FinalizeRing(ctx, &types.MsgFinalizeRing{Creator: peer2Addr, RingId: createRingResp.RingId, RingPk: "ring-pk"})
	require.NoError(t, err)

	missingNodeInfoReporting := types.DefaultReportingConfig()
	missingNodeInfoReporting.BackupNodeKeys = []string{missingBackupKey}
	_, err = k.SetRingReportingByAcp(ctx, &types.MsgSetRingReportingByAcp{
		Creator:   creatorAddr,
		RingId:    createRingResp.RingId,
		Reporting: missingNodeInfoReporting,
	})
	require.ErrorIs(t, err, types.ErrInvalidRing)
	require.ErrorContains(t, err, "backup_node_key")
	require.ErrorContains(t, err, "has no registered node info")
	require.Equal(t, types.DefaultReportingConfig(), k.GetRing(ctx, createRingResp.RingId).Reporting)

	notWhitelistedReporting := types.DefaultReportingConfig()
	notWhitelistedReporting.BackupNodeKeys = []string{backupNoWhitelistKey}
	_, err = k.SetRingReportingByAcp(ctx, &types.MsgSetRingReportingByAcp{
		Creator:   creatorAddr,
		RingId:    createRingResp.RingId,
		Reporting: notWhitelistedReporting,
	})
	require.ErrorIs(t, err, types.ErrInvalidRing)
	require.ErrorContains(t, err, "backup_node_key")
	require.ErrorContains(t, err, "is not whitelisted")
	require.Equal(t, types.DefaultReportingConfig(), k.GetRing(ctx, createRingResp.RingId).Reporting)

	updatePeerNodeWhitelists(t, k, ctx, backupPolicyAddr, backupPolicyKey, []string{policyID}, nil)
	policyWhitelistedReporting := types.DefaultReportingConfig()
	policyWhitelistedReporting.BackupNodeKeys = []string{backupPolicyKey}
	_, err = k.SetRingReportingByAcp(ctx, &types.MsgSetRingReportingByAcp{
		Creator:   creatorAddr,
		RingId:    createRingResp.RingId,
		Reporting: policyWhitelistedReporting,
	})
	require.NoError(t, err)
	require.Equal(t, policyWhitelistedReporting, k.GetRing(ctx, createRingResp.RingId).Reporting)

	updatePeerNodeWhitelists(t, k, ctx, backupRingAddr, backupRingKey, nil, []string{createRingResp.RingId})
	ringWhitelistedReporting := types.DefaultReportingConfig()
	ringWhitelistedReporting.BackupNodeKeys = []string{backupRingKey, backupPolicyKey}
	_, err = k.SetRingReportingByAcp(ctx, &types.MsgSetRingReportingByAcp{
		Creator:   creatorAddr,
		RingId:    createRingResp.RingId,
		Reporting: ringWhitelistedReporting,
	})
	require.NoError(t, err)
	require.Equal(t, []string{backupRingKey, backupPolicyKey}, k.GetRing(ctx, createRingResp.RingId).Reporting.BackupNodeKeys)
}

func TestMsgServer_StartRingReshareByAcpRejectsThresholdOnlyAboveExistingCommitteeSize(t *testing.T) {
	k, authKeeper, ctx := setupOrbisKeeper(t)
	ctx = ctxWithDID(ctx, testDID)

	creatorAddr, _ := testAccountWithPubKey(t, ctx, authKeeper)

	peer1Addr, peer1Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer1")
	peer2Addr, peer2Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer2")
	policyID := createOrbisRingPolicy(t, k, ctx, creatorAddr)

	createRingResp, err := k.CreateRing(ctx, &types.MsgCreateRing{
		Creator:      creatorAddr,
		PeerNodeKeys: []string{peer1Key, peer2Key},
		Threshold:    1,
		PssInterval:  types.MinPSSIntervalSeconds,
		PolicyId:     policyID,
	})
	require.NoError(t, err)

	_, err = k.FinalizeRing(ctx, &types.MsgFinalizeRing{Creator: peer1Addr, RingId: createRingResp.RingId, RingPk: "ring-pk"})
	require.NoError(t, err)
	_, err = k.FinalizeRing(ctx, &types.MsgFinalizeRing{Creator: peer2Addr, RingId: createRingResp.RingId, RingPk: "ring-pk"})
	require.NoError(t, err)

	_, err = k.StartRingReshareByAcp(ctx, &types.MsgStartRingReshareByAcp{
		Creator: creatorAddr,
		RingId:  createRingResp.RingId,
	})
	require.ErrorContains(t, err, "reshare must change the committee or threshold")

	_, err = k.StartRingReshareByAcp(ctx, &types.MsgStartRingReshareByAcp{
		Creator: creatorAddr,
		RingId:  createRingResp.RingId,
		XNewThreshold: &types.MsgStartRingReshareByAcp_NewThreshold{
			NewThreshold: 1,
		},
	})
	require.ErrorContains(t, err, "reshare must change the committee or threshold")

	_, err = k.StartRingReshareByAcp(ctx, &types.MsgStartRingReshareByAcp{
		Creator: creatorAddr,
		RingId:  createRingResp.RingId,
		XNewThreshold: &types.MsgStartRingReshareByAcp_NewThreshold{
			NewThreshold: 3,
		},
	})
	require.ErrorIs(t, err, types.ErrInvalidRing)
	require.ErrorContains(t, err, "new_threshold (3) cannot exceed existing committee size (2)")

	ring := k.GetRing(ctx, createRingResp.RingId)
	require.NotNil(t, ring)
	require.Nil(t, ring.XNewThreshold)
}

func TestMsgServer_StartRingReshareByAcpRejectsCommitteeOnlyWhenThresholdExceedsTargetSize(t *testing.T) {
	k, authKeeper, ctx := setupOrbisKeeper(t)
	ctx = ctxWithDID(ctx, testDID)

	creatorAddr, _ := testAccountWithPubKey(t, ctx, authKeeper)

	peer1Addr, peer1Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer1")
	peer2Addr, peer2Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer2")
	policyID := createOrbisRingPolicy(t, k, ctx, creatorAddr)

	createRingResp, err := k.CreateRing(ctx, &types.MsgCreateRing{
		Creator:      creatorAddr,
		PeerNodeKeys: []string{peer1Key, peer2Key},
		Threshold:    2,
		PssInterval:  types.MinPSSIntervalSeconds,
		PolicyId:     policyID,
	})
	require.NoError(t, err)

	_, err = k.FinalizeRing(ctx, &types.MsgFinalizeRing{Creator: peer1Addr, RingId: createRingResp.RingId, RingPk: "ring-pk"})
	require.NoError(t, err)
	_, err = k.FinalizeRing(ctx, &types.MsgFinalizeRing{Creator: peer2Addr, RingId: createRingResp.RingId, RingPk: "ring-pk"})
	require.NoError(t, err)

	peer3Addr, peer3Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer3")
	updatePeerNodeWhitelists(t, k, ctx, peer3Addr, peer3Key, []string{policyID}, nil)
	_, err = k.StartRingReshareByAcp(ctx, &types.MsgStartRingReshareByAcp{
		Creator:         creatorAddr,
		RingId:          createRingResp.RingId,
		NewPeerNodeKeys: []string{peer3Key},
	})
	require.ErrorIs(t, err, types.ErrInvalidRing)
	require.ErrorContains(t, err, "effective new_threshold (2) cannot exceed target committee size (1)")

	ring := k.GetRing(ctx, createRingResp.RingId)
	require.NotNil(t, ring)
	require.Empty(t, ring.NewPeerNodeKeys)
	require.Nil(t, ring.XNewThreshold)
}

func TestMsgServer_StartRingReshareByAcpRejectsReorderedExistingCommittee(t *testing.T) {
	k, authKeeper, ctx := setupOrbisKeeper(t)
	ctx = ctxWithDID(ctx, testDID)

	creatorAddr, _ := testAccountWithPubKey(t, ctx, authKeeper)
	peer1Addr, peer1Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer1")
	peer2Addr, peer2Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer2")
	policyID := createOrbisRingPolicy(t, k, ctx, creatorAddr)

	createRingResp, err := k.CreateRing(ctx, &types.MsgCreateRing{
		Creator:      creatorAddr,
		PeerNodeKeys: []string{peer1Key, peer2Key},
		Threshold:    1,
		PssInterval:  types.MinPSSIntervalSeconds,
		PolicyId:     policyID,
	})
	require.NoError(t, err)

	_, err = k.FinalizeRing(ctx, &types.MsgFinalizeRing{Creator: peer1Addr, RingId: createRingResp.RingId, RingPk: "ring-pk"})
	require.NoError(t, err)
	_, err = k.FinalizeRing(ctx, &types.MsgFinalizeRing{Creator: peer2Addr, RingId: createRingResp.RingId, RingPk: "ring-pk"})
	require.NoError(t, err)

	storedCommittee := k.GetRing(ctx, createRingResp.RingId).PeerNodeKeys
	reorderedCommittee := []string{storedCommittee[1], storedCommittee[0]}
	_, err = k.StartRingReshareByAcp(ctx, &types.MsgStartRingReshareByAcp{
		Creator:         creatorAddr,
		RingId:          createRingResp.RingId,
		NewPeerNodeKeys: reorderedCommittee,
	})
	require.ErrorIs(t, err, types.ErrInvalidRing)
	require.ErrorContains(t, err, "reshare must change the committee or threshold")

	ring := k.GetRing(ctx, createRingResp.RingId)
	require.Empty(t, ring.NewPeerNodeKeys)
	require.Equal(t, []string{storedCommittee[1], storedCommittee[0]}, reorderedCommittee)
}

func TestMsgServer_StartRingReshareByAcpAllowsCommitteeOnlyForSingleNodeTarget(t *testing.T) {
	k, authKeeper, ctx := setupOrbisKeeper(t)
	ctx = ctxWithDID(ctx, testDID)

	creatorAddr, _ := testAccountWithPubKey(t, ctx, authKeeper)

	peer1Addr, peer1Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer1")
	peer2Addr, peer2Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer2")
	policyID := createOrbisRingPolicy(t, k, ctx, creatorAddr)

	createRingResp, err := k.CreateRing(ctx, &types.MsgCreateRing{
		Creator:      creatorAddr,
		PeerNodeKeys: []string{peer1Key, peer2Key},
		Threshold:    1,
		PssInterval:  types.MinPSSIntervalSeconds,
		PolicyId:     policyID,
	})
	require.NoError(t, err)

	_, err = k.FinalizeRing(ctx, &types.MsgFinalizeRing{Creator: peer1Addr, RingId: createRingResp.RingId, RingPk: "ring-pk"})
	require.NoError(t, err)
	_, err = k.FinalizeRing(ctx, &types.MsgFinalizeRing{Creator: peer2Addr, RingId: createRingResp.RingId, RingPk: "ring-pk"})
	require.NoError(t, err)

	peer3Addr, peer3Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer3")
	updatePeerNodeWhitelists(t, k, ctx, peer3Addr, peer3Key, []string{policyID}, nil)
	_, err = k.StartRingReshareByAcp(ctx, &types.MsgStartRingReshareByAcp{
		Creator:         creatorAddr,
		RingId:          createRingResp.RingId,
		NewPeerNodeKeys: []string{peer3Key},
	})
	require.NoError(t, err)

	ring := k.GetRing(ctx, createRingResp.RingId)
	require.NotNil(t, ring)
	require.Equal(t, []string{peer3Key}, ring.NewPeerNodeKeys)
	require.Nil(t, ring.XNewThreshold)

	_, err = k.StartRingReshareByAcp(ctx, &types.MsgStartRingReshareByAcp{
		Creator: creatorAddr,
		RingId:  createRingResp.RingId,
		XNewThreshold: &types.MsgStartRingReshareByAcp_NewThreshold{
			NewThreshold: 1,
		},
	})
	require.ErrorIs(t, err, types.ErrReshareInProgress)
}

func TestMsgServer_StartRingReshareByAcpAllowsCommitteeOnlyForSameSizeTarget(t *testing.T) {
	k, authKeeper, ctx := setupOrbisKeeper(t)
	ctx = ctxWithDID(ctx, testDID)

	creatorAddr, _ := testAccountWithPubKey(t, ctx, authKeeper)

	peer1Addr, peer1Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer1")
	peer2Addr, peer2Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer2")
	policyID := createOrbisRingPolicy(t, k, ctx, creatorAddr)

	createRingResp, err := k.CreateRing(ctx, &types.MsgCreateRing{
		Creator:      creatorAddr,
		PeerNodeKeys: []string{peer1Key, peer2Key},
		Threshold:    2,
		PssInterval:  types.MinPSSIntervalSeconds,
		PolicyId:     policyID,
	})
	require.NoError(t, err)

	_, err = k.FinalizeRing(ctx, &types.MsgFinalizeRing{Creator: peer1Addr, RingId: createRingResp.RingId, RingPk: "ring-pk"})
	require.NoError(t, err)
	_, err = k.FinalizeRing(ctx, &types.MsgFinalizeRing{Creator: peer2Addr, RingId: createRingResp.RingId, RingPk: "ring-pk"})
	require.NoError(t, err)

	peer3Addr, peer3Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer3")
	peer4Addr, peer4Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer4")
	updatePeerNodeWhitelists(t, k, ctx, peer3Addr, peer3Key, []string{policyID}, nil)
	updatePeerNodeWhitelists(t, k, ctx, peer4Addr, peer4Key, []string{policyID}, nil)
	canonicalCommittee := canonicalStrings([]string{peer3Key, peer4Key})
	submittedCommittee := []string{canonicalCommittee[1], canonicalCommittee[0]}
	_, err = k.StartRingReshareByAcp(ctx, &types.MsgStartRingReshareByAcp{
		Creator:         creatorAddr,
		RingId:          createRingResp.RingId,
		NewPeerNodeKeys: submittedCommittee,
	})
	require.NoError(t, err)

	ring := k.GetRing(ctx, createRingResp.RingId)
	require.NotNil(t, ring)
	require.Equal(t, canonicalCommittee, ring.NewPeerNodeKeys)
	require.Equal(t, []string{canonicalCommittee[1], canonicalCommittee[0]}, submittedCommittee)
	require.Nil(t, ring.XNewThreshold)
}

func TestMsgServer_StartRingReshareByAcpAllowsCommitteeWithLowerThreshold(t *testing.T) {
	k, authKeeper, ctx := setupOrbisKeeper(t)
	ctx = ctxWithDID(ctx, testDID)

	creatorAddr, _ := testAccountWithPubKey(t, ctx, authKeeper)

	peer1Addr, peer1Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer1")
	peer2Addr, peer2Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer2")
	policyID := createOrbisRingPolicy(t, k, ctx, creatorAddr)

	createRingResp, err := k.CreateRing(ctx, &types.MsgCreateRing{
		Creator:      creatorAddr,
		PeerNodeKeys: []string{peer1Key, peer2Key},
		Threshold:    2,
		PssInterval:  types.MinPSSIntervalSeconds,
		PolicyId:     policyID,
	})
	require.NoError(t, err)

	_, err = k.FinalizeRing(ctx, &types.MsgFinalizeRing{Creator: peer1Addr, RingId: createRingResp.RingId, RingPk: "ring-pk"})
	require.NoError(t, err)
	_, err = k.FinalizeRing(ctx, &types.MsgFinalizeRing{Creator: peer2Addr, RingId: createRingResp.RingId, RingPk: "ring-pk"})
	require.NoError(t, err)

	peer3Addr, peer3Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer3")
	updatePeerNodeWhitelists(t, k, ctx, peer3Addr, peer3Key, []string{policyID}, nil)
	_, err = k.StartRingReshareByAcp(ctx, &types.MsgStartRingReshareByAcp{
		Creator:         creatorAddr,
		RingId:          createRingResp.RingId,
		NewPeerNodeKeys: []string{peer3Key},
		XNewThreshold: &types.MsgStartRingReshareByAcp_NewThreshold{
			NewThreshold: 1,
		},
	})
	require.NoError(t, err)

	ring := k.GetRing(ctx, createRingResp.RingId)
	require.NotNil(t, ring)
	require.Equal(t, []string{peer3Key}, ring.NewPeerNodeKeys)
	require.Equal(t, uint32(1), ring.GetNewThreshold())
}

func TestMsgServer_StartRingReshareByAcpRejectsNewPeerWithoutNodeInfo(t *testing.T) {
	k, authKeeper, ctx := setupOrbisKeeper(t)
	ctx = ctxWithDID(ctx, testDID)

	creatorAddr, _ := testAccountWithPubKey(t, ctx, authKeeper)

	peer1Addr, peer1Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer1")
	peer2Addr, peer2Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer2")
	policyID := createOrbisRingPolicy(t, k, ctx, creatorAddr)

	createRingResp, err := k.CreateRing(ctx, &types.MsgCreateRing{
		Creator:      creatorAddr,
		PeerNodeKeys: []string{peer1Key, peer2Key},
		Threshold:    1,
		PssInterval:  types.MinPSSIntervalSeconds,
		PolicyId:     policyID,
	})
	require.NoError(t, err)

	_, err = k.FinalizeRing(ctx, &types.MsgFinalizeRing{Creator: peer1Addr, RingId: createRingResp.RingId, RingPk: "ring-pk"})
	require.NoError(t, err)
	_, err = k.FinalizeRing(ctx, &types.MsgFinalizeRing{Creator: peer2Addr, RingId: createRingResp.RingId, RingPk: "ring-pk"})
	require.NoError(t, err)

	_, missingNodeInfoKey := testAccountWithPubKey(t, ctx, authKeeper)
	_, err = k.StartRingReshareByAcp(ctx, &types.MsgStartRingReshareByAcp{
		Creator:         creatorAddr,
		RingId:          createRingResp.RingId,
		NewPeerNodeKeys: []string{missingNodeInfoKey},
		XNewThreshold: &types.MsgStartRingReshareByAcp_NewThreshold{
			NewThreshold: 1,
		},
	})
	require.ErrorIs(t, err, types.ErrInvalidRing)
	require.ErrorContains(t, err, fmt.Sprintf("peer_node_key %q has no registered node info", missingNodeInfoKey))

	ring := k.GetRing(ctx, createRingResp.RingId)
	require.NotNil(t, ring)
	require.Empty(t, ring.NewPeerNodeKeys)
	require.Nil(t, ring.XNewThreshold)
}

func TestMsgServer_StartRingReshareByAcpRejectsNewPeerWithoutWhitelist(t *testing.T) {
	k, authKeeper, ctx := setupOrbisKeeper(t)
	ctx = ctxWithDID(ctx, testDID)

	creatorAddr, _ := testAccountWithPubKey(t, ctx, authKeeper)

	peer1Addr, peer1Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer1")
	peer2Addr, peer2Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer2")
	policyID := createOrbisRingPolicy(t, k, ctx, creatorAddr)

	createRingResp, err := k.CreateRing(ctx, &types.MsgCreateRing{
		Creator:      creatorAddr,
		PeerNodeKeys: []string{peer1Key, peer2Key},
		Threshold:    1,
		PssInterval:  types.MinPSSIntervalSeconds,
		PolicyId:     policyID,
	})
	require.NoError(t, err)

	_, err = k.FinalizeRing(ctx, &types.MsgFinalizeRing{Creator: peer1Addr, RingId: createRingResp.RingId, RingPk: "ring-pk"})
	require.NoError(t, err)
	_, err = k.FinalizeRing(ctx, &types.MsgFinalizeRing{Creator: peer2Addr, RingId: createRingResp.RingId, RingPk: "ring-pk"})
	require.NoError(t, err)

	_, peer3Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer3")
	_, err = k.StartRingReshareByAcp(ctx, &types.MsgStartRingReshareByAcp{
		Creator:         creatorAddr,
		RingId:          createRingResp.RingId,
		NewPeerNodeKeys: []string{peer3Key},
		XNewThreshold: &types.MsgStartRingReshareByAcp_NewThreshold{
			NewThreshold: 1,
		},
	})
	require.ErrorIs(t, err, types.ErrInvalidRing)
	require.ErrorContains(
		t,
		err,
		fmt.Sprintf("peer_node_key %q is not whitelisted for policy_id %q or ring_id %q", peer3Key, policyID, createRingResp.RingId),
	)

	ring := k.GetRing(ctx, createRingResp.RingId)
	require.NotNil(t, ring)
	require.Empty(t, ring.NewPeerNodeKeys)
	require.Nil(t, ring.XNewThreshold)
}

func TestMsgServer_RingMutationsRejectUnauthorizedActor(t *testing.T) {
	k, authKeeper, ctx := setupOrbisKeeper(t)
	creatorCtx := ctxWithDID(ctx, testDID)
	peerCtx := ctxWithDID(ctx, testPeerDID)
	outsiderCtx := ctxWithDID(ctx, testOutsiderDID)

	creatorAddr, _ := testAccountWithPubKey(t, creatorCtx, authKeeper)

	peer1Addr, peer1Key := setupPeerWithNodeInfo(t, k, authKeeper, creatorCtx, "12D3KooWPeer1")
	peer2Addr, peer2Key := setupPeerWithNodeInfo(t, k, authKeeper, creatorCtx, "12D3KooWPeer2")
	policyID := createOrbisRingPolicy(t, k, creatorCtx, creatorAddr)

	createRingResp, err := k.CreateRing(creatorCtx, &types.MsgCreateRing{
		Creator:      creatorAddr,
		PeerNodeKeys: []string{peer1Key, peer2Key},
		Threshold:    1,
		PssInterval:  types.MinPSSIntervalSeconds,
		PolicyId:     policyID,
	})
	require.NoError(t, err)

	_, err = k.FinalizeRing(peerCtx, &types.MsgFinalizeRing{Creator: peer1Addr, RingId: createRingResp.RingId, RingPk: "ring-pk"})
	require.NoError(t, err)
	_, err = k.FinalizeRing(peerCtx, &types.MsgFinalizeRing{Creator: peer2Addr, RingId: createRingResp.RingId, RingPk: "ring-pk"})
	require.NoError(t, err)

	outsiderAddr, _ := testAccountWithPubKey(t, outsiderCtx, authKeeper)
	_, err = k.SetRingPssIntervalByAcp(outsiderCtx, &types.MsgSetRingPssIntervalByAcp{
		Creator:     outsiderAddr,
		RingId:      createRingResp.RingId,
		PssInterval: types.MinPSSIntervalSeconds,
	})
	require.ErrorIs(t, err, types.ErrUnauthorizedRingUpdate)

	_, err = k.ScheduleRingUpgradeByAcp(outsiderCtx, &types.MsgScheduleRingUpgradeByAcp{
		Creator:        outsiderAddr,
		RingId:         createRingResp.RingId,
		NextVersion:    1,
		ActivationTime: MinRingUpgradeLeadSeconds,
	})
	require.ErrorIs(t, err, types.ErrUnauthorizedRingUpdate)

	_, err = k.CancelRingUpgradeByAcp(outsiderCtx, &types.MsgCancelRingUpgradeByAcp{
		Creator: outsiderAddr,
		RingId:  createRingResp.RingId,
	})
	require.ErrorIs(t, err, types.ErrUnauthorizedRingUpdate)

	_, err = k.SetRingReportingByAcp(outsiderCtx, &types.MsgSetRingReportingByAcp{
		Creator:   outsiderAddr,
		RingId:    createRingResp.RingId,
		Reporting: types.DefaultReportingConfig(),
	})
	require.ErrorIs(t, err, types.ErrUnauthorizedRingUpdate)

	ring := k.GetRing(creatorCtx, createRingResp.RingId)
	require.NotNil(t, ring)
	require.Equal(t, types.MinPSSIntervalSeconds, ring.GetPssInterval())
	require.Equal(t, types.DefaultReportingConfig(), ring.Reporting)
	require.Nil(t, ring.UpgradeInfo.XNextVersion)
	require.Nil(t, ring.UpgradeInfo.XActivationTime)
}

func TestMsgServer_RingMutationUsesRingPolicy(t *testing.T) {
	k, authKeeper, ctx := setupOrbisKeeper(t)
	creatorCtx := ctxWithDID(ctx, testDID)
	peerCtx := ctxWithDID(ctx, testPeerDID)
	operatorCtx := ctxWithDID(ctx, testOperatorDID)

	creatorAddr, _ := testAccountWithPubKey(t, creatorCtx, authKeeper)

	peer1Addr, peer1Key := setupPeerWithNodeInfo(t, k, authKeeper, creatorCtx, "12D3KooWPeer1")
	peer2Addr, peer2Key := setupPeerWithNodeInfo(t, k, authKeeper, creatorCtx, "12D3KooWPeer2")
	policyID := createOrbisRingPolicy(t, k, creatorCtx, creatorAddr)

	createRingResp, err := k.CreateRing(creatorCtx, &types.MsgCreateRing{
		Creator:      creatorAddr,
		PeerNodeKeys: []string{peer1Key, peer2Key},
		Threshold:    1,
		PssInterval:  types.MinPSSIntervalSeconds,
		PolicyId:     policyID,
	})
	require.NoError(t, err)

	_, err = k.FinalizeRing(peerCtx, &types.MsgFinalizeRing{Creator: peer1Addr, RingId: createRingResp.RingId, RingPk: "ring-pk"})
	require.NoError(t, err)
	_, err = k.FinalizeRing(peerCtx, &types.MsgFinalizeRing{Creator: peer2Addr, RingId: createRingResp.RingId, RingPk: "ring-pk"})
	require.NoError(t, err)

	operatorAddr, _ := testAccountWithPubKey(t, operatorCtx, authKeeper)
	otherPolicyID := createOrbisRingPolicy(t, k, creatorCtx, creatorAddr)
	_, err = k.GetAcpKeeper().DirectPolicyCmd(creatorCtx, &acptypes.MsgDirectPolicyCmd{
		Creator:  creatorAddr,
		PolicyId: otherPolicyID,
		Cmd:      acptypes.NewRegisterObjectCmd(coretypes.NewObject(types.ACPResourceRing, createRingResp.RingId)),
	})
	require.NoError(t, err)
	_, err = k.GetAcpKeeper().DirectPolicyCmd(creatorCtx, &acptypes.MsgDirectPolicyCmd{
		Creator:  creatorAddr,
		PolicyId: otherPolicyID,
		Cmd: acptypes.NewSetRelationshipCmd(coretypes.NewActorRelationship(
			types.ACPResourceRing,
			createRingResp.RingId,
			types.ACPRelationOperator,
			testOperatorDID,
		)),
	})
	require.NoError(t, err)

	_, err = k.SetRingPssIntervalByAcp(operatorCtx, &types.MsgSetRingPssIntervalByAcp{
		Creator:     operatorAddr,
		RingId:      createRingResp.RingId,
		PssInterval: types.MinPSSIntervalSeconds,
	})
	require.ErrorIs(t, err, types.ErrUnauthorizedRingUpdate)
	require.Equal(t, policyID, k.GetRing(creatorCtx, createRingResp.RingId).PolicyId)
}

func TestMsgServer_StartRingReshareByAcpAllowsOperatorRelation(t *testing.T) {
	k, authKeeper, ctx := setupOrbisKeeper(t)
	creatorCtx := ctxWithDID(ctx, testDID)
	peerCtx := ctxWithDID(ctx, testPeerDID)
	operatorCtx := ctxWithDID(ctx, testOperatorDID)

	creatorAddr, _ := testAccountWithPubKey(t, creatorCtx, authKeeper)

	peer1Addr, peer1Key := setupPeerWithNodeInfo(t, k, authKeeper, creatorCtx, "12D3KooWPeer1")
	peer2Addr, peer2Key := setupPeerWithNodeInfo(t, k, authKeeper, creatorCtx, "12D3KooWPeer2")
	policyID := createOrbisRingPolicy(t, k, creatorCtx, creatorAddr)

	createRingResp, err := k.CreateRing(creatorCtx, &types.MsgCreateRing{
		Creator:      creatorAddr,
		PeerNodeKeys: []string{peer1Key, peer2Key},
		Threshold:    1,
		PssInterval:  types.MinPSSIntervalSeconds,
		PolicyId:     policyID,
	})
	require.NoError(t, err)

	_, err = k.FinalizeRing(peerCtx, &types.MsgFinalizeRing{Creator: peer1Addr, RingId: createRingResp.RingId, RingPk: "ring-pk"})
	require.NoError(t, err)
	_, err = k.FinalizeRing(peerCtx, &types.MsgFinalizeRing{Creator: peer2Addr, RingId: createRingResp.RingId, RingPk: "ring-pk"})
	require.NoError(t, err)

	operatorAddr, _ := testAccountWithPubKey(t, operatorCtx, authKeeper)
	_, err = k.GetAcpKeeper().DirectPolicyCmd(creatorCtx, &acptypes.MsgDirectPolicyCmd{
		Creator:  creatorAddr,
		PolicyId: policyID,
		Cmd: acptypes.NewSetRelationshipCmd(coretypes.NewActorRelationship(
			types.ACPResourceRing,
			createRingResp.RingId,
			types.ACPRelationOperator,
			testOperatorDID,
		)),
	})
	require.NoError(t, err)

	peer3Addr, peer3Key := setupPeerWithNodeInfo(t, k, authKeeper, creatorCtx, "12D3KooWPeer3")
	peer4Addr, peer4Key := setupPeerWithNodeInfo(t, k, authKeeper, creatorCtx, "12D3KooWPeer4")
	updatePeerNodeWhitelists(t, k, creatorCtx, peer3Addr, peer3Key, []string{policyID}, nil)
	updatePeerNodeWhitelists(t, k, creatorCtx, peer4Addr, peer4Key, nil, []string{createRingResp.RingId})
	_, err = k.StartRingReshareByAcp(operatorCtx, &types.MsgStartRingReshareByAcp{
		Creator:         operatorAddr,
		RingId:          createRingResp.RingId,
		NewPeerNodeKeys: []string{peer3Key, peer4Key},
		XNewThreshold: &types.MsgStartRingReshareByAcp_NewThreshold{
			NewThreshold: 1,
		},
	})
	require.NoError(t, err)

	ring := k.GetRing(creatorCtx, createRingResp.RingId)
	require.NotNil(t, ring)
	require.Equal(t, policyID, ring.PolicyId)
	require.Equal(t, canonicalStrings([]string{peer3Key, peer4Key}), ring.NewPeerNodeKeys)
	require.Equal(t, uint32(1), ring.GetNewThreshold())
}

func TestMsgServer_FinalizeRingReshareRequiresPendingUpdate(t *testing.T) {
	k, authKeeper, ctx := setupOrbisKeeper(t)
	ctx = ctx.WithValue(appparams.ExtractedDIDContextKey, testDID)

	creatorAddr, _ := testAccountWithPubKey(t, ctx, authKeeper)

	_, peer1Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer1")
	_, peer2Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWPeer2")
	policyID := createOrbisRingPolicy(t, k, ctx, creatorAddr)

	createRingResp, err := k.CreateRing(ctx, &types.MsgCreateRing{
		Creator:      creatorAddr,
		PeerNodeKeys: []string{peer1Key, peer2Key},
		Threshold:    1,
		PssInterval:  types.MinPSSIntervalSeconds,
		PolicyId:     policyID,
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
	k, authKeeper, ctx := setupOrbisKeeper(t)
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
	require.Empty(t, nodeInfo.WhitelistedPolicyIds)
	require.Empty(t, nodeInfo.WhitelistedRingIds)
}

func TestMsgServer_CreateNodeInfo_WithWhitelists(t *testing.T) {
	k, authKeeper, ctx := setupOrbisKeeper(t)
	ctx = ctx.WithValue(appparams.ExtractedDIDContextKey, testDID)

	nodeAddr, nodePubKeyHex := testAccountWithPubKey(t, ctx, authKeeper)
	_, controllerPubKeyHex := testAccountWithPubKey(t, ctx, authKeeper)

	_, err := k.CreateNodeInfo(ctx, &types.MsgCreateNodeInfo{
		Creator:              nodeAddr,
		PeerId:               "peer-1",
		ControllerKey:        controllerPubKeyHex,
		WhitelistedPolicyIds: []string{"policy-a", "policy-b"},
		WhitelistedRingIds:   []string{"ring-1"},
	})
	require.NoError(t, err)

	nodeInfo := k.GetNodeInfo(ctx, nodePubKeyHex)
	require.NotNil(t, nodeInfo)
	require.Equal(t, []string{"policy-a", "policy-b"}, nodeInfo.WhitelistedPolicyIds)
	require.Equal(t, []string{"ring-1"}, nodeInfo.WhitelistedRingIds)
}

func TestMsgServer_CreateNodeInfo_CanonicalizesControllerKey(t *testing.T) {
	k, authKeeper, ctx := setupOrbisKeeper(t)
	ctx = ctx.WithValue(appparams.ExtractedDIDContextKey, testDID)

	nodeAddr, nodePubKeyHex := testAccountWithPubKey(t, ctx, authKeeper)
	controllerAddr, controllerPubKeyHex := testAccountWithPubKey(t, ctx, authKeeper)

	_, err := k.CreateNodeInfo(ctx, &types.MsgCreateNodeInfo{
		Creator:       nodeAddr,
		PeerId:        "peer-1",
		ControllerKey: "0x" + strings.ToUpper(controllerPubKeyHex),
	})
	require.NoError(t, err)

	nodeInfo := k.GetNodeInfo(ctx, nodePubKeyHex)
	require.NotNil(t, nodeInfo)
	require.Equal(t, controllerPubKeyHex, nodeInfo.ControllerKey)

	_, err = k.AddNodeToWhitelist(ctx, &types.MsgAddNodeToWhitelist{
		Creator: controllerAddr,
		NodeKey: nodePubKeyHex,
		Target:  &types.MsgAddNodeToWhitelist_RingId{RingId: "ring-1"},
	})
	require.NoError(t, err)
}

func TestMsgServer_CreateNodeInfo_AlreadyExists(t *testing.T) {
	k, authKeeper, ctx := setupOrbisKeeper(t)
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

func TestMsgServer_CreateNodeInfoRejectsInvalidWhitelistSets(t *testing.T) {
	k, authKeeper, ctx := setupOrbisKeeper(t)
	ctx = ctx.WithValue(appparams.ExtractedDIDContextKey, testDID)

	for _, testCase := range []struct {
		name      string
		policyIDs []string
		ringIDs   []string
		errorText string
	}{
		{name: "empty policy", policyIDs: []string{""}, errorText: "contains an empty value"},
		{name: "duplicate policy", policyIDs: []string{"policy-a", "policy-a"}, errorText: "contains duplicate value"},
		{name: "empty ring", ringIDs: []string{""}, errorText: "contains an empty value"},
		{name: "duplicate ring", ringIDs: []string{"ring-a", "ring-a"}, errorText: "contains duplicate value"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			nodeAddr, _ := testAccountWithPubKey(t, ctx, authKeeper)
			_, controllerPubKeyHex := testAccountWithPubKey(t, ctx, authKeeper)
			_, err := k.CreateNodeInfo(ctx, &types.MsgCreateNodeInfo{
				Creator:              nodeAddr,
				PeerId:               "peer-1",
				ControllerKey:        controllerPubKeyHex,
				WhitelistedPolicyIds: testCase.policyIDs,
				WhitelistedRingIds:   testCase.ringIDs,
			})
			require.ErrorContains(t, err, testCase.errorText)
		})
	}
}

func TestMsgServer_UpdateNodePeerId(t *testing.T) {
	k, authKeeper, ctx := setupOrbisKeeper(t)
	ctx = ctx.WithValue(appparams.ExtractedDIDContextKey, testDID)

	nodeAddr, nodePubKeyHex := testAccountWithPubKey(t, ctx, authKeeper)
	controllerAddr, controllerPubKeyHex := testAccountWithPubKey(t, ctx, authKeeper)
	wrongAddr, _ := testAccountWithPubKey(t, ctx, authKeeper)
	_, err := k.CreateNodeInfo(ctx, &types.MsgCreateNodeInfo{
		Creator:       nodeAddr,
		PeerId:        "peer-original",
		ControllerKey: controllerPubKeyHex,
	})
	require.NoError(t, err)
	ctx = ctx.WithEventManager(sdk.NewEventManager())

	_, err = k.UpdateNodePeerId(ctx, &types.MsgUpdateNodePeerId{
		Creator: controllerAddr,
		NodeKey: nodePubKeyHex,
		PeerId:  "peer-updated",
	})
	require.NoError(t, err)
	require.Equal(t, "peer-updated", k.GetNodeInfo(ctx, nodePubKeyHex).PeerId)
	require.Equal(t, []proto.Message{
		&types.EventNodeInfoUpdated{
			PeerId:        "peer-updated",
			ControllerKey: controllerPubKeyHex,
		},
	}, parseTypedEvents(t, ctx))

	_, err = k.UpdateNodePeerId(ctx, &types.MsgUpdateNodePeerId{
		Creator: controllerAddr,
		NodeKey: nodePubKeyHex,
		PeerId:  "peer-updated",
	})
	require.ErrorIs(t, err, types.ErrNodeInfoUnchanged)

	_, err = k.UpdateNodePeerId(ctx, &types.MsgUpdateNodePeerId{
		Creator: wrongAddr,
		NodeKey: nodePubKeyHex,
		PeerId:  "peer-wrong",
	})
	require.ErrorIs(t, err, types.ErrUnauthorizedNodeInfoUpdate)

	_, err = k.UpdateNodePeerId(ctx, &types.MsgUpdateNodePeerId{
		Creator: controllerAddr,
		NodeKey: "missing",
		PeerId:  "peer-missing",
	})
	require.ErrorIs(t, err, types.ErrNodeInfoNotFound)
}

func TestMsgServer_TransferNodeController(t *testing.T) {
	k, authKeeper, ctx := setupOrbisKeeper(t)
	ctx = ctx.WithValue(appparams.ExtractedDIDContextKey, testDID)

	nodeAddr, nodePubKeyHex := testAccountWithPubKey(t, ctx, authKeeper)
	oldControllerAddr, oldControllerKey := testAccountWithPubKey(t, ctx, authKeeper)
	newControllerAddr, newControllerKey := testAccountWithPubKey(t, ctx, authKeeper)
	_, err := k.CreateNodeInfo(ctx, &types.MsgCreateNodeInfo{
		Creator:       nodeAddr,
		PeerId:        "peer-1",
		ControllerKey: oldControllerKey,
	})
	require.NoError(t, err)

	_, err = k.TransferNodeController(ctx, &types.MsgTransferNodeController{
		Creator:       oldControllerAddr,
		NodeKey:       nodePubKeyHex,
		ControllerKey: "0x" + strings.ToUpper(newControllerKey),
	})
	require.NoError(t, err)
	require.Equal(t, newControllerKey, k.GetNodeInfo(ctx, nodePubKeyHex).ControllerKey)

	_, err = k.UpdateNodePeerId(ctx, &types.MsgUpdateNodePeerId{
		Creator: oldControllerAddr,
		NodeKey: nodePubKeyHex,
		PeerId:  "old-controller",
	})
	require.ErrorIs(t, err, types.ErrUnauthorizedNodeInfoUpdate)

	_, err = k.UpdateNodePeerId(ctx, &types.MsgUpdateNodePeerId{
		Creator: newControllerAddr,
		NodeKey: nodePubKeyHex,
		PeerId:  "new-controller",
	})
	require.NoError(t, err)

	_, err = k.TransferNodeController(ctx, &types.MsgTransferNodeController{
		Creator:       newControllerAddr,
		NodeKey:       nodePubKeyHex,
		ControllerKey: newControllerKey,
	})
	require.ErrorIs(t, err, types.ErrNodeInfoUnchanged)

	_, err = k.TransferNodeController(ctx, &types.MsgTransferNodeController{
		Creator:       newControllerAddr,
		NodeKey:       nodePubKeyHex,
		ControllerKey: "not-a-key",
	})
	require.ErrorIs(t, err, types.ErrInvalidNodeInfo)
	require.Equal(t, newControllerKey, k.GetNodeInfo(ctx, nodePubKeyHex).ControllerKey)
}

func TestMsgServer_DrainNodeKey(t *testing.T) {
	k, authKeeper, bankKeeper, ctx := setupOrbisKeeperWithBank(t)
	ctx = ctx.WithValue(appparams.ExtractedDIDContextKey, testDID)

	nodeAddr, nodePubKeyHex := testAccountWithPubKey(t, ctx, authKeeper)
	controllerAddr, controllerPubKeyHex := testAccountWithPubKey(t, ctx, authKeeper)
	wrongAddr, _ := testAccountWithPubKey(t, ctx, authKeeper)
	_, err := k.CreateNodeInfo(ctx, &types.MsgCreateNodeInfo{
		Creator:       nodeAddr,
		PeerId:        "peer-1",
		ControllerKey: controllerPubKeyHex,
	})
	require.NoError(t, err)

	// No balance to drain yet.
	_, err = k.DrainNodeKey(ctx, &types.MsgDrainNodeKey{
		Creator: controllerAddr,
		NodeKey: nodePubKeyHex,
	})
	require.ErrorIs(t, err, types.ErrNodeKeyBalanceEmpty)

	// A signer that isn't the controller cannot drain.
	_, err = k.DrainNodeKey(ctx, &types.MsgDrainNodeKey{
		Creator: wrongAddr,
		NodeKey: nodePubKeyHex,
	})
	require.ErrorIs(t, err, types.ErrUnauthorizedNodeInfoUpdate)

	// Draining an unregistered node key fails.
	_, err = k.DrainNodeKey(ctx, &types.MsgDrainNodeKey{
		Creator: controllerAddr,
		NodeKey: "missing",
	})
	require.ErrorIs(t, err, types.ErrNodeInfoNotFound)

	nodeAccAddr, err := sdk.AccAddressFromBech32(nodeAddr)
	require.NoError(t, err)
	controllerAccAddr, err := sdk.AccAddressFromBech32(controllerAddr)
	require.NoError(t, err)
	balance := sdk.NewCoins(sdk.NewInt64Coin(appparams.DefaultBondDenom, 1000))
	fundAccount(t, ctx, bankKeeper, nodeAccAddr, balance)
	require.True(t, bankKeeper.SpendableCoins(ctx, nodeAccAddr).Equal(balance))

	ctx = ctx.WithEventManager(sdk.NewEventManager())
	resp, err := k.DrainNodeKey(ctx, &types.MsgDrainNodeKey{
		Creator: controllerAddr,
		NodeKey: nodePubKeyHex,
	})
	require.NoError(t, err)
	require.True(t, sdk.Coins(resp.Amount).Equal(balance))
	require.True(t, bankKeeper.SpendableCoins(ctx, nodeAccAddr).IsZero())
	require.True(t, bankKeeper.SpendableCoins(ctx, controllerAccAddr).Equal(balance))

	// SendCoins also emits bank's own legacy (non-typed) events alongside our typed
	// event, so filter for EventNodeKeyDrained specifically rather than requiring
	// every emitted event to be a typed proto event.
	var drained *types.EventNodeKeyDrained
	for _, event := range ctx.EventManager().Events().ToABCIEvents() {
		typedEvent, parseErr := sdk.ParseTypedEvent(event)
		if parseErr != nil {
			continue
		}
		if e, ok := typedEvent.(*types.EventNodeKeyDrained); ok {
			drained = e
		}
	}
	require.Equal(t, &types.EventNodeKeyDrained{
		NodeKey:   nodePubKeyHex,
		Recipient: controllerAddr,
		Amount:    balance,
	}, drained)
}

func TestMsgServer_AddAndRemoveNodeWhitelistEntries(t *testing.T) {
	k, authKeeper, ctx := setupOrbisKeeper(t)
	ctx = ctx.WithValue(appparams.ExtractedDIDContextKey, testDID)

	nodeAddr, nodePubKeyHex := testAccountWithPubKey(t, ctx, authKeeper)
	controllerAddr, controllerPubKeyHex := testAccountWithPubKey(t, ctx, authKeeper)
	_, err := k.CreateNodeInfo(ctx, &types.MsgCreateNodeInfo{
		Creator:              nodeAddr,
		PeerId:               "peer-1",
		ControllerKey:        controllerPubKeyHex,
		WhitelistedPolicyIds: []string{"policy-a"},
		WhitelistedRingIds:   []string{"ring-a"},
	})
	require.NoError(t, err)

	_, err = k.AddNodeToWhitelist(ctx, &types.MsgAddNodeToWhitelist{
		Creator: controllerAddr,
		NodeKey: nodePubKeyHex,
		Target:  &types.MsgAddNodeToWhitelist_PolicyId{PolicyId: "policy-b"},
	})
	require.NoError(t, err)
	_, err = k.AddNodeToWhitelist(ctx, &types.MsgAddNodeToWhitelist{
		Creator: controllerAddr,
		NodeKey: nodePubKeyHex,
		Target:  &types.MsgAddNodeToWhitelist_RingId{RingId: "ring-b"},
	})
	require.NoError(t, err)

	nodeInfo := k.GetNodeInfo(ctx, nodePubKeyHex)
	require.Equal(t, []string{"policy-a", "policy-b"}, nodeInfo.WhitelistedPolicyIds)
	require.Equal(t, []string{"ring-a", "ring-b"}, nodeInfo.WhitelistedRingIds)

	_, err = k.AddNodeToWhitelist(ctx, &types.MsgAddNodeToWhitelist{
		Creator: controllerAddr,
		NodeKey: nodePubKeyHex,
		Target:  &types.MsgAddNodeToWhitelist_PolicyId{PolicyId: "policy-b"},
	})
	require.ErrorIs(t, err, types.ErrNodeWhitelistEntryExists)

	_, err = k.RemoveNodeFromWhitelist(ctx, &types.MsgRemoveNodeFromWhitelist{
		Creator: controllerAddr,
		NodeKey: nodePubKeyHex,
		Target:  &types.MsgRemoveNodeFromWhitelist_PolicyId{PolicyId: "policy-a"},
	})
	require.NoError(t, err)
	_, err = k.RemoveNodeFromWhitelist(ctx, &types.MsgRemoveNodeFromWhitelist{
		Creator: controllerAddr,
		NodeKey: nodePubKeyHex,
		Target:  &types.MsgRemoveNodeFromWhitelist_RingId{RingId: "ring-a"},
	})
	require.NoError(t, err)

	nodeInfo = k.GetNodeInfo(ctx, nodePubKeyHex)
	require.Equal(t, []string{"policy-b"}, nodeInfo.WhitelistedPolicyIds)
	require.Equal(t, []string{"ring-b"}, nodeInfo.WhitelistedRingIds)

	_, err = k.RemoveNodeFromWhitelist(ctx, &types.MsgRemoveNodeFromWhitelist{
		Creator: controllerAddr,
		NodeKey: nodePubKeyHex,
		Target:  &types.MsgRemoveNodeFromWhitelist_RingId{RingId: "ring-missing"},
	})
	require.ErrorIs(t, err, types.ErrNodeWhitelistEntryNotFound)
}

// setupPeerWithNodeInfo registers an account and creates a NodeInfo entry for it.
// Returns the account bech32 address and its hex-encoded public key (the node_key used in rings).
func setupPeerWithNodeInfo(t *testing.T, k Keeper, ak authkeeper.AccountKeeper, ctx sdk.Context, networkID string) (addr string, nodeKey string) {
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

func updatePeerNodeWhitelists(
	t *testing.T,
	k Keeper,
	ctx sdk.Context,
	creator string,
	nodeKey string,
	policyIDs []string,
	ringIDs []string,
) {
	t.Helper()
	for _, policyID := range policyIDs {
		_, err := k.AddNodeToWhitelist(ctx, &types.MsgAddNodeToWhitelist{
			Creator: creator,
			NodeKey: nodeKey,
			Target:  &types.MsgAddNodeToWhitelist_PolicyId{PolicyId: policyID},
		})
		require.NoError(t, err)
	}
	for _, ringID := range ringIDs {
		_, err := k.AddNodeToWhitelist(ctx, &types.MsgAddNodeToWhitelist{
			Creator: creator,
			NodeKey: nodeKey,
			Target:  &types.MsgAddNodeToWhitelist_RingId{RingId: ringID},
		})
		require.NoError(t, err)
	}
}

func createOrbisRingPolicy(t *testing.T, k Keeper, ctx sdk.Context, creator string) string {
	t.Helper()
	policyID := createACPPolicy(t, k, ctx, creator, testOrbisRingPolicy)
	registerRingPolicyControlObject(t, k, ctx, creator, policyID)
	return policyID
}

func createACPPolicy(t *testing.T, k Keeper, ctx sdk.Context, creator string, policy string) string {
	t.Helper()
	resp, err := k.GetAcpKeeper().CreatePolicy(ctx, &acptypes.MsgCreatePolicy{
		Creator:     creator,
		Policy:      policy,
		MarshalType: coretypes.PolicyMarshalingType_YAML,
	})
	require.NoError(t, err)
	return resp.Record.Policy.Id
}

func registerRingPolicyControlObject(t *testing.T, k Keeper, ctx sdk.Context, creator string, policyID string) {
	t.Helper()
	_, err := k.GetAcpKeeper().DirectPolicyCmd(ctx, &acptypes.MsgDirectPolicyCmd{
		Creator:  creator,
		PolicyId: policyID,
		Cmd:      acptypes.NewRegisterObjectCmd(coretypes.NewObject(types.ACPResourceRingPolicy, policyID)),
	})
	require.NoError(t, err)
}

func grantRingCreator(t *testing.T, k Keeper, ctx sdk.Context, creator string, policyID string, actorDID string) {
	t.Helper()
	_, err := k.GetAcpKeeper().DirectPolicyCmd(ctx, &acptypes.MsgDirectPolicyCmd{
		Creator:  creator,
		PolicyId: policyID,
		Cmd: acptypes.NewSetRelationshipCmd(coretypes.NewActorRelationship(
			types.ACPResourceRingPolicy,
			policyID,
			types.ACPRelationRingCreator,
			actorDID,
		)),
	})
	require.NoError(t, err)
}

func ctxWithDID(ctx sdk.Context, did string) sdk.Context {
	return ctx.WithValue(appparams.ExtractedDIDContextKey, did)
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
