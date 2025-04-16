package bulletin

import (
	"testing"

	keepertest "github.com/sourcenetwork/sourcehub/testutil/keeper"
	"github.com/sourcenetwork/sourcehub/testutil/nullify"
	"github.com/sourcenetwork/sourcehub/x/bulletin/types"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
	}

	k, ctx := keepertest.BulletinKeeper(t)
	InitGenesis(ctx, &k, genesisState)
	got := ExportGenesis(ctx, &k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)
}
