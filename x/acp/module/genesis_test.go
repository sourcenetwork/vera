package acp_test

import (
	"testing"

	keepertest "github.com/sourcenetwork/sourcehub/testutil/keeper"
	"github.com/sourcenetwork/sourcehub/testutil/nullify"
	acp "github.com/sourcenetwork/sourcehub/x/acp/module"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
	}

	k, ctx := keepertest.AcpKeeper(t)
	acp.InitGenesis(ctx, &k, genesisState)
	got := acp.ExportGenesis(ctx, &k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)
}
