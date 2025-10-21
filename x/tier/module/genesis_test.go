package tier_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sourcenetwork/sourcehub/app"
	keepertest "github.com/sourcenetwork/sourcehub/testutil/keeper"
	"github.com/sourcenetwork/sourcehub/testutil/nullify"
	tier "github.com/sourcenetwork/sourcehub/x/tier/module"
	"github.com/sourcenetwork/sourcehub/x/tier/types"
	"github.com/stretchr/testify/require"
)

func init() {
	app.SetConfig(true)
}

func TestGenesis(t *testing.T) {
	timestamp1 := time.Date(2006, time.January, 2, 15, 4, 5, 1, time.UTC)
	timestamp2 := time.Date(2006, time.January, 2, 15, 4, 5, 2, time.UTC)
	timestamp3 := time.Date(2006, time.January, 2, 15, 4, 5, 3, time.UTC)
	timestamp4 := time.Date(2006, time.January, 2, 15, 4, 5, 4, time.UTC)
	timestamp5 := time.Date(2006, time.January, 2, 15, 4, 5, 5, time.UTC)
	timestamp6 := time.Date(2006, time.January, 2, 15, 4, 5, 6, time.UTC)

	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
		Lockups: []types.Lockup{
			{
				DelegatorAddress: "source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et",
				ValidatorAddress: "sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm",
				Amount:           math.NewInt(1000),
			},
			{
				DelegatorAddress: "source1n34fvpteuanu2nx2a4hql4jvcrcnal3gsrjppy",
				ValidatorAddress: "sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm",
				Amount:           math.NewInt(500),
			},
			{
				DelegatorAddress: "source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9",
				ValidatorAddress: "sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm",
				Amount:           math.NewInt(2000),
			},
		},
		UnlockingLockups: []types.UnlockingLockup{
			{
				DelegatorAddress: "source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et",
				ValidatorAddress: "sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm",
				Amount:           math.NewInt(1000),
				CreationHeight:   1,
				CompletionTime:   timestamp1,
				UnlockTime:       timestamp4,
			},
			{
				DelegatorAddress: "source1n34fvpteuanu2nx2a4hql4jvcrcnal3gsrjppy",
				ValidatorAddress: "sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm",
				Amount:           math.NewInt(500),
				CreationHeight:   2,
				CompletionTime:   timestamp2,
				UnlockTime:       timestamp5,
			},
			{
				DelegatorAddress: "source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9",
				ValidatorAddress: "sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm",
				Amount:           math.NewInt(2000),
				CreationHeight:   3,
				CompletionTime:   timestamp3,
				UnlockTime:       timestamp6,
			},
		},
		Developers: []types.Developer{
			{
				Address:         "source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et",
				AutoLockEnabled: true,
			},
			{
				Address:         "source1n34fvpteuanu2nx2a4hql4jvcrcnal3gsrjppy",
				AutoLockEnabled: false,
			},
			{
				Address:         "source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9",
				AutoLockEnabled: true,
			},
		},
		UserSubscriptions: []types.UserSubscription{
			{
				Developer:    "source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et",
				UserDid:      "did:key:alice",
				CreditAmount: sdk.NewCoin("ucredit", math.NewInt(1000)),
				Period:       30,
				StartDate:    timestamp1,
				LastRenewed:  timestamp2,
			},
			{
				Developer:    "source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et",
				UserDid:      "did:key:bob",
				CreditAmount: sdk.NewCoin("ucredit", math.NewInt(500)),
				Period:       60,
				StartDate:    timestamp3,
				LastRenewed:  timestamp4,
			},
			{
				Developer:    "source1n34fvpteuanu2nx2a4hql4jvcrcnal3gsrjppy",
				UserDid:      "did:key:charlie",
				CreditAmount: sdk.NewCoin("ucredit", math.NewInt(2000)),
				Period:       90,
				StartDate:    timestamp5,
				LastRenewed:  timestamp6,
			},
		},
	}

	k, ctx := keepertest.TierKeeper(t)
	tier.InitGenesis(ctx, &k, genesisState)
	got := tier.ExportGenesis(ctx, &k)
	require.NotNil(t, got)
	require.Equal(t, genesisState.Params, got.Params)
	require.Equal(t, len(genesisState.Lockups), len(got.Lockups))

	for i, lockup := range genesisState.Lockups {
		require.Equal(t, lockup.DelegatorAddress, got.Lockups[i].DelegatorAddress)
		require.Equal(t, lockup.ValidatorAddress, got.Lockups[i].ValidatorAddress)
		require.Equal(t, lockup.Amount, got.Lockups[i].Amount)
	}

	for i, unlockingLockup := range genesisState.UnlockingLockups {
		require.Equal(t, unlockingLockup.DelegatorAddress, got.UnlockingLockups[i].DelegatorAddress)
		require.Equal(t, unlockingLockup.ValidatorAddress, got.UnlockingLockups[i].ValidatorAddress)
		require.Equal(t, unlockingLockup.Amount, got.UnlockingLockups[i].Amount)
		require.Equal(t, unlockingLockup.CreationHeight, got.UnlockingLockups[i].CreationHeight)
		require.Equal(t, unlockingLockup.CompletionTime.UTC(), got.UnlockingLockups[i].CompletionTime.UTC())
		require.Equal(t, unlockingLockup.UnlockTime.UTC(), got.UnlockingLockups[i].UnlockTime.UTC())
	}

	require.Equal(t, len(genesisState.Developers), len(got.Developers))
	for i, developer := range genesisState.Developers {
		require.Equal(t, developer.Address, got.Developers[i].Address)
		require.Equal(t, developer.AutoLockEnabled, got.Developers[i].AutoLockEnabled)
	}

	require.Equal(t, len(genesisState.UserSubscriptions), len(got.UserSubscriptions))
	for i, userSubscription := range genesisState.UserSubscriptions {
		require.Equal(t, userSubscription.Developer, got.UserSubscriptions[i].Developer)
		require.Equal(t, userSubscription.UserDid, got.UserSubscriptions[i].UserDid)
		require.Equal(t, userSubscription.CreditAmount, got.UserSubscriptions[i].CreditAmount)
		require.Equal(t, userSubscription.Period, got.UserSubscriptions[i].Period)
		require.Equal(t, userSubscription.StartDate.UTC(), got.UserSubscriptions[i].StartDate.UTC())
		require.Equal(t, userSubscription.LastRenewed.UTC(), got.UserSubscriptions[i].LastRenewed.UTC())
	}

	nullify.Fill(&genesisState)
	nullify.Fill(got)
}

