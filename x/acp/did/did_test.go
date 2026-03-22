package did

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProduceDID(t *testing.T) {
	did, signer, err := ProduceDID()
	require.NoError(t, err)
	require.NotEmpty(t, did)
	require.NotNil(t, signer)
	require.Contains(t, did, "did:key:")
}

func TestProduceDIDUniqueness(t *testing.T) {
	did1, _, err := ProduceDID()
	require.NoError(t, err)
	did2, _, err := ProduceDID()
	require.NoError(t, err)
	require.NotEqual(t, did1, did2)
}
