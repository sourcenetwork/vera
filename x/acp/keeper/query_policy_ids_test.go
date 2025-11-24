package keeper

import (
	"context"
	"strconv"
	"testing"

	"github.com/cosmos/cosmos-sdk/types/query"
	"gopkg.in/yaml.v3"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

type queryPolicyIdsSuite struct {
	suite.Suite
}

func TestPolicyIds(t *testing.T) {
	suite.Run(t, &queryPolicyIdsSuite{})
}

func (s *queryPolicyIdsSuite) setupPolicies(
	t *testing.T,
	ctx context.Context,
	k Keeper,
	creator string,
	policyNames []string,
	marshalingType coretypes.PolicyMarshalingType,
) []string {
	policyIds := []string{}

	for _, name := range policyNames {
		policy := &coretypes.PolicyShort{
			Name:        name,
			Description: "Test policy for " + name,
			Meta: map[string]string{
				"k1": "v1",
				"k2": "v2",
			},
			Resources: map[string]*coretypes.ResourceShort{
				"file": {
					Doc: "A test resource",
					Permissions: map[string]*coretypes.PermissionShort{
						"manage": {
							Doc:  "Permission to manage resources",
							Expr: "owner",
						},
					},
					Relations: map[string]*coretypes.RelationShort{
						"owner": {
							Doc:     "Owner relation",
							Manages: []string{"reader"},
							Types: []string{
								"actor-resource->",
							},
						},
						"reader": {
							Doc: "Reader relation",
						},
					},
				},
			},
			Actor: &coretypes.ActorResource{
				Name: "actor-resource",
				Doc:  "Test actor resource",
			},
		}

		var policyString string
		switch marshalingType {
		case coretypes.PolicyMarshalingType_YAML:
			policyYAML, err := yaml.Marshal(policy)
			require.NoError(t, err, "failed to marshal policy to YAML")
			policyString = string(policyYAML)
		default:
			t.Fatalf("unsupported marshaling type: %v", marshalingType)
		}

		msg := types.MsgCreatePolicy{
			Creator:     creator,
			Policy:      policyString,
			MarshalType: marshalingType,
		}

		resp, err := k.CreatePolicy(ctx, &msg)
		require.NoError(t, err)
		require.NotNil(t, resp)

		policyIds = append(policyIds, resp.Record.Policy.Id)
	}

	return policyIds
}

func (s *queryPolicyIdsSuite) TestQueryPolicyIds_YAML() {
	ctx, k, accKeep := setupKeeper(s.T())

	creator := accKeep.FirstAcc().GetAddress().String()

	policyIds := s.setupPolicies(s.T(), ctx, k, creator, []string{"P1", "P2", "P3"}, coretypes.PolicyMarshalingType_YAML)

	resp, err := k.PolicyIds(ctx, &types.QueryPolicyIdsRequest{})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), resp)
	require.ElementsMatch(s.T(), policyIds, resp.Ids)
}

func (s *queryPolicyIdsSuite) TestQueryPolicyIds_NoPoliciesRegistered() {
	ctx, k, _ := setupKeeper(s.T())

	resp, err := k.PolicyIds(ctx, &types.QueryPolicyIdsRequest{})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), resp)
	require.Empty(s.T(), resp.Ids)
}

func (s *queryPolicyIdsSuite) TestQueryPolicyIds_DuplicatePolicyNames() {
	ctx, k, accKeep := setupKeeper(s.T())

	creator := accKeep.FirstAcc().GetAddress().String()

	_ = s.setupPolicies(s.T(), ctx, k, creator, []string{"P1", "P1"}, coretypes.PolicyMarshalingType_YAML)

	resp, err := k.PolicyIds(ctx, &types.QueryPolicyIdsRequest{})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), resp)
	require.Equal(s.T(), 2, len(resp.Ids))
}

func (s *queryPolicyIdsSuite) TestQueryPolicyIds_LargeNumberOfPolicies_YAML() {
	ctx, k, accKeep := setupKeeper(s.T())

	creator := accKeep.FirstAcc().GetAddress().String()

	names := []string{}
	for i := 0; i < 10_000; i++ {
		names = append(names, "Policy"+strconv.Itoa(i))
	}
	policyIds := s.setupPolicies(s.T(), ctx, k, creator, names, coretypes.PolicyMarshalingType_YAML)

	resp, err := k.PolicyIds(ctx, &types.QueryPolicyIdsRequest{})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), resp)
	require.ElementsMatch(s.T(), policyIds, resp.Ids)
}

