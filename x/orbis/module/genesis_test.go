package orbis

import (
	"testing"

	"github.com/stretchr/testify/require"

	keepertestutil "github.com/sourcenetwork/sourcehub/testutil/keeper"
	"github.com/sourcenetwork/sourcehub/x/orbis/types"
)

func TestGenesisRoundTrip(t *testing.T) {
	k, ctx := keepertestutil.OrbisKeeper(t)

	expectedRing := types.Ring{
		Id:               "ring-1",
		CreatorDid:       "did:example:creator",
		RingPk:           "ring-pk",
		PeerNodeKeys:     []string{"node-1", "node-2"},
		Threshold:        2,
		PssInterval:      types.MinPSSIntervalSeconds,
		BlockNumberNonce: 42,
		PolicyId:         "policy-1",
		UpgradeInfo: types.UpgradeInfo{
			CurrentVersion: 3,
		},
		DemeritConfig: types.DemeritConfig{
			NodeOfflineDemerits:  4,
			ResetIntervalSeconds: 30,
		},
	}
	expectedDocument := types.Document{
		Id:         "document-1",
		CreatorDid: "did:example:creator",
		RingId:     expectedRing.Id,
		Document:   "ciphertext",
		Proof:      "proof",
		PolicyId:   "policy-doc",
		Resource:   "secret",
		Permission: "decrypt",
	}
	expectedKeyDerivation := types.KeyDerivation{
		Id:         "key-derivation-1",
		CreatorDid: "did:example:creator",
		RingId:     expectedRing.Id,
		Derivation: "m/0",
		PolicyId:   "policy-key",
		Resource:   "key",
		Permission: "derive",
	}
	expectedNodeInfo := types.NodeInfoEntry{
		NodeKey: "node-1",
		NodeInfo: &types.NodeInfo{
			PeerId:               "12D3KooWGenesisPeer",
			ControllerKey:        "controller-key",
			WhitelistedPolicyIds: []string{"policy-1"},
			WhitelistedRingIds:   []string{expectedRing.Id},
		},
	}
	expectedNodeDemerits := []types.NodeDemeritEntry{
		{
			RingId:          expectedRing.Id,
			NodeKey:         "node-1",
			Points:          3,
			WindowStartedAt: 100,
		},
		{
			RingId:          expectedRing.Id,
			NodeKey:         "node-2",
			Points:          5,
			WindowStartedAt: 110,
		},
		{
			RingId:          "ring-2",
			NodeKey:         "node-1",
			Points:          7,
			WindowStartedAt: 120,
		},
	}

	k.SetRing(ctx, expectedRing)
	k.SetDocument(ctx, expectedDocument)
	k.SetKeyDerivation(ctx, expectedKeyDerivation)
	k.SetNodeInfo(ctx, expectedNodeInfo.NodeKey, *expectedNodeInfo.NodeInfo)
	for _, entry := range expectedNodeDemerits {
		k.SetNodeDemerits(ctx, entry.RingId, entry.NodeKey, entry.Points, entry.WindowStartedAt)
	}

	exported := ExportGenesis(ctx, &k)
	require.Equal(t, types.DefaultParams(), exported.Params)
	require.ElementsMatch(t, []types.Ring{expectedRing}, exported.Rings)
	require.ElementsMatch(t, []types.Document{expectedDocument}, exported.Documents)
	require.ElementsMatch(t, []types.KeyDerivation{expectedKeyDerivation}, exported.KeyDerivations)
	require.ElementsMatch(t, []types.NodeInfoEntry{expectedNodeInfo}, exported.NodeInfos)
	require.ElementsMatch(t, expectedNodeDemerits, exported.NodeDemerits)

	importedKeeper, importedCtx := keepertestutil.OrbisKeeper(t)
	InitGenesis(importedCtx, &importedKeeper, *exported)

	require.Equal(t, &expectedRing, importedKeeper.GetRing(importedCtx, expectedRing.Id))
	require.Equal(t, &expectedDocument, importedKeeper.GetDocument(importedCtx, expectedDocument.Id))
	require.Equal(t, &expectedKeyDerivation, importedKeeper.GetKeyDerivation(importedCtx, expectedKeyDerivation.Id))
	require.Equal(t, expectedNodeInfo.NodeInfo, importedKeeper.GetNodeInfo(importedCtx, expectedNodeInfo.NodeKey))
	for _, entry := range expectedNodeDemerits {
		require.Equal(t, entry.Points, importedKeeper.GetNodeDemerits(importedCtx, entry.RingId, entry.NodeKey))
	}
	require.Equal(t, uint64(3), importedKeeper.GetEffectiveNodeDemerits(importedCtx, expectedRing.Id, "node-1", 109, 10))
	require.Equal(t, uint64(0), importedKeeper.GetEffectiveNodeDemerits(importedCtx, expectedRing.Id, "node-1", 110, 10))

	reexported := ExportGenesis(importedCtx, &importedKeeper)
	require.Equal(t, exported, reexported)
}
