package policy

import (
	"testing"

	"github.com/sourcenetwork/acp_core/pkg/errors"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/stretchr/testify/require"

	test "github.com/sourcenetwork/sourcehub/tests/integration/acp"
)

func TestCreatePolicy_ValidPolicyIsCreated(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	policyStr := `
name: policy
description: ok
resources:
  file:
    relations:
      owner:
        doc: owner owns
        types:
          - actor-resource
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
  name: actor-resource
  doc: my actor
`
	want := &coretypes.Policy{
		Id:                "da7be65027664708551f97197ba5f5993aa99bc7b57055df9766426dc6da9605",
		Name:              "policy",
		Description:       "ok",
		SpecificationType: coretypes.PolicySpecificationType_NO_SPEC,
		Resources: []*coretypes.Resource{
			&coretypes.Resource{
				Name: "file",
				Relations: []*coretypes.Relation{
					&coretypes.Relation{
						Name: "admin",
						Manages: []string{
							"reader",
						},
						VrTypes: []*coretypes.Restriction{},
					},
					&coretypes.Relation{
						Name: "owner",
						Doc:  "owner owns",
						VrTypes: []*coretypes.Restriction{
							{
								ResourceName: "actor-resource",
								RelationName: "",
							},
						},
					},
					&coretypes.Relation{
						Name: "reader",
					},
				},
				Permissions: []*coretypes.Permission{
					{
						Name:       "_can_manage_admin",
						Expression: "owner",
						Doc:        "permission controls actors which are allowed to create relationships for the admin relation (permission was auto-generated).",
					},
					{
						Name:       "_can_manage_owner",
						Expression: "owner",
						Doc:        "permission controls actors which are allowed to create relationships for the owner relation (permission was auto-generated).",
					},
					{
						Name:       "_can_manage_reader",
						Expression: "(admin + owner)",
						Doc:        "permission controls actors which are allowed to create relationships for the reader relation (permission was auto-generated).",
					},
					{
						Name:       "own",
						Expression: "owner",
						Doc:        "own doc",
					},
					{
						Name:       "read",
						Expression: "(owner + reader)",
					},
				},
			},
		},
		ActorResource: &coretypes.ActorResource{
			Name: "actor-resource",
			Doc:  "my actor",
		},
	}

	action := test.CreatePolicyAction{
		Policy:   policyStr,
		Expected: want,
		Creator:  ctx.TxSigner,
	}
	action.Run(ctx)

	event := &coretypes.EventPolicyCreated{
		PolicyId:   "4419a8abb886c641bc794b9b3289bc2118ab177542129627b6b05d540de03e46",
		PolicyName: "policy",
	}
	_ = event
	//.AssertEventEmmited(t, ctx, event)
}

func TestCreatePolicy_PolicyResources_OwnerRelationImplicitlyAdded(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	action := test.CreatePolicyAction{
		Policy: `
name: policy
description: ok
resources:
  file:
    relations:
      reader:
    permissions:
  foo:
    relations:
      owner:
    permissions:
`,
		Creator: ctx.TxSigner,
	}
	pol := action.Run(ctx)
	require.Equal(t, "owner", pol.GetResourceByName("file").GetRelationByName("owner").Name)
}

func TestCreatePolicy_ManagementReferencingUndefinedRelationReturnsError(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	action := test.CreatePolicyAction{
		Policy: `
name: policy
description: ok
resources:
  file:
    relations:
      owner:
      admin:
        manages:
          - deleter
    permissions:
`,
		Creator: ctx.TxSigner,
		//ExpectedErr: coretypes.ErrInvalidManagementRule, // FIXME
		ExpectedErr: errors.ErrorType_BAD_INPUT,
	}
	action.Run(ctx)
}
