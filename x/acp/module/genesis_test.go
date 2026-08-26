package acp_test

import (
	"testing"

	keepertest "github.com/sourcenetwork/vera/testutil/keeper"
	"github.com/sourcenetwork/vera/testutil/nullify"
	acp "github.com/sourcenetwork/vera/x/acp/module"
	"github.com/sourcenetwork/vera/x/acp/types"
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
	require.Equal(t, genesisState.Params, got.Params)

	nullify.Fill(&genesisState)
	nullify.Fill(got)
}
