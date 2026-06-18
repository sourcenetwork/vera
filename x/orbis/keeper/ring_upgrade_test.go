package keeper

import (
	"fmt"
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
	peer1Addr, peer1Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWUpgradeFixturePeer1")
	peer2Addr, peer2Key := setupPeerWithNodeInfo(t, k, authKeeper, ctx, "12D3KooWUpgradeFixturePeer2")
	policyID := createOrbisRingPolicy(t, k, ctx, creatorAddr)

	createResp, err := k.CreateRing(ctx, &types.MsgCreateRing{
		Creator:        creatorAddr,
		PeerNodeKeys:   []string{peer1Key, peer2Key},
		Threshold:      1,
		PssInterval:    types.MinPSSIntervalSeconds,
		PolicyId:       policyID,
		CurrentVersion: currentVersion,
	})
	require.NoError(t, err)
	_, err = k.FinalizeRing(ctx, &types.MsgFinalizeRing{
		Creator: peer1Addr,
		RingId:  createResp.RingId,
		RingPk:  "ring-pk",
	})
	require.NoError(t, err)
	_, err = k.FinalizeRing(ctx, &types.MsgFinalizeRing{
		Creator: peer2Addr,
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
	_, err := fixture.k.ScheduleRingUpgradeByAcp(updateCtx, &types.MsgScheduleRingUpgradeByAcp{
		Creator:        fixture.creatorAddr,
		RingId:         fixture.ringID,
		NextVersion:    nextVersion,
		ActivationTime: activationTime,
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
		PssInterval:    types.MinPSSIntervalSeconds,
		PolicyId:       policyID,
		CurrentVersion: 0,
	})
	require.NoError(t, err)

	createVersionOne, err := k.CreateRing(ctx, &types.MsgCreateRing{
		Creator:        creatorAddr,
		PeerNodeKeys:   []string{peerKey},
		Threshold:      1,
		PssInterval:    types.MinPSSIntervalSeconds,
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

	_, err = k.ScheduleRingUpgradeByAcp(ctx, &types.MsgScheduleRingUpgradeByAcp{
		Creator:        creatorAddr,
		RingId:         createVersionZero.RingId,
		NextVersion:    1,
		ActivationTime: baseTime + 599,
	})
	require.ErrorContains(
		t,
		err,
		fmt.Sprintf("must be at least %d", baseTime+MinRingUpgradeLeadSeconds),
	)

	_, err = k.ScheduleRingUpgradeByAcp(ctx, &types.MsgScheduleRingUpgradeByAcp{
		Creator:        creatorAddr,
		RingId:         createVersionZero.RingId,
		NextVersion:    0,
		ActivationTime: baseTime + 600,
	})
	require.ErrorContains(t, err, "must be greater than current_version")

	schedule := func(atTime uint64, nextVersion uint64, activationTime uint64) {
		t.Helper()
		updateCtx := ctx.WithBlockTime(time.Unix(int64(atTime), 0))
		_, updateErr := k.ScheduleRingUpgradeByAcp(updateCtx, &types.MsgScheduleRingUpgradeByAcp{
			Creator:        creatorAddr,
			RingId:         createVersionZero.RingId,
			NextVersion:    nextVersion,
			ActivationTime: activationTime,
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

	_, err = k.CancelRingUpgradeByAcp(ctx.WithBlockTime(time.Unix(int64(baseTime+2), 0)), &types.MsgCancelRingUpgradeByAcp{
		Creator: creatorAddr,
		RingId:  createVersionZero.RingId,
	})
	require.NoError(t, err)
	ring = k.GetRing(ctx, createVersionZero.RingId)
	require.Nil(t, ring.UpgradeInfo.XNextVersion)
	require.Nil(t, ring.UpgradeInfo.XActivationTime)

	schedule(baseTime+2, 1, baseTime+602)
	activationCtx := ctx.WithBlockTime(time.Unix(int64(baseTime+602), 0))
	_, err = k.CancelRingUpgradeByAcp(activationCtx, &types.MsgCancelRingUpgradeByAcp{
		Creator: creatorAddr,
		RingId:  createVersionZero.RingId,
	})
	require.ErrorIs(t, err, types.ErrRingUpgradeAlreadyActive)
	_, err = k.SetRingPssIntervalByAcp(activationCtx, &types.MsgSetRingPssIntervalByAcp{
		Creator:     creatorAddr,
		RingId:      createVersionZero.RingId,
		PssInterval: types.MinPSSIntervalSeconds + 1,
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

func TestScheduleRingUpgradeAllowsStoredPSSIntervalBelowMinimum(t *testing.T) {
	fixture := setupFinalizedRingUpgradeFixture(t, 0)
	ring := fixture.k.GetRing(fixture.ctx, fixture.ringID)
	require.NotNil(t, ring)
	ring.PssInterval = types.MinPSSIntervalSeconds - 1
	fixture.k.SetRing(fixture.ctx, *ring)

	activationTime := ringUpgradeBaseTime + MinRingUpgradeLeadSeconds
	_, err := fixture.k.ScheduleRingUpgradeByAcp(fixture.ctx, &types.MsgScheduleRingUpgradeByAcp{
		Creator:        fixture.creatorAddr,
		RingId:         fixture.ringID,
		NextVersion:    1,
		ActivationTime: activationTime,
	})
	require.NoError(t, err)

	stored := fixture.k.GetRing(fixture.ctx, fixture.ringID)
	require.Equal(t, types.MinPSSIntervalSeconds-1, stored.GetPssInterval())
	require.Equal(t, uint64(1), stored.UpgradeInfo.GetNextVersion())
	require.Equal(t, activationTime, stored.UpgradeInfo.GetActivationTime())
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

func TestRingUpgradeScheduleValidateBasicRejectsZeroFields(t *testing.T) {
	fixture := setupFinalizedRingUpgradeFixture(t, 0)

	err := (&types.MsgScheduleRingUpgradeByAcp{
		Creator:        fixture.creatorAddr,
		RingId:         fixture.ringID,
		ActivationTime: ringUpgradeBaseTime + MinRingUpgradeLeadSeconds,
	}).ValidateBasic()
	require.ErrorContains(t, err, "next_version must be positive")

	err = (&types.MsgScheduleRingUpgradeByAcp{
		Creator:     fixture.creatorAddr,
		RingId:      fixture.ringID,
		NextVersion: 1,
	}).ValidateBasic()
	require.ErrorContains(t, err, "activation_time must be positive")
}

func TestRingUpgradeFailedUpdateDoesNotPersistNormalization(t *testing.T) {
	fixture := setupFinalizedRingUpgradeFixture(t, 0)
	activationTime := ringUpgradeBaseTime + MinRingUpgradeLeadSeconds
	scheduleRingUpgrade(t, fixture, ringUpgradeBaseTime, 1, activationTime)
	before := fixture.k.GetRing(fixture.ctx, fixture.ringID)

	updateCtx := fixture.ctx.
		WithBlockTime(time.Unix(int64(activationTime), 0)).
		WithEventManager(sdk.NewEventManager())
	_, err := fixture.k.ScheduleRingUpgradeByAcp(updateCtx, &types.MsgScheduleRingUpgradeByAcp{
		Creator:        fixture.creatorAddr,
		RingId:         fixture.ringID,
		NextVersion:    1,
		ActivationTime: activationTime + MinRingUpgradeLeadSeconds,
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

func TestRingUpgradeNormalizesWhenStartingReshare(t *testing.T) {
	fixture := setupFinalizedRingUpgradeFixture(t, 0)
	activationTime := ringUpgradeBaseTime + MinRingUpgradeLeadSeconds
	scheduleRingUpgrade(t, fixture, ringUpgradeBaseTime, 1, activationTime)

	updateCtx := fixture.ctx.
		WithBlockTime(time.Unix(int64(activationTime), 0)).
		WithEventManager(sdk.NewEventManager())
	_, err := fixture.k.StartRingReshareByAcp(updateCtx, &types.MsgStartRingReshareByAcp{
		Creator: fixture.creatorAddr,
		RingId:  fixture.ringID,
		XNewThreshold: &types.MsgStartRingReshareByAcp_NewThreshold{
			NewThreshold: 2,
		},
	})
	require.NoError(t, err)

	ring := fixture.k.GetRing(updateCtx, fixture.ringID)
	require.Equal(t, uint64(1), ring.UpgradeInfo.CurrentVersion)
	require.Nil(t, ring.UpgradeInfo.XNextVersion)
	require.Equal(t, uint32(2), ring.GetNewThreshold())
	require.IsType(t, &types.EventRingUpgradeNormalized{}, parseTypedEvents(t, updateCtx)[1])
}

func TestRingUpgradeNormalizesWhenSettingPssInterval(t *testing.T) {
	fixture := setupFinalizedRingUpgradeFixture(t, 0)
	activationTime := ringUpgradeBaseTime + MinRingUpgradeLeadSeconds
	scheduleRingUpgrade(t, fixture, ringUpgradeBaseTime, 1, activationTime)

	updateCtx := fixture.ctx.
		WithBlockTime(time.Unix(int64(activationTime), 0)).
		WithEventManager(sdk.NewEventManager())
	_, err := fixture.k.SetRingPssIntervalByAcp(updateCtx, &types.MsgSetRingPssIntervalByAcp{
		Creator:     fixture.creatorAddr,
		RingId:      fixture.ringID,
		PssInterval: types.MinPSSIntervalSeconds + 1,
	})
	require.NoError(t, err)

	ring := fixture.k.GetRing(updateCtx, fixture.ringID)
	require.Equal(t, uint64(1), ring.UpgradeInfo.CurrentVersion)
	require.Nil(t, ring.UpgradeInfo.XNextVersion)
	require.Equal(t, types.MinPSSIntervalSeconds+1, ring.GetPssInterval())
	require.IsType(t, &types.EventRingUpgradeNormalized{}, parseTypedEvents(t, updateCtx)[1])
}

func TestRingUpgradeRejectsUnchangedSchedule(t *testing.T) {
	fixture := setupFinalizedRingUpgradeFixture(t, 0)
	activationTime := ringUpgradeBaseTime + MinRingUpgradeLeadSeconds
	scheduleRingUpgrade(t, fixture, ringUpgradeBaseTime, 1, activationTime)
	before := fixture.k.GetRing(fixture.ctx, fixture.ringID)

	_, err := fixture.k.ScheduleRingUpgradeByAcp(fixture.ctx, &types.MsgScheduleRingUpgradeByAcp{
		Creator:        fixture.creatorAddr,
		RingId:         fixture.ringID,
		NextVersion:    1,
		ActivationTime: activationTime,
	})
	require.ErrorIs(t, err, types.ErrRingUpgradeUnchanged)
	require.Equal(t, before, fixture.k.GetRing(fixture.ctx, fixture.ringID))
}

func TestRingUpgradeCancellationEmitsExactEvent(t *testing.T) {
	fixture := setupFinalizedRingUpgradeFixture(t, 3)
	activationTime := ringUpgradeBaseTime + MinRingUpgradeLeadSeconds
	scheduleRingUpgrade(t, fixture, ringUpgradeBaseTime, 4, activationTime)

	updateCtx := fixture.ctx.
		WithBlockTime(time.Unix(int64(ringUpgradeBaseTime+1), 0)).
		WithEventManager(sdk.NewEventManager())
	_, err := fixture.k.CancelRingUpgradeByAcp(updateCtx, &types.MsgCancelRingUpgradeByAcp{
		Creator: fixture.creatorAddr,
		RingId:  fixture.ringID,
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

func TestRingUpgradeCancelWithoutPendingUpgradeFails(t *testing.T) {
	fixture := setupFinalizedRingUpgradeFixture(t, 3)

	_, err := fixture.k.CancelRingUpgradeByAcp(fixture.ctx, &types.MsgCancelRingUpgradeByAcp{
		Creator: fixture.creatorAddr,
		RingId:  fixture.ringID,
	})
	require.ErrorIs(t, err, types.ErrRingUpgradeNotScheduled)

	ring := fixture.k.GetRing(fixture.ctx, fixture.ringID)
	require.Equal(t, uint64(3), ring.UpgradeInfo.CurrentVersion)
	require.Nil(t, ring.UpgradeInfo.XNextVersion)
	require.Nil(t, ring.UpgradeInfo.XActivationTime)
	require.Empty(t, parseTypedEvents(t, fixture.ctx))
}
