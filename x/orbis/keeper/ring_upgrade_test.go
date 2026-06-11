package keeper

import (
	"strings"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"

	appparams "github.com/sourcenetwork/sourcehub/app/params"
	"github.com/sourcenetwork/sourcehub/x/orbis/types"
)

const ringUpgradeBaseTime uint64 = 1_800_000_000

type ringUpgradeFixture struct {
	k           Keeper
	ctx         sdk.Context
	creatorAddr string
	ringID      string
}

func setupFinalizedRingUpgradeFixture(t *testing.T, currentVersion uint64) ringUpgradeFixture {
	t.Helper()

	k, authKeeper, ctx := setupOrbisKeeper(t)
	ctx = ctx.
		WithValue(appparams.ExtractedDIDContextKey, testDID).
		WithBlockTime(time.Unix(int64(ringUpgradeBaseTime), 0))

	creatorAddr, _ := testAccountWithPubKey(t, ctx, authKeeper)
	peerAddr, peerKey := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWUpgradeFixturePeer")
	policyID := createOrbisRingPolicy(t, k, ctx, creatorAddr)

	createResp, err := k.CreateRing(ctx, &types.MsgCreateRing{
		Creator:        creatorAddr,
		PeerNodeKeys:   []string{peerKey},
		Threshold:      1,
		PolicyId:       policyID,
		CurrentVersion: currentVersion,
	})
	require.NoError(t, err)
	_, err = k.FinalizeRing(ctx, &types.MsgFinalizeRing{
		Creator: peerAddr,
		RingId:  createResp.RingId,
		RingPk:  "ring-pk",
	})
	require.NoError(t, err)

	return ringUpgradeFixture{
		k:           k,
		ctx:         ctx.WithEventManager(sdk.NewEventManager()),
		creatorAddr: creatorAddr,
		ringID:      createResp.RingId,
	}
}

func scheduleRingUpgrade(
	t *testing.T,
	fixture ringUpgradeFixture,
	currentTime uint64,
	nextVersion uint64,
	activationTime uint64,
) sdk.Context {
	t.Helper()

	updateCtx := fixture.ctx.
		WithBlockTime(time.Unix(int64(currentTime), 0)).
		WithEventManager(sdk.NewEventManager())
	_, err := fixture.k.UpdateRingByAcp(updateCtx, &types.MsgUpdateRingByAcp{
		Creator: fixture.creatorAddr,
		RingId:  fixture.ringID,
		XNextVersion: &types.MsgUpdateRingByAcp_NextVersion{
			NextVersion: nextVersion,
		},
		XActivationTime: &types.MsgUpdateRingByAcp_ActivationTime{
			ActivationTime: activationTime,
		},
	})
	require.NoError(t, err)
	return updateCtx
}

func parseTypedEvents(t *testing.T, ctx sdk.Context) []proto.Message {
	t.Helper()

	abciEvents := ctx.EventManager().Events().ToABCIEvents()
	events := make([]proto.Message, 0, len(abciEvents))
	for _, event := range abciEvents {
		typedEvent, err := sdk.ParseTypedEvent(event)
		require.NoError(t, err)
		events = append(events, typedEvent)
	}
	return events
}

