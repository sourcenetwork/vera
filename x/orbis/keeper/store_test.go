package keeper

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sourcenetwork/vera/x/orbis/types"
)

func TestCanonicalStringsClonesSortedInput(t *testing.T) {
	nodeKeys := []string{"node-a", "node-b", "node-c"}

	canonical := canonicalStrings(nodeKeys)
	canonical[0] = "changed"

	require.Equal(t, []string{"node-a", "node-b", "node-c"}, nodeKeys)
}

func TestSetRingCanonicalizesSlicesWithoutMutatingInput(t *testing.T) {
	k, _, ctx := setupOrbisKeeper(t)

	activeCommittee := []string{"node-c", "node-a", "node-b"}
	pendingCommittee := []string{"node-f", "node-d", "node-e"}
	trustedAuthRelayDIDs := []string{testRelayDID, testSecondRelayDID}
	ring := types.Ring{
		Id:                     "ring-id",
		PeerNodeKeys:           activeCommittee,
		NewPeerNodeKeys:        pendingCommittee,
		AllowTrustedAuthRelays: true,
		TrustedAuthRelayDids:   trustedAuthRelayDIDs,
	}

	k.SetRing(ctx, ring)

	require.Equal(t, []string{"node-c", "node-a", "node-b"}, activeCommittee)
	require.Equal(t, []string{"node-f", "node-d", "node-e"}, pendingCommittee)
	require.Equal(t, []string{testRelayDID, testSecondRelayDID}, trustedAuthRelayDIDs)

	stored := k.GetRing(ctx, ring.Id)
	require.NotNil(t, stored)
	require.Equal(t, []string{"node-a", "node-b", "node-c"}, stored.PeerNodeKeys)
	require.Equal(t, []string{"node-d", "node-e", "node-f"}, stored.NewPeerNodeKeys)
	require.Equal(t, []string{testSecondRelayDID, testRelayDID}, stored.TrustedAuthRelayDids)
}

func TestDeleteExpiredAcceptedReportPairs(t *testing.T) {
	k, _, ctx := setupOrbisKeeper(t)

	const (
		reportA  = "report-a"
		reportB  = "report-b"
		sessionA = "session-a"
		sessionB = "session-b"
	)

	k.SetAcceptedReportPair(ctx, reportA, sessionA, 100)
	k.SetAcceptedReportPair(ctx, reportB, sessionB, 200)

	require.True(t, k.HasAcceptedReport(ctx, reportA))
	require.True(t, k.HasAcceptedReport(ctx, reportB))
	require.True(t, k.HasAcceptedReportSession(ctx, sessionA))
	require.True(t, k.HasAcceptedReportSession(ctx, sessionB))

	// at t=100 only entries with expiresAt <= 100 are deleted
	k.DeleteExpiredAcceptedReportPairs(ctx, 100)

	require.False(t, k.HasAcceptedReport(ctx, reportA))
	require.True(t, k.HasAcceptedReport(ctx, reportB))
	require.False(t, k.HasAcceptedReportSession(ctx, sessionA))
	require.True(t, k.HasAcceptedReportSession(ctx, sessionB))

	// at t=200 the remaining pair is also deleted
	k.DeleteExpiredAcceptedReportPairs(ctx, 200)

	require.False(t, k.HasAcceptedReport(ctx, reportB))
	require.False(t, k.HasAcceptedReportSession(ctx, sessionB))
}

func TestIncrementNodeDemeritsAppliesLazyResetWindow(t *testing.T) {
	k, _, ctx := setupOrbisKeeper(t)

	const (
		ringID        = "ring-id"
		otherRingID   = "other-ring-id"
		nodeKey       = "node-key"
		otherNodeKey  = "other-node-key"
		resetInterval = uint64(10)
	)

	require.Equal(t, uint64(0), k.GetEffectiveNodeDemerits(ctx, ringID, nodeKey, 100, resetInterval))

	require.Equal(t, uint64(2), k.IncrementNodeDemerits(ctx, ringID, nodeKey, 2, 100, resetInterval))
	state, found := k.getNodeDemeritState(ctx, ringID, nodeKey)
	require.True(t, found)
	require.Equal(t, nodeDemeritState{
		points:          2,
		windowStartedAt: 100,
	}, state)

	require.Equal(t, uint64(5), k.IncrementNodeDemerits(ctx, ringID, nodeKey, 3, 109, resetInterval))
	state, found = k.getNodeDemeritState(ctx, ringID, nodeKey)
	require.True(t, found)
	require.Equal(t, nodeDemeritState{
		points:          5,
		windowStartedAt: 100,
	}, state)
	require.Equal(t, uint64(5), k.GetEffectiveNodeDemerits(ctx, ringID, nodeKey, 109, resetInterval))
	require.Equal(t, uint64(0), k.GetEffectiveNodeDemerits(ctx, ringID, nodeKey, 110, resetInterval))
	state, found = k.getNodeDemeritState(ctx, ringID, nodeKey)
	require.True(t, found)
	require.Equal(t, nodeDemeritState{
		points:          5,
		windowStartedAt: 100,
	}, state)

	require.Equal(t, uint64(4), k.IncrementNodeDemerits(ctx, ringID, nodeKey, 4, 110, resetInterval))
	state, found = k.getNodeDemeritState(ctx, ringID, nodeKey)
	require.True(t, found)
	require.Equal(t, nodeDemeritState{
		points:          4,
		windowStartedAt: 110,
	}, state)

	require.Equal(t, uint64(7), k.IncrementNodeDemerits(ctx, ringID, otherNodeKey, 7, 105, resetInterval))
	require.Equal(t, uint64(8), k.IncrementNodeDemerits(ctx, otherRingID, nodeKey, 8, 105, resetInterval))

	require.Equal(t, uint64(4), k.GetNodeDemerits(ctx, ringID, nodeKey))
	require.Equal(t, uint64(7), k.GetNodeDemerits(ctx, ringID, otherNodeKey))
	require.Equal(t, uint64(8), k.GetNodeDemerits(ctx, otherRingID, nodeKey))

	otherNodeState, found := k.getNodeDemeritState(ctx, ringID, otherNodeKey)
	require.True(t, found)
	require.Equal(t, uint64(105), otherNodeState.windowStartedAt)

	otherRingState, found := k.getNodeDemeritState(ctx, otherRingID, nodeKey)
	require.True(t, found)
	require.Equal(t, uint64(105), otherRingState.windowStartedAt)
}
