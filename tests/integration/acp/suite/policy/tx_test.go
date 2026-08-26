package policy

import (
	"testing"

	"github.com/sourcenetwork/acp_core/pkg/errors"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/stretchr/testify/require"

	test "github.com/sourcenetwork/vera/tests/integration/acp"
)

func TestCreatePolicy_ValidPolicyIsCreated(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	policyStr := `
description: ok
name: policy
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
`

	want := &coretypes.Policy{
		Id:                "199091661bdd06221eb0a8070673c76f25ca8c8dcc04d47934f0abb123daf78b",
		Name:              "policy",
		Description:       "ok",
		SpecificationType: coretypes.PolicySpecificationType_NO_SPEC,
		Resources: []*coretypes.Resource{
			{
				Name: "file",
				Owner: &coretypes.Relation{
					Name: "owner",
					Doc:  "owner relations represents the object owner",
					VrTypes: []*coretypes.Restriction{
						{
							ResourceName: "actor",
						},
					},
					Manages: []string{
						"admin",
						"reader",
						"owner",
					},
				},
				Relations: []*coretypes.Relation{
					{
						Name: "admin",
						Manages: []string{
							"reader",
						},
						VrTypes: []*coretypes.Restriction{},
					},
					{
						Name:    "reader",
						VrTypes: []*coretypes.Restriction{},
					},
				},
				Permissions: []*coretypes.Permission{
					{
						Name:                "own",
						Expression:          "",
						Doc:                 "own doc",
						EffectiveExpression: "owner",
					},
					{
						Name:                "read",
						Expression:          "reader",
						EffectiveExpression: "(owner + reader)",
					},
				},
				ManagementRules: []*coretypes.ManagementRule{
					{
						Relation:   "admin",
						Expression: "owner",
						Managers: []string{
							"owner",
						},
					},
					{
						Relation:   "owner",
						Expression: "owner",
						Managers: []string{
							"owner",
						},
					},
					{
						Relation:   "reader",
						Expression: "(admin + owner)",
						Managers: []string{
							"admin", "owner",
						},
					},
				},
			},
		},
		ActorResource: &coretypes.ActorResource{
			Name:      "actor",
			Doc:       "actor resource models the set of actors defined within a policy",
			Relations: nil,
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
`,

		Creator: ctx.TxSigner,
		//ExpectedErr: coretypes.ErrInvalidManagementRule, // FIXME
		ExpectedErr: errors.ErrorType_BAD_INPUT,
	}
	action.Run(ctx)
}
