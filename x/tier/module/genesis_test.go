package tier_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
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
