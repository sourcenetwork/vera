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
					XActivationHeight: &UpgradeInfo_ActivationHeight{
						ActivationHeight: 500,
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
	require.Equal(t, int64(500), decoded.Rings[0].UpgradeInfo.GetActivationHeight())
}
