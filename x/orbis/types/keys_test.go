package types

import (
	"testing"

	"github.com/sourcenetwork/immutable"
	"github.com/stretchr/testify/require"
)

func TestGenerateRingIDIsOrderIndependentWithoutMutatingInput(t *testing.T) {
	unsorted := []string{"node-b", "node-a"}
	sorted := []string{"node-a", "node-b"}

	unsortedID := GenerateRingID(unsorted, 1, MinPSSIntervalSeconds, "policy-id", immutable.None[string](), 1)
	sortedID := GenerateRingID(sorted, 1, MinPSSIntervalSeconds, "policy-id", immutable.None[string](), 1)

	require.Equal(t, sortedID, unsortedID)
	require.Equal(t, []string{"node-b", "node-a"}, unsorted)
}