func TestInitWithMultipleIdenticalLockups(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
		Lockups: []types.Lockup{
			{
				DelegatorAddress: "source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9",
				ValidatorAddress: "sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm",
				Amount:           math.NewInt(1000),
			},
			{
				DelegatorAddress: "source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9",
				ValidatorAddress: "sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm",
				Amount:           math.NewInt(2000),
			},
			{
				DelegatorAddress: "source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9",
				ValidatorAddress: "sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm",
				Amount:           math.NewInt(3000),
			},
		},
	}

	k, ctx := keepertest.TierKeeper(t)
	tier.InitGenesis(ctx, &k, genesisState)
	got := tier.ExportGenesis(ctx, &k)
	require.NotNil(t, got)
	require.Equal(t, genesisState.Params, got.Params)

	// Lockups with identical del/val are added and exported as a single record.
	require.Equal(t, 1, len(got.Lockups))
	require.Equal(t, int64(6000), got.Lockups[0].Amount.Int64())

	nullify.Fill(&genesisState)
	nullify.Fill(got)
}

func TestInitWithMultipleIdenticalUnlockingLockups(t *testing.T) {
	timestamp1 := time.Date(2006, time.January, 2, 15, 4, 5, 1, time.UTC)
	timestamp2 := time.Date(2006, time.January, 2, 15, 4, 5, 2, time.UTC)
	timestamp3 := time.Date(2006, time.January, 2, 15, 4, 5, 3, time.UTC)
	timestamp4 := time.Date(2006, time.January, 2, 15, 4, 5, 4, time.UTC)
	timestamp5 := time.Date(2006, time.January, 2, 15, 4, 5, 5, time.UTC)
	timestamp6 := time.Date(2006, time.January, 2, 15, 4, 5, 6, time.UTC)

	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
		UnlockingLockups: []types.UnlockingLockup{
			{
				DelegatorAddress: "source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9",
				ValidatorAddress: "sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm",
				Amount:           math.NewInt(1000),
				CreationHeight:   1,
				CompletionTime:   timestamp1,
				UnlockTime:       timestamp4,
			},
			{
				DelegatorAddress: "source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9",
				ValidatorAddress: "sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm",
				Amount:           math.NewInt(2000),
				CreationHeight:   2,
				CompletionTime:   timestamp2,
				UnlockTime:       timestamp5,
			},
			{
				DelegatorAddress: "source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9",
				ValidatorAddress: "sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm",
				Amount:           math.NewInt(3000),
				CreationHeight:   3,
				CompletionTime:   timestamp3,
				UnlockTime:       timestamp6,
			},
		},
	}

	k, ctx := keepertest.TierKeeper(t)
	tier.InitGenesis(ctx, &k, genesisState)
	got := tier.ExportGenesis(ctx, &k)
	require.NotNil(t, got)
	require.Equal(t, genesisState.Params, got.Params)

	// Unlocking lockups with identical del/val and different creationHeight are added and exported separately.
	require.Equal(t, 3, len(got.UnlockingLockups))
	require.Equal(t, int64(1000), got.UnlockingLockups[0].Amount.Int64())
	require.Equal(t, int64(2000), got.UnlockingLockups[1].Amount.Int64())
	require.Equal(t, int64(3000), got.UnlockingLockups[2].Amount.Int64())

	nullify.Fill(&genesisState)
	nullify.Fill(got)
}

