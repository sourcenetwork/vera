package relationship

import (
	"testing"

	"github.com/sourcenetwork/acp_core/pkg/errors"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"

	test "github.com/sourcenetwork/sourcehub/tests/integration/acp"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

var deletePolicy = `
name: policy
resources:
- name: file
  relations:
  - manages:
    - reader
    name: admin
    types:
    - actor
  - name: reader
    types:
    - actor
  - name: writer
    types:
    - actor
`

func setupDelete(t *testing.T) *test.TestCtx {
	ctx := test.NewTestCtx(t)

	reader := ctx.GetActor("reader")
	writer := ctx.GetActor("writer")
	admin := ctx.GetActor("admin")
	action := test.PolicySetupAction{
		Policy:        deletePolicy,
		PolicyCreator: ctx.TxSigner,
		ObjectsPerActor: map[string][]*coretypes.Object{
			"alice": []*coretypes.Object{
				coretypes.NewObject("file", "foo"),
			},
		},
		RelationshipsPerActor: map[string][]*coretypes.Relationship{
			"alice": []*coretypes.Relationship{
				coretypes.NewActorRelationship("file", "foo", "reader", reader.DID),
				coretypes.NewActorRelationship("file", "foo", "writer", writer.DID),
				coretypes.NewActorRelationship("file", "foo", "admin", admin.DID),
			},
		},
	}
	action.Run(ctx)

	return ctx
}

func TestDeleteRelationship_ObjectOwnerCanRemoveRelationship(t *testing.T) {
	ctx := setupDelete(t)
	defer ctx.Cleanup()

	action := test.DeleteRelationshipAction{
		PolicyId:     ctx.State.PolicyId,
		Relationship: coretypes.NewActorRelationship("file", "foo", "reader", ctx.GetActor("reader").DID),
		Actor:        ctx.GetActor("alice"),
		Expected: &types.DeleteRelationshipCmdResult{
			RecordFound: true,
		},
	}
	action.Run(ctx)
}

func TestDeleteRelationship_ObjectManagerCanRemoveRelationshipsForRelationTheyManage(t *testing.T) {
	ctx := setupDelete(t)
	defer ctx.Cleanup()

	action := test.DeleteRelationshipAction{
		PolicyId:     ctx.State.PolicyId,
		Relationship: coretypes.NewActorRelationship("file", "foo", "reader", ctx.GetActor("reader").DID),
		Actor:        ctx.GetActor("admin"),
		Expected: &types.DeleteRelationshipCmdResult{
			RecordFound: true,
		},
	}
	action.Run(ctx)
}

func TestDeleteRelationship_ObjectManagerCannotRemoveRelationshipForRelationTheyDontManage(t *testing.T) {
	ctx := setupDelete(t)
	defer ctx.Cleanup()

	action := test.DeleteRelationshipAction{
		PolicyId:     ctx.State.PolicyId,
		Relationship: coretypes.NewActorRelationship("file", "foo", "writer", ctx.GetActor("writer").DID),
		Actor:        ctx.GetActor("admin"),
		ExpectedErr:  errors.ErrorType_UNAUTHORIZED,
	}
	action.Run(ctx)
}
