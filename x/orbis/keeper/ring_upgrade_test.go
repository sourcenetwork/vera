package keeper

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	appparams "github.com/sourcenetwork/sourcehub/app/params"
	"github.com/sourcenetwork/sourcehub/x/orbis/types"
)

func TestRingUpgradeLifecycle(t *testing.T) {
	k, authKeeper, ctx := setupOrbisKeeper(t)
	const baseTime uint64 = 1_800_000_000
	ctx = ctx.
		WithValue(appparams.ExtractedDIDContextKey, testDID).
		WithBlockTime(time.Unix(int64(baseTime), 0))

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
		XActivationTime: &types.MsgUpdateRingByAcp_ActivationTime{
			ActivationTime: baseTime + 599,
		},
	})
	require.ErrorContains(t, err, "must be at least 1800000600")

	_, err = k.UpdateRingByAcp(ctx, &types.MsgUpdateRingByAcp{
		Creator: creatorAddr,
		RingId:  createVersionZero.RingId,
		XNextVersion: &types.MsgUpdateRingByAcp_NextVersion{
			NextVersion: 0,
		},
		XActivationTime: &types.MsgUpdateRingByAcp_ActivationTime{
			ActivationTime: baseTime + 600,
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
		XActivationTime: &types.MsgUpdateRingByAcp_ActivationTime{
			ActivationTime: baseTime + 600,
		},
	})
	require.ErrorContains(t, err, "cannot be combined")

	schedule := func(atTime uint64, nextVersion uint64, activationTime uint64) {
		t.Helper()
		updateCtx := ctx.WithBlockTime(time.Unix(int64(atTime), 0))
		_, updateErr := k.UpdateRingByAcp(updateCtx, &types.MsgUpdateRingByAcp{
			Creator: creatorAddr,
			RingId:  createVersionZero.RingId,
			XNextVersion: &types.MsgUpdateRingByAcp_NextVersion{
				NextVersion: nextVersion,
			},
			XActivationTime: &types.MsgUpdateRingByAcp_ActivationTime{
				ActivationTime: activationTime,
			},
		})
		require.NoError(t, updateErr)
	}

	schedule(baseTime, 1, baseTime+600)
	ring := k.GetRing(ctx, createVersionZero.RingId)
	require.Equal(t, uint64(0), ring.UpgradeInfo.CurrentVersion)
	require.Equal(t, uint64(1), ring.UpgradeInfo.GetNextVersion())
	require.Equal(t, baseTime+600, ring.UpgradeInfo.GetActivationTime())

	schedule(baseTime+1, 2, baseTime+601)
	ring = k.GetRing(ctx, createVersionZero.RingId)
	require.Equal(t, uint64(2), ring.UpgradeInfo.GetNextVersion())
	require.Equal(t, baseTime+601, ring.UpgradeInfo.GetActivationTime())

	_, err = k.UpdateRingByAcp(ctx.WithBlockTime(time.Unix(int64(baseTime+2), 0)), &types.MsgUpdateRingByAcp{
		Creator:      creatorAddr,
		RingId:       createVersionZero.RingId,
		ClearUpgrade: true,
	})
	require.NoError(t, err)
	ring = k.GetRing(ctx, createVersionZero.RingId)
	require.Nil(t, ring.UpgradeInfo.XNextVersion)
	require.Nil(t, ring.UpgradeInfo.XActivationTime)

	schedule(baseTime+2, 1, baseTime+602)
	_, err = k.UpdateRingByAcp(ctx.WithBlockTime(time.Unix(int64(baseTime+602), 0)), &types.MsgUpdateRingByAcp{
		Creator:      creatorAddr,
		RingId:       createVersionZero.RingId,
		ClearUpgrade: true,
	})
	require.NoError(t, err)
	ring = k.GetRing(ctx, createVersionZero.RingId)
	require.Equal(t, uint64(1), ring.UpgradeInfo.CurrentVersion)
	require.Nil(t, ring.UpgradeInfo.XNextVersion)
	require.Nil(t, ring.UpgradeInfo.XActivationTime)

	schedule(baseTime+602, 2, baseTime+1202)
	ring = k.GetRing(ctx, createVersionZero.RingId)
	require.Equal(t, uint64(1), ring.UpgradeInfo.CurrentVersion)
	require.Equal(t, uint64(2), ring.UpgradeInfo.GetNextVersion())
	require.Equal(t, baseTime+1202, ring.UpgradeInfo.GetActivationTime())

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
