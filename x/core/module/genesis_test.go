package core_test

import (
	"testing"
	"time"

	keepertest "github.com/sourcenetwork/vera/testutil/keeper"
	"github.com/sourcenetwork/vera/testutil/nullify"
	core "github.com/sourcenetwork/vera/x/core/module"
	"github.com/sourcenetwork/vera/x/core/types"
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

	k, ctx := keepertest.CoreKeeper(t)
	core.InitGenesis(ctx, &k, genesisState)
	got := core.ExportGenesis(ctx, &k)
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

	k, ctx := keepertest.CoreKeeper(t)
	core.InitGenesis(ctx, &k, genesisState)
	got := core.ExportGenesis(ctx, &k)
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

	k, ctx := keepertest.CoreKeeper(t)
	core.InitGenesis(ctx, &k, genesisState)
	got := core.ExportGenesis(ctx, &k)
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

	k, ctx := keepertest.CoreKeeper(t)
	core.InitGenesis(ctx, &k, genesisState)
	got := core.ExportGenesis(ctx, &k)
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

	k, ctx := keepertest.CoreKeeper(t)

	// Genesis initialization should panic with invalid ICA connections
	require.Panics(t, func() {
		core.InitGenesis(ctx, &k, genesisState)
	})
}

func TestGenesisWithJWSTokens(t *testing.T) {
	now := time.Now()
	later := now.Add(24 * time.Hour)

	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
		JwsTokens: []types.JWSTokenRecord{
			{
				TokenHash:         "hash1",
				BearerToken:       "token1",
				IssuerDid:         "did:key:alice",
				AuthorizedAccount: "source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9",
				IssuedAt:          now,
				ExpiresAt:         later,
				Status:            types.JWSTokenStatus_STATUS_VALID,
				FirstUsedAt:       &now,
				LastUsedAt:        &now,
			},
			{
				TokenHash:         "hash2",
				BearerToken:       "token2",
				IssuerDid:         "did:key:bob",
				AuthorizedAccount: "source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et",
				IssuedAt:          now,
				ExpiresAt:         later,
				Status:            types.JWSTokenStatus_STATUS_VALID,
				FirstUsedAt:       &now,
				LastUsedAt:        &now,
			},
			{
				TokenHash:         "hash3",
				BearerToken:       "token3",
				IssuerDid:         "did:key:charlie",
				AuthorizedAccount: "source18jtkvj0995fy7lggqayg2f5syna92ndq5mkuv4",
				IssuedAt:          now,
				ExpiresAt:         later,
				Status:            types.JWSTokenStatus_STATUS_INVALID,
				FirstUsedAt:       &now,
				LastUsedAt:        &now,
				InvalidatedAt:     &now,
				InvalidatedBy:     "source18jtkvj0995fy7lggqayg2f5syna92ndq5mkuv4",
			},
		},
	}

	k, ctx := keepertest.CoreKeeper(t)
	core.InitGenesis(ctx, &k, genesisState)
	got := core.ExportGenesis(ctx, &k)

	require.NotNil(t, got)
	require.Equal(t, genesisState.Params, got.Params)
	require.Equal(t, len(genesisState.JwsTokens), len(got.JwsTokens))

	// Verify all tokens are preserved
	for i, token := range genesisState.JwsTokens {
		require.Equal(t, token.TokenHash, got.JwsTokens[i].TokenHash)
		require.Equal(t, token.BearerToken, got.JwsTokens[i].BearerToken)
		require.Equal(t, token.IssuerDid, got.JwsTokens[i].IssuerDid)
		require.Equal(t, token.AuthorizedAccount, got.JwsTokens[i].AuthorizedAccount)
		require.Equal(t, token.Status, got.JwsTokens[i].Status)
		require.Equal(t, token.IssuedAt.Unix(), got.JwsTokens[i].IssuedAt.Unix())
		require.Equal(t, token.ExpiresAt.Unix(), got.JwsTokens[i].ExpiresAt.Unix())
	}

	nullify.Fill(&genesisState)
	nullify.Fill(got)
}

func TestGenesisWithEmptyJWSTokens(t *testing.T) {
	genesisState := types.GenesisState{
		Params:    types.DefaultParams(),
		JwsTokens: []types.JWSTokenRecord{},
	}

	k, ctx := keepertest.CoreKeeper(t)
	core.InitGenesis(ctx, &k, genesisState)
	got := core.ExportGenesis(ctx, &k)

	require.NotNil(t, got)
	require.Equal(t, genesisState.Params, got.Params)
	require.Equal(t, 0, len(got.JwsTokens))

	nullify.Fill(&genesisState)
	nullify.Fill(got)
}

