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
actor:
  doc: my actor
  name: actor-resource
description: ok
name: policy
resources:
- name: file
  permissions:
  - doc: own doc
    expr: owner
    name: own
  - expr: owner + reader
    name: read
  relations:
  - manages:
    - reader
    name: admin
  - doc: owner owns
    name: owner
    types:
    - actor-resource
  - name: reader
`
	want := &coretypes.Policy{
		Id:                "ba5162bd61996b6fb6e66ef85449f0de2e89584743df7f71577674cfb531eb25",
		Name:              "policy",
		Description:       "ok",
		SpecificationType: coretypes.PolicySpecificationType_NO_SPEC,
		Resources: []*coretypes.Resource{
			{
				Name: "file",
				Relations: []*coretypes.Relation{
					{
						Name: "admin",
						Manages: []string{
							"reader",
						},
						VrTypes: []*coretypes.Restriction{},
					},
					{
						Name: "owner",
						Doc:  "owner owns",
						VrTypes: []*coretypes.Restriction{
							{
								ResourceName: "actor-resource",
								RelationName: "",
							},
						},
					},
					{
						Name:    "reader",
						VrTypes: []*coretypes.Restriction{},
					},
				},
				Permissions: []*coretypes.Permission{
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
				ManagementRules: []*coretypes.ManagementRule{
					{
						Relation:   "admin",
						Expression: "owner",
					},
					{
						Relation:   "owner",
						Expression: "owner",
					},
					{
						Relation:   "reader",
						Expression: "(admin + owner)",
					},
				},
			},
		},
		ActorResource: &coretypes.ActorResource{
			Name:      "actor-resource",
			Doc:       "my actor",
			Relations: []*coretypes.Relation{},
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
description: ok
name: policy
resources:
- name: file
  relations:
  - name: reader
- name: foo
  relations:
  - name: owner
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
description: ok
name: policy
resources:
- name: file
  relations:
  - manages:
    - deleter
    name: admin
  - name: owner
`,

		Creator: ctx.TxSigner,
		//ExpectedErr: coretypes.ErrInvalidManagementRule, // FIXME
		ExpectedErr: errors.ErrorType_BAD_INPUT,
	}
	action.Run(ctx)
}
