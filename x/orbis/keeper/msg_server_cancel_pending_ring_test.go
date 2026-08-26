package keeper

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/stretchr/testify/require"

	acptypes "github.com/sourcenetwork/vera/x/acp/types"
	"github.com/sourcenetwork/vera/x/orbis/types"
)

func TestMsgServer_CancelPendingRing_CreatorDeletesRingAndCanRetryWithNonce(t *testing.T) {
	k, authKeeper, ctx := setupOrbisKeeper(t)
	creatorCtx := ctxWithDID(ctx, testDID)

	creatorAddr, _ := testAccountWithPubKey(t, creatorCtx, authKeeper)
	_, peerKey := setupPeerWithNodeInfo(t, k, authKeeper, creatorCtx, "12D3KooWPeer1")
	policyID := createOrbisRingPolicy(t, k, creatorCtx, creatorAddr)

	createMsg := &types.MsgCreateRing{
		Creator:      creatorAddr,
		PeerNodeKeys: []string{peerKey},
		Threshold:    1,
		PssInterval:  types.MinPSSIntervalSeconds,
		PolicyId:     policyID,
	}
	createResp, err := k.CreateRing(creatorCtx, createMsg)
	require.NoError(t, err)

	cancelCtx := creatorCtx.WithEventManager(sdk.NewEventManager())
	_, err = k.CancelPendingRing(cancelCtx, &types.MsgCancelPendingRing{
		Creator: creatorAddr,
		RingId:  createResp.RingId,
	})
	require.NoError(t, err)
	require.Nil(t, k.GetRing(ctx, createResp.RingId))
	require.Equal(t, []proto.Message{
		&types.EventRingDeleted{
			RingId: createResp.RingId,
			Reason: "dkg_cancelled",
		},
	}, parseTypedEvents(t, cancelCtx))

	ownerResp, err := k.GetAcpKeeper().ObjectOwner(ctx, &acptypes.QueryObjectOwnerRequest{
		PolicyId: policyID,
		Object:   coretypes.NewObject(types.ACPResourceRing, createResp.RingId),
	})
	require.NoError(t, err)
	require.True(t, ownerResp.IsRegistered)
	require.NotNil(t, ownerResp.Record)

	_, err = k.CreateRing(creatorCtx, createMsg)
	require.Error(t, err)

	retryMsg := *createMsg
	retryMsg.XNonce = &types.MsgCreateRing_Nonce{Nonce: "attempt-2"}
	retryResp, err := k.CreateRing(creatorCtx, &retryMsg)
	require.NoError(t, err)
	require.NotEqual(t, createResp.RingId, retryResp.RingId)
	require.NotNil(t, k.GetRing(ctx, retryResp.RingId))
}

func TestMsgServer_CancelPendingRing_ParticipatingNodesCanDelete(t *testing.T) {
	tests := []struct {
		name             string
		cancellerIndex   int
		recordPartialDKG bool
	}{
		{name: "first node cancels pending ring", cancellerIndex: 0},
		{name: "second node cancels partially confirmed ring", cancellerIndex: 1, recordPartialDKG: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k, authKeeper, ctx := setupOrbisKeeper(t)
			creatorCtx := ctxWithDID(ctx, testDID)
			peerCtx := ctxWithDID(ctx, testPeerDID)

			creatorAddr, _ := testAccountWithPubKey(t, creatorCtx, authKeeper)
			peer1Addr, peer1Key := setupPeerWithNodeInfo(t, k, authKeeper, creatorCtx, "12D3KooWPeer1")
			peer2Addr, peer2Key := setupPeerWithNodeInfo(t, k, authKeeper, creatorCtx, "12D3KooWPeer2")
			policyID := createOrbisRingPolicy(t, k, creatorCtx, creatorAddr)

			createResp, err := k.CreateRing(creatorCtx, &types.MsgCreateRing{
				Creator:      creatorAddr,
				PeerNodeKeys: []string{peer1Key, peer2Key},
				Threshold:    2,
				PssInterval:  types.MinPSSIntervalSeconds,
				PolicyId:     policyID,
			})
			require.NoError(t, err)

			if tt.recordPartialDKG {
				finalizeResp, err := k.FinalizeRing(peerCtx, &types.MsgFinalizeRing{
					Creator: peer1Addr,
					RingId:  createResp.RingId,
					RingPk:  "ring-pk",
				})
				require.NoError(t, err)
				require.Equal(t, types.FinalizeRingOutcome_CONFIRMATION_RECORDED, finalizeResp.Outcome)
				require.Len(t, k.GetRing(ctx, createResp.RingId).Confirmations, 1)
			}

			cancellers := []string{peer1Addr, peer2Addr}
			_, err = k.CancelPendingRing(peerCtx, &types.MsgCancelPendingRing{
				Creator: cancellers[tt.cancellerIndex],
				RingId:  createResp.RingId,
			})
			require.NoError(t, err)
			require.Nil(t, k.GetRing(ctx, createResp.RingId))
		})
	}
}