func TestGenesisWithInvalidJWSToken(t *testing.T) {
	now := time.Now()
	later := now.Add(24 * time.Hour)

	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
		JwsTokens: []types.JWSTokenRecord{
			{
				TokenHash:         "",
				BearerToken:       "token1",
				IssuerDid:         "did:key:alice",
				AuthorizedAccount: "source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9",
				IssuedAt:          now,
				ExpiresAt:         later,
				Status:            types.JWSTokenStatus_STATUS_VALID,
			},
		},
	}

	k, ctx := keepertest.CoreKeeper(t)

	// Should panic with empty token hash
	require.Panics(t, func() {
		core.InitGenesis(ctx, &k, genesisState)
	})
}

func TestGenesisWithMultipleTokensSameDID(t *testing.T) {
	now := time.Now()
	later := now.Add(24 * time.Hour)

	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
		JwsTokens: []types.JWSTokenRecord{
			{
				TokenHash:         "hash1",
				BearerToken:       "token1",
				IssuerDid:         "did:key:alice",
				AuthorizedAccount: "source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9",
				IssuedAt:          now,
				ExpiresAt:         later,
				Status:            types.JWSTokenStatus_STATUS_VALID,
				FirstUsedAt:       &now,
				LastUsedAt:        &now,
			},
			{
				TokenHash:         "hash2",
				BearerToken:       "token2",
				IssuerDid:         "did:key:alice",
				AuthorizedAccount: "source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9",
				IssuedAt:          now,
				ExpiresAt:         later,
				Status:            types.JWSTokenStatus_STATUS_VALID,
				FirstUsedAt:       &now,
				LastUsedAt:        &now,
			},
		},
	}

	k, ctx := keepertest.CoreKeeper(t)
	core.InitGenesis(ctx, &k, genesisState)
	got := core.ExportGenesis(ctx, &k)

	require.NotNil(t, got)
	require.Equal(t, 2, len(got.JwsTokens))

	// Both tokens should be preserved (different hashes)
	tokenHashes := make(map[string]bool)
	for _, token := range got.JwsTokens {
		tokenHashes[token.TokenHash] = true
	}
	require.True(t, tokenHashes["hash1"])
	require.True(t, tokenHashes["hash2"])

	nullify.Fill(&genesisState)
	nullify.Fill(got)
}

func TestGenesisWithJWSTokensAndICAConnections(t *testing.T) {
	now := time.Now()
	later := now.Add(24 * time.Hour)

	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
		IcaConnections: []types.ICAConnection{
			{
				IcaAddress:        "source18jtkvj0995fy7lggqayg2f5syna92ndq5mkuv4",
				ControllerAddress: "shinzo1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et",
				ControllerChainId: "shinzo-1",
				ConnectionId:      "connection-0",
			},
		},
		JwsTokens: []types.JWSTokenRecord{
			{
				TokenHash:         "hash1",
				BearerToken:       "token1",
				IssuerDid:         "did:key:alice",
				AuthorizedAccount: "source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9",
				IssuedAt:          now,
				ExpiresAt:         later,
				Status:            types.JWSTokenStatus_STATUS_VALID,
				FirstUsedAt:       &now,
				LastUsedAt:        &now,
			},
		},
	}

	k, ctx := keepertest.CoreKeeper(t)
	core.InitGenesis(ctx, &k, genesisState)
	got := core.ExportGenesis(ctx, &k)

	require.NotNil(t, got)
	require.Equal(t, genesisState.Params, got.Params)
	require.Equal(t, 1, len(got.IcaConnections))
	require.Equal(t, 1, len(got.JwsTokens))

	// Verify ICA connection
	require.Equal(t, genesisState.IcaConnections[0].IcaAddress, got.IcaConnections[0].IcaAddress)

	// Verify JWS token
	require.Equal(t, genesisState.JwsTokens[0].TokenHash, got.JwsTokens[0].TokenHash)
	require.Equal(t, genesisState.JwsTokens[0].IssuerDid, got.JwsTokens[0].IssuerDid)

	nullify.Fill(&genesisState)
	nullify.Fill(got)
}
