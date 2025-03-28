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
name: policy
description: ok
resources:
  file:
    relations:
      reader:
      writer:
    permissions:
      read:
        expr: reader + writer
      write:
        expr: writer
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
			&coretypes.Resource{
				Name: "file",
				Relations: []*coretypes.Relation{
					{
						Name: "collaborator",
					},
					{
						Name: "owner",
						Doc:  "owner relations represents the object owner",
						VrTypes: []*coretypes.Restriction{
							{
								ResourceName: "actor",
								RelationName: "",
							},
						},
					},
					{
						Name: "writer",
					},
				},
				Permissions: []*coretypes.Permission{
					{
						Name:       "_can_manage_collaborator",
						Expression: "owner",
						Doc:        "permission controls actors which are allowed to create relationships for the collaborator relation (permission was auto-generated).",
					},
					{
						Name:       "_can_manage_owner",
						Expression: "owner",
						Doc:        "permission controls actors which are allowed to create relationships for the owner relation (permission was auto-generated).",
					},
					{
						Name:       "_can_manage_writer",
						Expression: "owner",
						Doc:        "permission controls actors which are allowed to create relationships for the writer relation (permission was auto-generated).",
					},
					{
						Name:       "read",
						Expression: "(owner + collaborator)",
					},
					{
						Name:       "write",
						Expression: "(owner + (collaborator + writer))",
					},
				},
			},
		},
		ActorResource: &coretypes.ActorResource{
			Name: "actor",
			Doc:  "",
		},
	}
	a := test.EditPolicyAction{
		Id:      ctx.State.PolicyId,
		Creator: ctx.TxSigner,
		Policy: `
name: new policy
description: new ok
resources:
  file:
    relations:
      writer:
      collaborator:
    permissions:
      read:
        expr: collaborator
      write:
        expr: collaborator + writer
`,
		Expected: want,
	}
	a.Run(ctx)
}
