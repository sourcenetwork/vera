package keeper

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"

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
		case coretypes.PolicyMarshalingType_SHORT_YAML:
			policyYAML, err := yaml.Marshal(policy)
			require.NoError(t, err, "failed to marshal policy to YAML")
			policyString = string(policyYAML)
		case coretypes.PolicyMarshalingType_SHORT_JSON:
			policyJSON, err := json.Marshal(policy)
			require.NoError(t, err, "failed to marshal policy to JSON")
			policyString = string(policyJSON)
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

	policyIds := s.setupPolicies(s.T(), ctx, k, creator, []string{"P1", "P2", "P3"}, coretypes.PolicyMarshalingType_SHORT_YAML)

	resp, err := k.PolicyIds(ctx, &types.QueryPolicyIdsRequest{})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), resp)
	require.ElementsMatch(s.T(), policyIds, resp.Ids)
}

func (s *queryPolicyIdsSuite) TestQueryPolicyIds_JSON() {
	ctx, k, accKeep := setupKeeper(s.T())

	creator := accKeep.FirstAcc().GetAddress().String()

	policyIds := s.setupPolicies(s.T(), ctx, k, creator, []string{"P1", "P2", "P3"}, coretypes.PolicyMarshalingType_SHORT_JSON)

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

	_ = s.setupPolicies(s.T(), ctx, k, creator, []string{"P1", "P1"}, coretypes.PolicyMarshalingType_SHORT_YAML)

	resp, err := k.PolicyIds(ctx, &types.QueryPolicyIdsRequest{})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), resp)
	require.Equal(s.T(), 2, len(resp.Ids))
}

func (s *queryPolicyIdsSuite) TestQueryPolicyIds_LargeNumberOfPolicies_JSON() {
	ctx, k, accKeep := setupKeeper(s.T())

	creator := accKeep.FirstAcc().GetAddress().String()

	policyNames := []string{}
	for i := 0; i < 10_000; i++ {
		policyNames = append(policyNames, "Policy"+strconv.Itoa(i))
	}
	policyIds := s.setupPolicies(s.T(), ctx, k, creator, policyNames, coretypes.PolicyMarshalingType_SHORT_JSON)

	resp, err := k.PolicyIds(ctx, &types.QueryPolicyIdsRequest{})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), resp)
	require.ElementsMatch(s.T(), policyIds, resp.Ids)
}

func (s *queryPolicyIdsSuite) TestQueryPolicyIds_LargeNumberOfPolicies_YAML() {
	ctx, k, accKeep := setupKeeper(s.T())

	creator := accKeep.FirstAcc().GetAddress().String()

	names := []string{}
	for i := 0; i < 10_000; i++ {
		names = append(names, "Policy"+strconv.Itoa(i))
	}
	policyIds := s.setupPolicies(s.T(), ctx, k, creator, names, coretypes.PolicyMarshalingType_SHORT_YAML)

	resp, err := k.PolicyIds(ctx, &types.QueryPolicyIdsRequest{})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), resp)
	require.ElementsMatch(s.T(), policyIds, resp.Ids)
}
