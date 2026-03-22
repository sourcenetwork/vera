package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultPolicyCommandMaxExpirationDelta(t *testing.T) {
	// 12 hours in seconds
	require.Equal(t, uint64(43200), uint64(DefaultPolicyCommandMaxExpirationDelta))
}

func TestDefaultRegistrationCommitmentLifetime(t *testing.T) {
	require.NotNil(t, DefaultRegistrationCommitmentLifetime)
	_, ok := DefaultRegistrationCommitmentLifetime.Duration.(*Duration_ProtoDuration)
	require.True(t, ok)
}
