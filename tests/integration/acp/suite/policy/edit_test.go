package policy

import (
	"testing"

	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	test "github.com/sourcenetwork/sourcehub/tests/integration/acp"
)

func TestEditPolicy_CanEditPolicy(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	action := test.CreatePolicyAction{
		Policy: `
description: ok
name: policy
resources:
- name: file
  permissions:
  - expr: (reader + writer)
    name: read
  - expr: writer
    name: write
  relations:
  - name: reader
  - name: writer
`,
		Creator: ctx.TxSigner,
	}
	action.Run(ctx)

	want := &coretypes.Policy{
		Id:                ctx.State.PolicyId,
		Name:              "new policy",
		Description:       "new ok",
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
							RelationName: "",
						},
					},
					Manages: []string{
						"collaborator",
						"writer",
						"owner",
					},
				},
				Relations: []*coretypes.Relation{
					{
						Name: "collaborator",
					},
					{
						Name: "writer",
					},
				},
				Permissions: []*coretypes.Permission{
					{
						Name:                "read",
						Expression:          "collaborator",
						EffectiveExpression: "(owner + collaborator)",
					},
					{
						Name:                "write",
						Expression:          "(collaborator + writer)",
						EffectiveExpression: "(owner + (collaborator + writer))",
					},
				},
				ManagementRules: []*coretypes.ManagementRule{
					{
						Relation:   "collaborator",
						Expression: "owner",
						Managers:   []string{"owner"},
					},
					{
						Relation:   "owner",
						Expression: "owner",
						Managers:   []string{"owner"},
					},
					{
						Relation:   "writer",
						Expression: "owner",
						Managers:   []string{"owner"},
					},
				},
			},
		},
		ActorResource: &coretypes.ActorResource{
			Name: "actor",
			Doc:  "actor resource models the set of actors defined within a policy",
		},
	}
	a := test.EditPolicyAction{
		Id:      ctx.State.PolicyId,
		Creator: ctx.TxSigner,
		Policy: `
description: new ok
name: new policy
resources:
- name: file
  permissions:
  - expr: collaborator
    name: read
  - expr: (collaborator + writer)
    name: write
  relations:
  - name: collaborator
  - name: writer
`,
		Expected: want,
	}
	a.Run(ctx)
}
