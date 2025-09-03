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
		IcaConnections: []types.ICAConnection{
			{
				IcaAddress:        "source18jtkvj0995fy7lggqayg2f5syna92ndq5mkuv4",
				ControllerAddress: "shinzo1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et",
				ControllerChainId: "shinzo-1",
				ConnectionId:      "connection-0",
			},
			{
				IcaAddress:        "source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9",
				ControllerAddress: "shinzo18jtkvj0995fy7lggqayg2f5syna92ndq5mkuv4",
				ControllerChainId: "shinzo-1",
				ConnectionId:      "connection-1",
			},
			{
				IcaAddress:        "source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et",
				ControllerAddress: "shinzo1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9",
				ControllerChainId: "shinzo-1",
				ConnectionId:      "connection-2",
			},
		},
	}

	k, ctx := keepertest.AcpKeeper(t)
	acp.InitGenesis(ctx, &k, genesisState)
	got := acp.ExportGenesis(ctx, &k)
	require.NotNil(t, got)
	require.Equal(t, genesisState.Params, got.Params)
	require.Equal(t, len(genesisState.IcaConnections), len(got.IcaConnections))

	for i, connection := range genesisState.IcaConnections {
		require.Equal(t, connection.IcaAddress, got.IcaConnections[i].IcaAddress)
		require.Equal(t, connection.ControllerAddress, got.IcaConnections[i].ControllerAddress)
		require.Equal(t, connection.ControllerChainId, got.IcaConnections[i].ControllerChainId)
		require.Equal(t, connection.ConnectionId, got.IcaConnections[i].ConnectionId)
	}

	nullify.Fill(&genesisState)
	nullify.Fill(got)
}

func TestInitWithEmptyIcaConnections(t *testing.T) {
	genesisState := types.GenesisState{
		Params:         types.DefaultParams(),
		IcaConnections: []types.ICAConnection{},
	}

	k, ctx := keepertest.AcpKeeper(t)
	acp.InitGenesis(ctx, &k, genesisState)
	got := acp.ExportGenesis(ctx, &k)
	require.NotNil(t, got)
	require.Equal(t, genesisState.Params, got.Params)
	require.Equal(t, 0, len(got.IcaConnections))

	nullify.Fill(&genesisState)
	nullify.Fill(got)
}

func TestInitWithMultipleIdenticalIcaConnections(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
		IcaConnections: []types.ICAConnection{
			{
				IcaAddress:        "source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9",
				ControllerAddress: "shinzo1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et",
				ControllerChainId: "shinzo-1",
				ConnectionId:      "connection-0",
			},
			{
				IcaAddress:        "source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9",
				ControllerAddress: "shinzo1n34fvpteuanu2nx2a4hql4jvcrcnal3gsrjppy",
				ControllerChainId: "shinzo-1",
				ConnectionId:      "connection-1",
			},
		},
	}

	k, ctx := keepertest.AcpKeeper(t)
	acp.InitGenesis(ctx, &k, genesisState)
	got := acp.ExportGenesis(ctx, &k)
	require.NotNil(t, got)
	require.Equal(t, genesisState.Params, got.Params)

	// Multiple connections with identical ICA addresses should overwrite each other
	require.Equal(t, 1, len(got.IcaConnections))
	require.Equal(t, "source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9", got.IcaConnections[0].IcaAddress)
	require.Equal(t, "shinzo1n34fvpteuanu2nx2a4hql4jvcrcnal3gsrjppy", got.IcaConnections[0].ControllerAddress)
	require.Equal(t, "connection-1", got.IcaConnections[0].ConnectionId)

	nullify.Fill(&genesisState)
	nullify.Fill(got)
}

func TestInitWithDifferentChainIds(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
		IcaConnections: []types.ICAConnection{
			{
				IcaAddress:        "source18jtkvj0995fy7lggqayg2f5syna92ndq5mkuv4",
				ControllerAddress: "shinzo1cy0p47z24ejzvq55pu3lesxwf73xnrnd0lyxme",
				ControllerChainId: "shinzo-1",
				ConnectionId:      "connection-0",
			},
			{
				IcaAddress:        "source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9",
				ControllerAddress: "cosmos1cy0p47z24ejzvq55pu3lesxwf73xnrnd0lyxme",
				ControllerChainId: "cosmoshub-4",
				ConnectionId:      "connection-1",
			},
			{
				IcaAddress:        "source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et",
				ControllerAddress: "osmosis1cy0p47z24ejzvq55pu3lesxwf73xnrnd0lyxme",
				ControllerChainId: "osmosis-1",
				ConnectionId:      "connection-2",
			},
		},
	}

	k, ctx := keepertest.AcpKeeper(t)
	acp.InitGenesis(ctx, &k, genesisState)
	got := acp.ExportGenesis(ctx, &k)
	require.NotNil(t, got)
	require.Equal(t, genesisState.Params, got.Params)
	require.Equal(t, len(genesisState.IcaConnections), len(got.IcaConnections))

	// Verify all connections are preserved with different chain IDs
	for i, connection := range genesisState.IcaConnections {
		require.Equal(t, connection.IcaAddress, got.IcaConnections[i].IcaAddress)
		require.Equal(t, connection.ControllerAddress, got.IcaConnections[i].ControllerAddress)
		require.Equal(t, connection.ControllerChainId, got.IcaConnections[i].ControllerChainId)
		require.Equal(t, connection.ConnectionId, got.IcaConnections[i].ConnectionId)
	}

	nullify.Fill(&genesisState)
	nullify.Fill(got)
}

func TestInitWithEmptyStrings(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
		IcaConnections: []types.ICAConnection{
			{
				IcaAddress:        "",
				ControllerAddress: "",
				ControllerChainId: "",
				ConnectionId:      "",
			},
		},
	}

	k, ctx := keepertest.AcpKeeper(t)
	
	// Genesis initialization should panic with invalid ICA connections
	require.Panics(t, func() {
		acp.InitGenesis(ctx, &k, genesisState)
	})
}
