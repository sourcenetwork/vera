package keeper

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sourcenetwork/sourcehub/x/orbis/types"
)

func TestCanonicalNodeKeysClonesSortedInput(t *testing.T) {
	nodeKeys := []string{"node-a", "node-b", "node-c"}

	canonical := canonicalNodeKeys(nodeKeys)
	canonical[0] = "changed"

	require.Equal(t, []string{"node-a", "node-b", "node-c"}, nodeKeys)
}

func TestSetRingCanonicalizesCommitteesWithoutMutatingInput(t *testing.T) {
	k, _, ctx := setupOrbisKeeper(t)

	activeCommittee := []string{"node-c", "node-a", "node-b"}
	pendingCommittee := []string{"node-f", "node-d", "node-e"}
	ring := types.Ring{
		Id:              "ring-id",
		PeerNodeKeys:    activeCommittee,
		NewPeerNodeKeys: pendingCommittee,
	}

	k.SetRing(ctx, ring)

	require.Equal(t, []string{"node-c", "node-a", "node-b"}, activeCommittee)
	require.Equal(t, []string{"node-f", "node-d", "node-e"}, pendingCommittee)

	stored := k.GetRing(ctx, ring.Id)
	require.NotNil(t, stored)
	require.Equal(t, []string{"node-a", "node-b", "node-c"}, stored.PeerNodeKeys)
	require.Equal(t, []string{"node-d", "node-e", "node-f"}, stored.NewPeerNodeKeys)
}