func TestInitWithMultipleDevelopers(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
		Developers: []types.Developer{
			{
				Address:         "source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et",
				AutoLockEnabled: true,
			},
			{
				Address:         "source1n34fvpteuanu2nx2a4hql4jvcrcnal3gsrjppy",
				AutoLockEnabled: false,
			},
			{
				Address:         "source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9",
				AutoLockEnabled: true,
			},
		},
	}

	k, ctx := keepertest.TierKeeper(t)
	tier.InitGenesis(ctx, &k, genesisState)
	got := tier.ExportGenesis(ctx, &k)
	require.NotNil(t, got)
	require.Equal(t, genesisState.Params, got.Params)

	// All developers should be exported correctly
	require.Equal(t, 3, len(got.Developers))
	require.Equal(t, "source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et", got.Developers[0].Address)
	require.Equal(t, true, got.Developers[0].AutoLockEnabled)
	require.Equal(t, "source1n34fvpteuanu2nx2a4hql4jvcrcnal3gsrjppy", got.Developers[1].Address)
	require.Equal(t, false, got.Developers[1].AutoLockEnabled)
	require.Equal(t, "source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9", got.Developers[2].Address)
	require.Equal(t, true, got.Developers[2].AutoLockEnabled)

	nullify.Fill(&genesisState)
	nullify.Fill(got)
}

func TestInitWithMultipleUserSubscriptions(t *testing.T) {
	timestamp1 := time.Date(2006, time.January, 2, 15, 4, 5, 1, time.UTC)
	timestamp2 := time.Date(2006, time.January, 2, 15, 4, 5, 2, time.UTC)
	timestamp3 := time.Date(2006, time.January, 2, 15, 4, 5, 3, time.UTC)

	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
		UserSubscriptions: []types.UserSubscription{
			{
				Developer:    "source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et",
				UserDid:      "did:key:alice",
				CreditAmount: sdk.NewCoin("ucredit", math.NewInt(1000)),
				Period:       30,
				StartDate:    timestamp1,
				LastRenewed:  timestamp2,
			},
			{
				Developer:    "source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et",
				UserDid:      "did:key:bob",
				CreditAmount: sdk.NewCoin("ucredit", math.NewInt(500)),
				Period:       60,
				StartDate:    timestamp2,
				LastRenewed:  timestamp3,
			},
			{
				Developer:    "source1n34fvpteuanu2nx2a4hql4jvcrcnal3gsrjppy",
				UserDid:      "did:key:charlie",
				CreditAmount: sdk.NewCoin("ucredit", math.NewInt(2000)),
				Period:       90,
				StartDate:    timestamp3,
				LastRenewed:  timestamp1,
			},
		},
	}

	k, ctx := keepertest.TierKeeper(t)
	tier.InitGenesis(ctx, &k, genesisState)
	got := tier.ExportGenesis(ctx, &k)
	require.NotNil(t, got)
	require.Equal(t, genesisState.Params, got.Params)

	// All user subscriptions should be exported correctly
	require.Equal(t, 3, len(got.UserSubscriptions))
	require.Equal(t, "source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et", got.UserSubscriptions[0].Developer)
	require.Equal(t, "did:key:alice", got.UserSubscriptions[0].UserDid)
	require.Equal(t, int64(1000), got.UserSubscriptions[0].CreditAmount.Amount.Int64())
	require.Equal(t, uint64(30), got.UserSubscriptions[0].Period)
	require.Equal(t, "source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et", got.UserSubscriptions[1].Developer)
	require.Equal(t, "did:key:bob", got.UserSubscriptions[1].UserDid)
	require.Equal(t, int64(500), got.UserSubscriptions[1].CreditAmount.Amount.Int64())
	require.Equal(t, uint64(60), got.UserSubscriptions[1].Period)
	require.Equal(t, "source1n34fvpteuanu2nx2a4hql4jvcrcnal3gsrjppy", got.UserSubscriptions[2].Developer)
	require.Equal(t, "did:key:charlie", got.UserSubscriptions[2].UserDid)
	require.Equal(t, int64(2000), got.UserSubscriptions[2].CreditAmount.Amount.Int64())
	require.Equal(t, uint64(90), got.UserSubscriptions[2].Period)

	nullify.Fill(&genesisState)
	nullify.Fill(got)
}
