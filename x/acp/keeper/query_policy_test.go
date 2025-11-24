package keeper

import (
	"context"
	"testing"

	"github.com/sourcenetwork/acp_core/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

type queryPolicySuite struct {
	suite.Suite
}

func TestQueryPolicy(t *testing.T) {
	suite.Run(t, &queryPolicySuite{})
}

func (s *queryPolicySuite) setupPolicy(t *testing.T) (context.Context, Keeper, string) {
	policyStr := `
description: A valid policy
name: Source Policy
resources:
- name: file
  permissions:
  - expr: owner
    name: read
  - expr: owner
    name: write
  relations:
  - name: owner
    types:
    - actor
  - name: rm-root
`

	ctx, k, accKeep := setupKeeper(t)
	creator := accKeep.FirstAcc().GetAddress().String()

	msg := types.MsgCreatePolicy{
		Creator:     creator,
		Policy:      policyStr,
		MarshalType: coretypes.PolicyMarshalingType_YAML,
	}

	resp, err := k.CreatePolicy(ctx, &msg)
	require.NoError(t, err)

	return ctx, k, resp.Record.Policy.Id
}

func (s *queryPolicySuite) TestQueryPolicy_Success() {
	ctx, k, policyID := s.setupPolicy(s.T())

	req := types.QueryPolicyRequest{
		Id: policyID,
	}

	resp, err := k.Policy(ctx, &req)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), resp)
	require.Equal(s.T(), "Source Policy", resp.Record.Policy.Name)
	require.Equal(s.T(), "A valid policy", resp.Record.Policy.Description)
}

func (s *queryPolicySuite) TestQueryPolicy_UnknownPolicyReturnsPolicyNotFoundErr() {
	ctx, k, _ := setupKeeper(s.T())

	req := types.QueryPolicyRequest{
		Id: "not found",
	}

	resp, err := k.Policy(ctx, &req)
	require.Nil(s.T(), resp)
	require.ErrorIs(s.T(), err, errors.ErrorType_NOT_FOUND)
	require.Contains(s.T(), err.Error(), "policy not found")
}

func (s *queryPolicySuite) TestQueryPolicy_NilRequestReturnsInvalidRequestErr() {
	ctx, k, _ := setupKeeper(s.T())

	resp, err := k.Policy(ctx, nil)
	require.Nil(s.T(), resp)
	require.Error(s.T(), err)
	require.Contains(s.T(), err.Error(), "invalid request")
}
