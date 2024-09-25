package bulletin_test

import (
	"testing"

	keepertest "github.com/sourcenetwork/sourcehub/testutil/keeper"
	"github.com/sourcenetwork/sourcehub/testutil/nullify"
	bulletin "github.com/sourcenetwork/sourcehub/x/bulletin/module"
	"github.com/sourcenetwork/sourcehub/x/bulletin/types"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),

		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.BulletinKeeper(t)
	bulletin.InitGenesis(ctx, k, genesisState)
	got := bulletin.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	// this line is used by starport scaffolding # genesis/test/assert
}
