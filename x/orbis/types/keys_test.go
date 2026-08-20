package types

import (
	"testing"

	"github.com/sourcenetwork/immutable"
	"github.com/stretchr/testify/require"
)

func TestGenerateRingIDIsOrderIndependentWithoutMutatingInput(t *testing.T) {
	unsorted := []string{"node-b", "node-a"}
	sorted := []string{"node-a", "node-b"}

	unsortedID := GenerateRingID(unsorted, 1, MinPSSIntervalSeconds, "policy-id", immutable.None[string](), 1, false, nil)
	sortedID := GenerateRingID(sorted, 1, MinPSSIntervalSeconds, "policy-id", immutable.None[string](), 1, false, nil)

	require.Equal(t, sortedID, unsortedID)
	require.Equal(t, []string{"node-b", "node-a"}, unsorted)
}

func TestGenerateRingIDCanonicalizesTrustedAuthRelays(t *testing.T) {
	relays := []string{"did:key:relay-b", "did:key:relay-a"}
	reordered := []string{"did:key:relay-a", "did:key:relay-b"}

	first := GenerateRingID([]string{"node"}, 1, MinPSSIntervalSeconds, "policy-id", immutable.None[string](), 1, true, relays)
	second := GenerateRingID([]string{"node"}, 1, MinPSSIntervalSeconds, "policy-id", immutable.None[string](), 1, true, reordered)
	withoutRelays := GenerateRingID([]string{"node"}, 1, MinPSSIntervalSeconds, "policy-id", immutable.None[string](), 1, true, nil)
	directOnly := GenerateRingID([]string{"node"}, 1, MinPSSIntervalSeconds, "policy-id", immutable.None[string](), 1, false, nil)

	require.Equal(t, first, second)
	require.NotEqual(t, first, withoutRelays)
	require.NotEqual(t, withoutRelays, directOnly)
	require.Equal(t, []string{"did:key:relay-b", "did:key:relay-a"}, relays)
}
