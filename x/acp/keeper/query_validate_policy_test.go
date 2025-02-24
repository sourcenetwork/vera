package keeper

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

type queryValidatePolicySuite struct {
	suite.Suite
}

func TestValidatePolicy(t *testing.T) {
	suite.Run(t, &queryValidatePolicySuite{})
}

func (s *queryValidatePolicySuite) TestValidatePolicy_ValidPolicy() {
	ctx, k, _ := setupKeeper(s.T())
	querier := NewQuerier(k)

	req := &types.QueryValidatePolicyRequest{
		Policy: `
name: Source Policy
description: A valid policy
resources:
  file:
    relations: 
      owner:
        types:
          - actor
    permissions: 
      read: 
        expr: owner
      write: 
        expr: owner
actor:
  name: actor
  doc: some actor
`,
		MarshalType: coretypes.PolicyMarshalingType_SHORT_YAML,
	}

	result, err := querier.ValidatePolicy(ctx, req)

	want := &types.QueryValidatePolicyResponse{
		Valid:    true,
		ErrorMsg: "",
	}
	require.Equal(s.T(), want, result)
	require.Nil(s.T(), err)
}

func (s *queryValidatePolicySuite) TestValidatePolicy_ComplexValidPolicy() {
	ctx, k, _ := setupKeeper(s.T())
	querier := NewQuerier(k)

	req := &types.QueryValidatePolicyRequest{
		Policy: `
name: Source Policy
description: Another valid policy
resources:
  file:
    relations:
      owner:
        doc: owner owns
        types:
          - actor-source
      reader:
      admin:
        manages:
          - reader
    permissions:
      own:
        expr: owner
        doc: own doc
      read:
        expr: owner + reader
actor:
  name: actor-source
  doc: my actor
`,
		MarshalType: coretypes.PolicyMarshalingType_SHORT_YAML,
	}

	result, err := querier.ValidatePolicy(ctx, req)

	want := &types.QueryValidatePolicyResponse{
		Valid:    true,
		ErrorMsg: "",
	}
	require.Equal(s.T(), want, result)
	require.Nil(s.T(), err)
}

func (s *queryValidatePolicySuite) TestValidatePolicy_InvalidSyntax() {
	ctx, k, _ := setupKeeper(s.T())
	querier := NewQuerier(k)

	req := &types.QueryValidatePolicyRequest{
		Policy: `
name: Invalid policy
description: Policy with invalid syntax
resources:
  file
    permissions:
      read:
        expr: owner
`,
		MarshalType: coretypes.PolicyMarshalingType_SHORT_YAML,
	}

	result, err := querier.ValidatePolicy(ctx, req)

	require.NotNil(s.T(), result)
	require.False(s.T(), result.Valid)
	require.Contains(s.T(), result.ErrorMsg, "mapping values are not allowed in this context")
	require.Nil(s.T(), err)
}

func (s *queryValidatePolicySuite) TestValidatePolicy_MissingOwner() {
	ctx, k, _ := setupKeeper(s.T())
	querier := NewQuerier(k)

	req := &types.QueryValidatePolicyRequest{
		Policy: `
name: Another invalid policy
description: Policy with missing owner
resources:
  file:
    permissions:
      read:
        expr: owner
`,
		MarshalType: coretypes.PolicyMarshalingType_SHORT_YAML,
	}

	result, err := querier.ValidatePolicy(ctx, req)

	require.NotNil(s.T(), result)
	require.False(s.T(), result.Valid)
	require.Contains(s.T(), result.ErrorMsg, "resource file: resource missing owner relation")
	require.Nil(s.T(), err)
}

func (s *queryValidatePolicySuite) TestValidatePolicy_EmptyPolicy() {
	ctx, k, _ := setupKeeper(s.T())
	querier := NewQuerier(k)

	req := &types.QueryValidatePolicyRequest{
		Policy:      "",
		MarshalType: coretypes.PolicyMarshalingType_SHORT_YAML,
	}

	result, err := querier.ValidatePolicy(ctx, req)

	want := &types.QueryValidatePolicyResponse{
		Valid:    false,
		ErrorMsg: "name is required: policy: code 4: type BAD_INPUT: ctx={[]}",
	}
	require.Equal(s.T(), want, result)
	require.Nil(s.T(), err)
}

func (s *queryValidatePolicySuite) TestValidatePolicy_BadActor() {
	ctx, k, _ := setupKeeper(s.T())
	querier := NewQuerier(k)

	req := &types.QueryValidatePolicyRequest{
		Policy: `
name: Yet another invalid policy
description: Policy with bad actor
resources:
  file:
    relations:
      owner:
        doc: owner owns
        types:
          - actor-source
      reader:
      admin:
        manages:
          - reader
    permissions:
      own:
        expr: owner
        doc: own doc
      read:
        expr: owner + reader
actor:
  name: actor-factor
  doc: bad actor
`,
		MarshalType: coretypes.PolicyMarshalingType_SHORT_YAML,
	}

	result, err := querier.ValidatePolicy(ctx, req)

	require.NotNil(s.T(), result)
	require.False(s.T(), result.Valid)
	require.Contains(s.T(), result.ErrorMsg, "subject restriction: resource file, relation owner: no resource actor-source")
	require.Nil(s.T(), err)
}
