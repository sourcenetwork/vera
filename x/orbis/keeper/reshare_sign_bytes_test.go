package keeper

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sourcenetwork/sourcehub/x/orbis/types"
)

func TestRingReshareFinalizeSignBytesUseCanonicalOrbisSignState(t *testing.T) {
	nodeA := "02" + strings.Repeat("11", 32)
	nodeB := "02" + strings.Repeat("22", 32)
	nodeC := "02" + strings.Repeat("33", 32)
	nodeD := "02" + strings.Repeat("44", 32)

	current := &types.Ring{
		Id:              "ring-id",
		CreatorDid:      "did:example:one",
		RingPk:          "ring-pk",
		PeerNodeKeys:    []string{nodeB, nodeA},
		Threshold:       2,
		NewPeerNodeKeys: []string{nodeD, nodeC},
		XNewThreshold: &types.Ring_NewThreshold{
			NewThreshold: 1,
		},
		XPssInterval: &types.Ring_PssInterval{
			PssInterval: 30,
		},
		BlockNumberNonce: 9,
		PolicyId:         "policy-id",
		Confirmations: []*types.RingConfirmation{
			{NodeKey: nodeA, RingPk: "ring-pk"},
		},
	}
	finalized, err := ringForReshareFinalization(current)
	require.NoError(t, err)

	signBytes, err := ringReshareFinalizeSignBytes("sourcehub-test", current, finalized)
	require.NoError(t, err)

	sameSignState := &types.Ring{
		Id:              "ring-id",
		CreatorDid:      "did:example:two",
		RingPk:          "ring-pk",
		PeerNodeKeys:    []string{nodeA, nodeB},
		Threshold:       2,
		NewPeerNodeKeys: []string{nodeC, nodeD},
		XNewThreshold: &types.Ring_NewThreshold{
			NewThreshold: 1,
		},
		XPssInterval: &types.Ring_PssInterval{
			PssInterval: 30,
		},
		BlockNumberNonce: 9,
		PolicyId:         "policy-id",
		Confirmations: []*types.RingConfirmation{
			{NodeKey: nodeB, RingPk: "different-storage-only-value"},
		},
	}
	sameSignStateFinalized, err := ringForReshareFinalization(sameSignState)
	require.NoError(t, err)

	sameSignStateBytes, err := ringReshareFinalizeSignBytes(
		"sourcehub-test",
		sameSignState,
		sameSignStateFinalized,
	)
	require.NoError(t, err)

	require.Equal(t, signBytes, sameSignStateBytes)
}