func TestMsgServer_CancelPendingRing_RejectsOutsider(t *testing.T) {
	k, authKeeper, ctx := setupOrbisKeeper(t)
	creatorCtx := ctxWithDID(ctx, testDID)
	outsiderCtx := ctxWithDID(ctx, testOutsiderDID)

	creatorAddr, _ := testAccountWithPubKey(t, creatorCtx, authKeeper)
	_, peerKey := setupPeerWithNodeInfo(t, k, authKeeper, creatorCtx, "12D3KooWPeer1")
	outsiderAddr, _ := testAccountWithPubKey(t, outsiderCtx, authKeeper)
	policyID := createOrbisRingPolicy(t, k, creatorCtx, creatorAddr)

	createResp, err := k.CreateRing(creatorCtx, &types.MsgCreateRing{
		Creator:      creatorAddr,
		PeerNodeKeys: []string{peerKey},
		Threshold:    1,
		PssInterval:  types.MinPSSIntervalSeconds,
		PolicyId:     policyID,
	})
	require.NoError(t, err)

	_, err = k.CancelPendingRing(outsiderCtx, &types.MsgCancelPendingRing{
		Creator: outsiderAddr,
		RingId:  createResp.RingId,
	})
	require.ErrorIs(t, err, types.ErrInvalidRingCanceller)
	require.NotNil(t, k.GetRing(ctx, createResp.RingId))
}

func TestMsgServer_CancelPendingRing_RejectsFinalizedAndMissingRings(t *testing.T) {
	k, authKeeper, ctx := setupOrbisKeeper(t)
	creatorCtx := ctxWithDID(ctx, testDID)
	peerCtx := ctxWithDID(ctx, testPeerDID)

	creatorAddr, _ := testAccountWithPubKey(t, creatorCtx, authKeeper)
	peer1Addr, peer1Key := setupPeerWithNodeInfo(t, k, authKeeper, creatorCtx, "12D3KooWPeer1")
	peer2Addr, peer2Key := setupPeerWithNodeInfo(t, k, authKeeper, creatorCtx, "12D3KooWPeer2")
	policyID := createOrbisRingPolicy(t, k, creatorCtx, creatorAddr)

	createResp, err := k.CreateRing(creatorCtx, &types.MsgCreateRing{
		Creator:      creatorAddr,
		PeerNodeKeys: []string{peer1Key, peer2Key},
		Threshold:    2,
		PssInterval:  types.MinPSSIntervalSeconds,
		PolicyId:     policyID,
	})
	require.NoError(t, err)

	_, err = k.FinalizeRing(peerCtx, &types.MsgFinalizeRing{
		Creator: peer1Addr,
		RingId:  createResp.RingId,
		RingPk:  "ring-pk",
	})
	require.NoError(t, err)
	_, err = k.FinalizeRing(peerCtx, &types.MsgFinalizeRing{
		Creator: peer2Addr,
		RingId:  createResp.RingId,
		RingPk:  "ring-pk",
	})
	require.NoError(t, err)

	_, err = k.CancelPendingRing(creatorCtx, &types.MsgCancelPendingRing{
		Creator: creatorAddr,
		RingId:  createResp.RingId,
	})
	require.ErrorIs(t, err, types.ErrRingAlreadyFinalized)
	require.NotNil(t, k.GetRing(ctx, createResp.RingId))

	_, err = k.CancelPendingRing(creatorCtx, &types.MsgCancelPendingRing{
		Creator: creatorAddr,
		RingId:  "missing-ring",
	})
	require.ErrorIs(t, err, types.ErrRingNotFound)
}
