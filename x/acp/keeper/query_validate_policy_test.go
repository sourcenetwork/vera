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

	req := &types.QueryValidatePolicyRequest{
		Policy: `
description: A valid policy
name: Source Policy
resources:
- name: file
  permissions:
  - name: read
  - name: write
`,

		MarshalType: coretypes.PolicyMarshalingType_YAML,
	}

	result, err := k.ValidatePolicy(ctx, req)

	want := &types.QueryValidatePolicyResponse{
		Valid:    true,
		ErrorMsg: "",
		Policy: &coretypes.Policy{
			Id:          "",
			Name:        "Source Policy",
			Description: "A valid policy",
			ActorResource: &coretypes.ActorResource{
				Name: "actor",
				Doc:  "actor resource models the set of actors defined within a policy",
			},
			Attributes:        nil,
			SpecificationType: 0,
			Resources: []*coretypes.Resource{
				{
					Name: "file",
					Permissions: []*coretypes.Permission{
						{
							Name:                "read",
							Expression:          "",
							EffectiveExpression: "owner",
						},
						{
							Name:                "write",
							Expression:          "",
							EffectiveExpression: "owner",
						},
					},
					Relations: []*coretypes.Relation{},
					ManagementRules: []*coretypes.ManagementRule{
						{
							Relation:   "owner",
							Expression: "owner",
							Managers: []string{
								"owner",
							},
						},
					},
					Owner: &coretypes.Relation{
						Name:    "owner",
						Doc:     "owner relations represents the object owner",
						Manages: []string{"owner"},
						VrTypes: []*coretypes.Restriction{
							{
								ResourceName: "actor",
							},
						},
					},
				},
			},
		},
	}
	require.Equal(s.T(), want, result)
	require.Nil(s.T(), err)
}

func (s *queryValidatePolicySuite) TestValidatePolicy_ComplexValidPolicy() {
	ctx, k, _ := setupKeeper(s.T())

	req := &types.QueryValidatePolicyRequest{
		Policy: `
description: Another valid policy
name: Source Policy
resources:
- name: file
  permissions:
  - doc: own doc
    name: own
  - expr: reader
    name: read
  relations:
  - manages:
    - reader
    name: admin
  - name: reader
`,

		MarshalType: coretypes.PolicyMarshalingType_YAML,
	}

	result, err := k.ValidatePolicy(ctx, req)
	require.True(s.T(), result.Valid)
	require.Nil(s.T(), err)
}

func (s *queryValidatePolicySuite) TestValidatePolicy_InvalidSyntax() {
	ctx, k, _ := setupKeeper(s.T())

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
		MarshalType: coretypes.PolicyMarshalingType_YAML,
	}

	result, err := k.ValidatePolicy(ctx, req)

	require.NotNil(s.T(), result)
	require.False(s.T(), result.Valid)
	require.Contains(s.T(), result.ErrorMsg, "mapping values are not allowed in this context")
	require.Nil(s.T(), err)
}

func (s *queryValidatePolicySuite) TestValidatePolicy_EmptyPolicy() {
	ctx, k, _ := setupKeeper(s.T())

	req := &types.QueryValidatePolicyRequest{
		Policy:      "",
		MarshalType: coretypes.PolicyMarshalingType_YAML,
	}

	result, err := k.ValidatePolicy(ctx, req)

	require.False(s.T(), result.Valid)
	require.Contains(s.T(), result.ErrorMsg, "name required")
	require.Nil(s.T(), err)
}