func (s *queryPolicyIdsSuite) TestQueryPolicyIds_WithPagination() {
	ctx, k, accKeep := setupKeeper(s.T())

	creator := accKeep.FirstAcc().GetAddress().String()

	// Create 25 policies
	names := []string{}
	for i := 0; i < 25; i++ {
		names = append(names, "Policy"+strconv.Itoa(i))
	}
	allPolicyIds := s.setupPolicies(s.T(), ctx, k, creator, names, coretypes.PolicyMarshalingType_YAML)

	// Test first page with limit 10
	resp1, err := k.PolicyIds(ctx, &types.QueryPolicyIdsRequest{
		Pagination: &query.PageRequest{
			Limit: 10,
		},
	})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), resp1)
	require.Len(s.T(), resp1.Ids, 10)
	require.Equal(s.T(), uint64(25), resp1.Pagination.Total)
	require.NotNil(s.T(), resp1.Pagination.NextKey)

	// Test second page with offset 10
	resp2, err := k.PolicyIds(ctx, &types.QueryPolicyIdsRequest{
		Pagination: &query.PageRequest{
			Offset: 10,
			Limit:  10,
		},
	})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), resp2)
	require.Len(s.T(), resp2.Ids, 10)
	require.Equal(s.T(), uint64(25), resp2.Pagination.Total)

	// Test third page with offset 20 (should get remaining 5)
	resp3, err := k.PolicyIds(ctx, &types.QueryPolicyIdsRequest{
		Pagination: &query.PageRequest{
			Offset: 20,
			Limit:  10,
		},
	})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), resp3)
	require.Len(s.T(), resp3.Ids, 5)
	require.Equal(s.T(), uint64(25), resp3.Pagination.Total)
	require.Nil(s.T(), resp3.Pagination.NextKey)

	// Verify all pages combined match all policy IDs
	allPages := append(resp1.Ids, resp2.Ids...)
	allPages = append(allPages, resp3.Ids...)
	require.ElementsMatch(s.T(), allPolicyIds, allPages)

	// Verify no overlap between pages
	require.NotContains(s.T(), resp2.Ids, resp1.Ids[0])
	require.NotContains(s.T(), resp3.Ids, resp1.Ids[0])
	require.NotContains(s.T(), resp3.Ids, resp2.Ids[0])
}

func (s *queryPolicyIdsSuite) TestQueryPolicyIds_WithKeyBasedPagination() {
	ctx, k, accKeep := setupKeeper(s.T())

	creator := accKeep.FirstAcc().GetAddress().String()

	// Create 25 policies
	names := []string{}
	for i := 0; i < 25; i++ {
		names = append(names, "Policy"+strconv.Itoa(i))
	}
	allPolicyIds := s.setupPolicies(s.T(), ctx, k, creator, names, coretypes.PolicyMarshalingType_YAML)

	// Test first page with limit 10
	resp1, err := k.PolicyIds(ctx, &types.QueryPolicyIdsRequest{
		Pagination: &query.PageRequest{
			Limit: 10,
		},
	})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), resp1)
	require.Len(s.T(), resp1.Ids, 10)
	require.Equal(s.T(), uint64(25), resp1.Pagination.Total)
	require.NotNil(s.T(), resp1.Pagination.NextKey)

	// Test second page using NextKey from first page
	resp2, err := k.PolicyIds(ctx, &types.QueryPolicyIdsRequest{
		Pagination: &query.PageRequest{
			Key:   resp1.Pagination.NextKey,
			Limit: 10,
		},
	})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), resp2)
	require.Len(s.T(), resp2.Ids, 10)
	require.Equal(s.T(), uint64(25), resp2.Pagination.Total)
	require.NotNil(s.T(), resp2.Pagination.NextKey)

	// Test third page using NextKey from second page
	resp3, err := k.PolicyIds(ctx, &types.QueryPolicyIdsRequest{
		Pagination: &query.PageRequest{
			Key:   resp2.Pagination.NextKey,
			Limit: 10,
		},
	})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), resp3)
	require.Len(s.T(), resp3.Ids, 5)
	require.Equal(s.T(), uint64(25), resp3.Pagination.Total)
	require.Nil(s.T(), resp3.Pagination.NextKey)

	// Verify all pages combined match all policy IDs
	allPages := append(resp1.Ids, resp2.Ids...)
	allPages = append(allPages, resp3.Ids...)
	require.ElementsMatch(s.T(), allPolicyIds, allPages)

	// Verify no overlap between pages
	require.NotContains(s.T(), resp2.Ids, resp1.Ids[0])
	require.NotContains(s.T(), resp3.Ids, resp1.Ids[0])
	require.NotContains(s.T(), resp3.Ids, resp2.Ids[0])
}

func (s *queryPolicyIdsSuite) TestQueryPolicyIds_LargeOffset() {
	ctx, k, accKeep := setupKeeper(s.T())

	creator := accKeep.FirstAcc().GetAddress().String()

	// Create policies to test large offset handling
	names := []string{}
	for i := 0; i < 100; i++ {
		names = append(names, "Policy"+strconv.Itoa(i))
	}
	_ = s.setupPolicies(s.T(), ctx, k, creator, names, coretypes.PolicyMarshalingType_YAML)

	// Test with offset beyond total
	resp, err := k.PolicyIds(ctx, &types.QueryPolicyIdsRequest{
		Pagination: &query.PageRequest{
			Offset: 300,
			Limit:  10,
		},
	})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), resp)
	require.Empty(s.T(), resp.Ids)
	require.Equal(s.T(), uint64(100), resp.Pagination.Total)
}

func (s *queryPolicyIdsSuite) TestQueryPolicyIds_MalformedKey() {
	ctx, k, accKeep := setupKeeper(s.T())

	creator := accKeep.FirstAcc().GetAddress().String()

	names := []string{"P1", "P2", "P3"}
	_ = s.setupPolicies(s.T(), ctx, k, creator, names, coretypes.PolicyMarshalingType_YAML)

	// Test with malformed key
	resp, err := k.PolicyIds(ctx, &types.QueryPolicyIdsRequest{
		Pagination: &query.PageRequest{
			Key:   []byte{0x01, 0x02}, // Only 2 bytes instead of 8
			Limit: 10,
		},
	})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), resp)
	require.Empty(s.T(), resp.Ids) // Should return empty on malformed key
	require.Equal(s.T(), uint64(3), resp.Pagination.Total)
}
