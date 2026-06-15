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
		UpgradeInfo: types.UpgradeInfo{
			CurrentVersion: 1,
			XNextVersion: &types.UpgradeInfo_NextVersion{
				NextVersion: 2,
			},
			XActivationTime: &types.UpgradeInfo_ActivationTime{
				ActivationTime: 100,
			},
		},
		Confirmations: []*types.RingConfirmation{
			{NodeKey: nodeA, RingPk: "ring-pk"},
		},
	}
	finalized, err := ringForReshareFinalization(current)
	require.NoError(t, err)
	require.Equal(t, current.UpgradeInfo, finalized.UpgradeInfo)
	require.Equal(t, []string{nodeC, nodeD}, finalized.PeerNodeKeys)

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
			PssInterval: 60,
		},
		BlockNumberNonce: 9,
		PolicyId:         "policy-id",
		UpgradeInfo: types.UpgradeInfo{
			CurrentVersion: 7,
			XNextVersion: &types.UpgradeInfo_NextVersion{
				NextVersion: 8,
			},
			XActivationTime: &types.UpgradeInfo_ActivationTime{
				ActivationTime: 900,
			},
		},
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

	distinctSignState := *current
	distinctSignState.XNewThreshold = &types.Ring_NewThreshold{
		NewThreshold: 2,
	}
	distinctSignStateFinalized, err := ringForReshareFinalization(&distinctSignState)
	require.NoError(t, err)

	distinctSignStateBytes, err := ringReshareFinalizeSignBytes(
		"sourcehub-test",
		&distinctSignState,
		distinctSignStateFinalized,
	)
	require.NoError(t, err)

	require.NotEqual(t, signBytes, distinctSignStateBytes)
}
