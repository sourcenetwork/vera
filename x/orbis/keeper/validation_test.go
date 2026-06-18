package keeper

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sourcenetwork/sourcehub/x/orbis/types"
)

func TestValidateRingDoesNotValidatePSSInterval(t *testing.T) {
	base := types.Ring{
		Id:           "ring-1",
		PeerNodeKeys: []string{"020000000000000000000000000000000000000000000000000000000000000000"},
		Threshold:    1,
		PssInterval:  types.MinPSSIntervalSeconds,
		PolicyId:     "policy-1",
	}
	require.NoError(t, validateRing(&base))

	for _, pssInterval := range []uint64{0, types.MinPSSIntervalSeconds - 1} {
		ring := base
		ring.PssInterval = pssInterval

		require.NoError(t, validateRing(&ring))
	}
}

func TestValidateRingPSSIntervalRejectsBelowMinimum(t *testing.T) {
	base := types.Ring{
		Id:           "ring-1",
		PeerNodeKeys: []string{"020000000000000000000000000000000000000000000000000000000000000000"},
		Threshold:    1,
		PssInterval:  types.MinPSSIntervalSeconds,
		PolicyId:     "policy-1",
	}
	require.NoError(t, validateRingPSSInterval(&base))

	for _, pssInterval := range []uint64{0, types.MinPSSIntervalSeconds - 1} {
		ring := base
		ring.PssInterval = pssInterval

		require.ErrorContains(t, validateRingPSSInterval(&ring), "pss_interval must be at least 86400 seconds")
	}
}