func TestRingUpgradeLifecycle(t *testing.T) {
	k, authKeeper, ctx := setupOrbisKeeper(t)
	const baseTime = ringUpgradeBaseTime
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

func TestRingUpgradeRemainsPendingUntilAnUpdateNormalizesIt(t *testing.T) {
	fixture := setupFinalizedRingUpgradeFixture(t, 0)
	activationTime := ringUpgradeBaseTime + MinRingUpgradeLeadSeconds
	scheduleRingUpgrade(t, fixture, ringUpgradeBaseTime, 1, activationTime)

	queryCtx := fixture.ctx.
		WithBlockTime(time.Unix(int64(activationTime+1), 0)).
		WithEventManager(sdk.NewEventManager())
	response, err := fixture.k.Ring(queryCtx, &types.QueryRingRequest{Id: fixture.ringID})
	require.NoError(t, err)
	require.Equal(t, uint64(0), response.Ring.UpgradeInfo.CurrentVersion)
	require.Equal(t, uint64(1), response.Ring.UpgradeInfo.GetNextVersion())
	require.Equal(t, activationTime, response.Ring.UpgradeInfo.GetActivationTime())
	require.Empty(t, queryCtx.EventManager().Events())
}

func TestRingUpgradeActivationTimeRequiresNextVersion(t *testing.T) {
	fixture := setupFinalizedRingUpgradeFixture(t, 0)
	before := fixture.k.GetRing(fixture.ctx, fixture.ringID)

	_, err := fixture.k.UpdateRingByAcp(fixture.ctx, &types.MsgUpdateRingByAcp{
		Creator: fixture.creatorAddr,
		RingId:  fixture.ringID,
		XActivationTime: &types.MsgUpdateRingByAcp_ActivationTime{
			ActivationTime: ringUpgradeBaseTime + MinRingUpgradeLeadSeconds,
		},
	})
	require.ErrorContains(t, err, "must be supplied together")
	require.Equal(t, before, fixture.k.GetRing(fixture.ctx, fixture.ringID))
	require.Empty(t, fixture.ctx.EventManager().Events())
}

func TestRingUpgradeFailedUpdateDoesNotPersistNormalization(t *testing.T) {
	fixture := setupFinalizedRingUpgradeFixture(t, 0)
	activationTime := ringUpgradeBaseTime + MinRingUpgradeLeadSeconds
	scheduleRingUpgrade(t, fixture, ringUpgradeBaseTime, 1, activationTime)
	before := fixture.k.GetRing(fixture.ctx, fixture.ringID)

	updateCtx := fixture.ctx.
		WithBlockTime(time.Unix(int64(activationTime), 0)).
		WithEventManager(sdk.NewEventManager())
	_, err := fixture.k.UpdateRingByAcp(updateCtx, &types.MsgUpdateRingByAcp{
		Creator: fixture.creatorAddr,
		RingId:  fixture.ringID,
		XNextVersion: &types.MsgUpdateRingByAcp_NextVersion{
			NextVersion: 1,
		},
		XActivationTime: &types.MsgUpdateRingByAcp_ActivationTime{
			ActivationTime: activationTime + MinRingUpgradeLeadSeconds,
		},
	})
	require.ErrorContains(t, err, "must be greater than current_version")
	require.Equal(t, before, fixture.k.GetRing(updateCtx, fixture.ringID))
	require.Empty(t, updateCtx.EventManager().Events())
}

func TestRingUpgradeNormalizesAndReschedulesWithExactEvents(t *testing.T) {
	fixture := setupFinalizedRingUpgradeFixture(t, 0)
	firstActivationTime := ringUpgradeBaseTime + MinRingUpgradeLeadSeconds
	scheduleRingUpgrade(t, fixture, ringUpgradeBaseTime, 1, firstActivationTime)

	secondActivationTime := firstActivationTime + MinRingUpgradeLeadSeconds
	updateCtx := scheduleRingUpgrade(t, fixture, firstActivationTime, 2, secondActivationTime)

	ring := fixture.k.GetRing(updateCtx, fixture.ringID)
	require.Equal(t, uint64(1), ring.UpgradeInfo.CurrentVersion)
	require.Equal(t, uint64(2), ring.UpgradeInfo.GetNextVersion())
	require.Equal(t, secondActivationTime, ring.UpgradeInfo.GetActivationTime())

	events := parseTypedEvents(t, updateCtx)
	require.Equal(t, []proto.Message{
		&types.EventRingUpdated{
			RingId:     fixture.ringID,
			UpdaterDid: testDID,
		},
		&types.EventRingUpgradeNormalized{
			RingId:          fixture.ringID,
			PreviousVersion: 0,
			CurrentVersion:  1,
			ActivationTime:  firstActivationTime,
		},
		&types.EventRingUpgradeScheduled{
			RingId:         fixture.ringID,
			CurrentVersion: 1,
			NextVersion:    2,
			ActivationTime: secondActivationTime,
		},
	}, events)
}

func TestRingUpgradeCancellationEmitsExactEvent(t *testing.T) {
	fixture := setupFinalizedRingUpgradeFixture(t, 3)
	activationTime := ringUpgradeBaseTime + MinRingUpgradeLeadSeconds
	scheduleRingUpgrade(t, fixture, ringUpgradeBaseTime, 4, activationTime)

	updateCtx := fixture.ctx.
		WithBlockTime(time.Unix(int64(ringUpgradeBaseTime+1), 0)).
		WithEventManager(sdk.NewEventManager())
	_, err := fixture.k.UpdateRingByAcp(updateCtx, &types.MsgUpdateRingByAcp{
		Creator:      fixture.creatorAddr,
		RingId:       fixture.ringID,
		ClearUpgrade: true,
	})
	require.NoError(t, err)

	events := parseTypedEvents(t, updateCtx)
	require.Equal(t, []proto.Message{
		&types.EventRingUpdated{
			RingId:     fixture.ringID,
			UpdaterDid: testDID,
		},
		&types.EventRingUpgradeCancelled{
			RingId:         fixture.ringID,
			CurrentVersion: 3,
		},
	}, events)
}

func TestRingUpgradeClearWithoutPendingUpgradeOnlyEmitsRingUpdated(t *testing.T) {
	fixture := setupFinalizedRingUpgradeFixture(t, 3)

	_, err := fixture.k.UpdateRingByAcp(fixture.ctx, &types.MsgUpdateRingByAcp{
		Creator:      fixture.creatorAddr,
		RingId:       fixture.ringID,
		ClearUpgrade: true,
	})
	require.NoError(t, err)

	ring := fixture.k.GetRing(fixture.ctx, fixture.ringID)
	require.Equal(t, uint64(3), ring.UpgradeInfo.CurrentVersion)
	require.Nil(t, ring.UpgradeInfo.XNextVersion)
	require.Nil(t, ring.UpgradeInfo.XActivationTime)
	require.Equal(t, []proto.Message{
		&types.EventRingUpdated{
			RingId:     fixture.ringID,
			UpdaterDid: testDID,
		},
	}, parseTypedEvents(t, fixture.ctx))
}
