package types

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/require"
)

func TestGenesisRingUpgradeInfoRoundTrips(t *testing.T) {
	genesis := &GenesisState{
		Rings: []Ring{
			{
				Id: "ring-1",
				UpgradeInfo: UpgradeInfo{
					CurrentVersion: 1,
					XNextVersion: &UpgradeInfo_NextVersion{
						NextVersion: 2,
					},
					XActivationTime: &UpgradeInfo_ActivationTime{
						ActivationTime: 500,
					},
				},
			},
		},
	}

	encoded, err := proto.Marshal(genesis)
	require.NoError(t, err)

	var decoded GenesisState
	require.NoError(t, proto.Unmarshal(encoded, &decoded))
	require.Len(t, decoded.Rings, 1)
	require.Equal(t, uint64(1), decoded.Rings[0].UpgradeInfo.CurrentVersion)
	require.Equal(t, uint64(2), decoded.Rings[0].UpgradeInfo.GetNextVersion())
	require.Equal(t, uint64(500), decoded.Rings[0].UpgradeInfo.GetActivationTime())
}

func TestGenesisValidateNodeDemerits(t *testing.T) {
	knownRing := Ring{Id: "ring-1"}

	valid := DefaultGenesis()
	valid.Rings = []Ring{knownRing}
	valid.NodeDemerits = []NodeDemeritEntry{
		{
			RingId:          "ring-1",
			NodeKey:         "node-1",
			Points:          3,
			WindowStartedAt: 100,
		},
	}
	require.NoError(t, valid.Validate())

	testCases := []struct {
		name     string
		entry    NodeDemeritEntry
		errMatch string
	}{
		{
			name:     "missing ring id",
			entry:    NodeDemeritEntry{NodeKey: "node-1", Points: 1},
			errMatch: "ring_id cannot be empty",
		},
		{
			name:     "missing node key",
			entry:    NodeDemeritEntry{RingId: "ring-1", Points: 1},
			errMatch: "node_key cannot be empty",
		},
		{
			name:     "zero points",
			entry:    NodeDemeritEntry{RingId: "ring-1", NodeKey: "node-1"},
			errMatch: "points must be at least 1",
		},
		{
			name:     "unknown ring",
			entry:    NodeDemeritEntry{RingId: "ring-999", NodeKey: "node-1", Points: 1},
			errMatch: "unknown ring",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			genesis := DefaultGenesis()
			genesis.Rings = []Ring{knownRing}
			genesis.NodeDemerits = []NodeDemeritEntry{tc.entry}
			require.ErrorContains(t, genesis.Validate(), tc.errMatch)
		})
	}

	duplicate := DefaultGenesis()
	duplicate.Rings = []Ring{knownRing}
	duplicate.NodeDemerits = []NodeDemeritEntry{
		{RingId: "ring-1", NodeKey: "node-1", Points: 1},
		{RingId: "ring-1", NodeKey: "node-1", Points: 2},
	}
	require.ErrorContains(t, duplicate.Validate(), "duplicate node_demerits entry")
}
