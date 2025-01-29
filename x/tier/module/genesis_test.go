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

	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
		Lockups: []types.Lockup{
			{
				DelegatorAddress: "source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et",
				ValidatorAddress: "sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm",
				Amount:           math.NewInt(1000),
				CreationHeight:   1,
				UnbondTime:       &timestamp1,
				UnlockTime:       &timestamp1,
			},
			{
				DelegatorAddress: "source1n34fvpteuanu2nx2a4hql4jvcrcnal3gsrjppy",
				ValidatorAddress: "sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm",
				Amount:           math.NewInt(500),
				CreationHeight:   2,
				UnbondTime:       &timestamp2,
				UnlockTime:       &timestamp2,
			},
			{
				DelegatorAddress: "source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9",
				ValidatorAddress: "sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm",
				Amount:           math.NewInt(2000),
				CreationHeight:   3,
				UnbondTime:       &timestamp3,
				UnlockTime:       &timestamp3,
			},
		},
	}

	k, ctx := keepertest.TierKeeper(t)
	tier.InitGenesis(ctx, k, genesisState)
	got := tier.ExportGenesis(ctx, k)
	require.NotNil(t, got)
	require.Equal(t, genesisState.Params, got.Params)
	require.Equal(t, len(genesisState.Lockups), len(got.Lockups))

	for i, lockup := range genesisState.Lockups {
		require.Equal(t, lockup.ValidatorAddress, got.Lockups[i].ValidatorAddress)
		require.Equal(t, lockup.Amount, got.Lockups[i].Amount)
		require.Equal(t, lockup.CreationHeight, got.Lockups[i].CreationHeight)
		if lockup.UnbondTime != nil {
			require.NotNil(t, got.Lockups[i].UnbondTime)
			require.Equal(t, lockup.UnbondTime.UTC(), got.Lockups[i].UnbondTime.UTC())
		} else {
			require.Nil(t, got.Lockups[i].UnbondTime)
		}
		if lockup.UnlockTime != nil {
			require.NotNil(t, got.Lockups[i].UnlockTime)
			require.Equal(t, lockup.UnlockTime.UTC(), got.Lockups[i].UnlockTime.UTC())
		} else {
			require.Nil(t, got.Lockups[i].UnlockTime)
		}
	}

	nullify.Fill(&genesisState)
	nullify.Fill(got)
}

func TestInitWithMultipleIdenticalLockups(t *testing.T) {
	timestamp1 := time.Date(2006, time.January, 2, 15, 4, 5, 1, time.UTC)
	timestamp2 := time.Date(2006, time.January, 2, 15, 4, 5, 2, time.UTC)
	timestamp3 := time.Date(2006, time.January, 2, 15, 4, 5, 3, time.UTC)

	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
		Lockups: []types.Lockup{
			{
				DelegatorAddress: "source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9",
				ValidatorAddress: "sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm",
				Amount:           math.NewInt(1000),
				CreationHeight:   1,
				UnbondTime:       &timestamp1,
				UnlockTime:       nil,
			},
			{
				DelegatorAddress: "source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9",
				ValidatorAddress: "sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm",
				Amount:           math.NewInt(2000),
				CreationHeight:   2,
				UnbondTime:       &timestamp2,
				UnlockTime:       nil,
			},
			{
				DelegatorAddress: "source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9",
				ValidatorAddress: "sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm",
				Amount:           math.NewInt(3000),
				CreationHeight:   3,
				UnbondTime:       &timestamp3,
				UnlockTime:       nil,
			},
		},
	}

	k, ctx := keepertest.TierKeeper(t)
	tier.InitGenesis(ctx, k, genesisState)
	got := tier.ExportGenesis(ctx, k)
	require.NotNil(t, got)
	require.Equal(t, genesisState.Params, got.Params)

	// Lockups of identical del/val records are added and exported as a single record.
	require.Equal(t, 1, len(got.Lockups))
	require.Equal(t, int64(6000), got.Lockups[0].Amount.Int64())

	nullify.Fill(&genesisState)
	nullify.Fill(got)
}
