package keeper

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sourcenetwork/sourcehub/x/orbis/types"
)

func TestValidateRingDoesNotValidatePSSInterval(t *testing.T) {
	base := types.Ring{
		Id:            "ring-1",
		PeerNodeKeys:  []string{"020000000000000000000000000000000000000000000000000000000000000000"},
		Threshold:     1,
		PssInterval:   types.MinPSSIntervalSeconds,
		PolicyId:      "policy-1",
		DemeritConfig: types.DefaultDemeritConfig(),
	}
	require.NoError(t, validateRing(&base))

	for _, pssInterval := range []uint64{0, types.MinPSSIntervalSeconds - 1} {
		ring := base
		ring.PssInterval = pssInterval

		require.NoError(t, validateRing(&ring))
	}
}

func TestValidateRingDemeritConfig(t *testing.T) {
	base := types.Ring{
		Id:            "ring-1",
		PeerNodeKeys:  []string{"020000000000000000000000000000000000000000000000000000000000000000"},
		Threshold:     1,
		PssInterval:   types.MinPSSIntervalSeconds,
		PolicyId:      "policy-1",
		DemeritConfig: types.DefaultDemeritConfig(),
	}
	require.NoError(t, validateRing(&base))

	cases := []struct {
		name        string
		config      types.DemeritConfig
		errContains string
	}{
		{
			name:        "zero NodeOfflineDemerits",
			config:      types.DemeritConfig{NodeOfflineDemerits: 0, ResetIntervalSeconds: types.DefaultDemeritResetIntervalSecs},
			errContains: "node_offline_demerits must be at least 1",
		},
		{
			name:        "zero ResetIntervalSeconds",
			config:      types.DemeritConfig{NodeOfflineDemerits: types.DefaultNodeOfflineDemerits, ResetIntervalSeconds: 0},
			errContains: "reset_interval_seconds must be at least 1",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ring := base
			ring.DemeritConfig = tc.config
			require.ErrorContains(t, validateRing(&ring), tc.errContains)
		})
	}
}

func TestValidateRingPSSIntervalRejectsBelowMinimum(t *testing.T) {
	base := types.Ring{
		Id:            "ring-1",
		PeerNodeKeys:  []string{"020000000000000000000000000000000000000000000000000000000000000000"},
		Threshold:     1,
		PssInterval:   types.MinPSSIntervalSeconds,
		PolicyId:      "policy-1",
		DemeritConfig: types.DefaultDemeritConfig(),
	}
	require.NoError(t, validateRingPSSInterval(&base))

	for _, pssInterval := range []uint64{0, types.MinPSSIntervalSeconds - 1} {
		ring := base
		ring.PssInterval = pssInterval

		require.ErrorContains(t, validateRingPSSInterval(&ring), "pss_interval must be at least 86400 seconds")
	}
}
