package keeper

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	appparams "github.com/sourcenetwork/sourcehub/app/params"
	"github.com/sourcenetwork/sourcehub/x/orbis/types"
)

func TestRingUpgradeLifecycle(t *testing.T) {
	k, authKeeper, ctx := setupOrbisKeeper(t)
	ctx = ctx.WithValue(appparams.ExtractedDIDContextKey, testDID).WithBlockHeight(10)

	creatorAddr, _ := testAccountWithPubKey(t, ctx, authKeeper)
	peerAddr, peerKey := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWUpgradePeer")
	policyID := createOrbisRingPolicy(t, k, ctx, creatorAddr)

	createVersionZero, err := k.CreateRing(ctx, &types.MsgCreateRing{
		Creator:        creatorAddr,
		PeerNodeKeys:   []string{peerKey},
		Threshold:      1,
		PolicyId:       policyID,
		CurrentVersion: 0,
	})
	require.NoError(t, err)

	createVersionOne, err := k.CreateRing(ctx, &types.MsgCreateRing{
		Creator:        creatorAddr,
		PeerNodeKeys:   []string{peerKey},
		Threshold:      1,
		PolicyId:       policyID,
		CurrentVersion: 1,
	})
	require.NoError(t, err)
	require.NotEqual(t, createVersionZero.RingId, createVersionOne.RingId)
	require.Equal(t, uint64(1), k.GetRing(ctx, createVersionOne.RingId).UpgradeInfo.CurrentVersion)

	_, err = k.FinalizeRing(ctx, &types.MsgFinalizeRing{
		Creator: peerAddr,
		RingId:  createVersionZero.RingId,
		RingPk:  "ring-pk",
	})
	require.NoError(t, err)
	require.Equal(t, uint64(0), k.GetRing(ctx, createVersionZero.RingId).UpgradeInfo.CurrentVersion)

	_, err = k.UpdateRingByAcp(ctx, &types.MsgUpdateRingByAcp{
		Creator: creatorAddr,
		RingId:  createVersionZero.RingId,
		XNextVersion: &types.MsgUpdateRingByAcp_NextVersion{
			NextVersion: 1,
		},
	})
	require.ErrorContains(t, err, "must be supplied together")

	_, err = k.UpdateRingByAcp(ctx, &types.MsgUpdateRingByAcp{
		Creator: creatorAddr,
		RingId:  createVersionZero.RingId,
		XNextVersion: &types.MsgUpdateRingByAcp_NextVersion{
			NextVersion: 1,
		},
		XActivationHeight: &types.MsgUpdateRingByAcp_ActivationHeight{
			ActivationHeight: 109,
		},
	})
	require.ErrorContains(t, err, "must be at least 110")

	_, err = k.UpdateRingByAcp(ctx, &types.MsgUpdateRingByAcp{
		Creator: creatorAddr,
		RingId:  createVersionZero.RingId,
		XNextVersion: &types.MsgUpdateRingByAcp_NextVersion{
			NextVersion: 0,
		},
		XActivationHeight: &types.MsgUpdateRingByAcp_ActivationHeight{
			ActivationHeight: 110,
		},
	})
	require.ErrorContains(t, err, "must be greater than current_version")

	_, err = k.UpdateRingByAcp(ctx, &types.MsgUpdateRingByAcp{
		Creator:      creatorAddr,
		RingId:       createVersionZero.RingId,
		ClearUpgrade: true,
		XNextVersion: &types.MsgUpdateRingByAcp_NextVersion{
			NextVersion: 1,
		},
		XActivationHeight: &types.MsgUpdateRingByAcp_ActivationHeight{
			ActivationHeight: 110,
		},
	})
	require.ErrorContains(t, err, "cannot be combined")

	schedule := func(atHeight int64, nextVersion uint64, activationHeight int64) {
		t.Helper()
		updateCtx := ctx.WithBlockHeight(atHeight)
		_, updateErr := k.UpdateRingByAcp(updateCtx, &types.MsgUpdateRingByAcp{
			Creator: creatorAddr,
			RingId:  createVersionZero.RingId,
			XNextVersion: &types.MsgUpdateRingByAcp_NextVersion{
				NextVersion: nextVersion,
			},
			XActivationHeight: &types.MsgUpdateRingByAcp_ActivationHeight{
				ActivationHeight: activationHeight,
			},
		})
		require.NoError(t, updateErr)
	}

	schedule(10, 1, 110)
	ring := k.GetRing(ctx, createVersionZero.RingId)
	require.Equal(t, uint64(0), ring.UpgradeInfo.CurrentVersion)
	require.Equal(t, uint64(1), ring.UpgradeInfo.GetNextVersion())
	require.Equal(t, int64(110), ring.UpgradeInfo.GetActivationHeight())

	schedule(11, 2, 111)
	ring = k.GetRing(ctx, createVersionZero.RingId)
	require.Equal(t, uint64(2), ring.UpgradeInfo.GetNextVersion())
	require.Equal(t, int64(111), ring.UpgradeInfo.GetActivationHeight())

	_, err = k.UpdateRingByAcp(ctx.WithBlockHeight(12), &types.MsgUpdateRingByAcp{
		Creator:      creatorAddr,
		RingId:       createVersionZero.RingId,
		ClearUpgrade: true,
	})
	require.NoError(t, err)
	ring = k.GetRing(ctx, createVersionZero.RingId)
	require.Nil(t, ring.UpgradeInfo.XNextVersion)
	require.Nil(t, ring.UpgradeInfo.XActivationHeight)

	schedule(12, 1, 112)
	_, err = k.UpdateRingByAcp(ctx.WithBlockHeight(112), &types.MsgUpdateRingByAcp{
		Creator:      creatorAddr,
		RingId:       createVersionZero.RingId,
		ClearUpgrade: true,
	})
	require.NoError(t, err)
	ring = k.GetRing(ctx, createVersionZero.RingId)
	require.Equal(t, uint64(1), ring.UpgradeInfo.CurrentVersion)
	require.Nil(t, ring.UpgradeInfo.XNextVersion)
	require.Nil(t, ring.UpgradeInfo.XActivationHeight)

	schedule(112, 2, 212)
	ring = k.GetRing(ctx, createVersionZero.RingId)
	require.Equal(t, uint64(1), ring.UpgradeInfo.CurrentVersion)
	require.Equal(t, uint64(2), ring.UpgradeInfo.GetNextVersion())
	require.Equal(t, int64(212), ring.UpgradeInfo.GetActivationHeight())

	seenEvent := func(suffix string) bool {
		for _, event := range ctx.EventManager().Events() {
			if strings.HasSuffix(event.Type, suffix) {
				return true
			}
		}
		return false
	}
	require.True(t, seenEvent("EventRingUpgradeScheduled"))
	require.True(t, seenEvent("EventRingUpgradeCancelled"))
	require.True(t, seenEvent("EventRingUpgradeNormalized"))
}
