package relationship

import (
	"testing"

	"github.com/sourcenetwork/acp_core/pkg/errors"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	test "github.com/sourcenetwork/sourcehub/tests/integration/acp"

	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

var setPolicy string = `
name: policy
resources:
  file:
    relations:
      owner:
        types:
          - actor
      admin:
        manages:
          - reader
        types:
          - actor
      reader:
        types:
          - actor
`

func setupSetRel(t *testing.T) *test.TestCtx {
	ctx := test.NewTestCtx(t)

	action := test.PolicySetupAction{
		Policy:        setPolicy,
		PolicyCreator: ctx.TxSigner,
		ObjectsPerActor: map[string][]*coretypes.Object{
			"alice": {
				coretypes.NewObject("file", "foo"),
			},
		},
	}
	action.Run(ctx)
	return ctx
}

func TestSetRelationship_OwnerCanShareObjectTheyOwn(t *testing.T) {
	ctx := setupSetRel(t)
	defer ctx.Cleanup()

	bob := ctx.GetActor("bob").DID
	a1 := test.SetRelationshipAction{
		PolicyId:     ctx.State.PolicyId,
		Relationship: coretypes.NewActorRelationship("file", "foo", "reader", bob),
		Actor:        ctx.GetActor("alice"),
	}
	a1.Run(ctx)
}
func TestSetRelationship_ActorCannotSetRelationshipForUnregisteredObject(t *testing.T) {
	ctx := setupSetRel(t)
	defer ctx.Cleanup()

	bob := ctx.GetActor("bob").DID
	a1 := test.SetRelationshipAction{
		PolicyId:     ctx.State.PolicyId,
		Relationship: coretypes.NewActorRelationship("file", "404-file-not-registered", "reader", bob),
		Actor:        ctx.GetActor("alice"),
		ExpectedErr:  errors.ErrorType_NOT_FOUND,
	}
	a1.Run(ctx)
}

func TestSetRelationship_ActorCannotSetRelationshipForObjectTheyDoNotOwn(t *testing.T) {
	// Given Alice as the Owner of File Foo
	ctx := setupSetRel(t)
	defer ctx.Cleanup()

	bob := ctx.GetActor("bob")
	a1 := test.SetRelationshipAction{
		PolicyId:     ctx.State.PolicyId,
		Relationship: coretypes.NewActorRelationship("file", "foo", "reader", bob.DID),
		Actor:        bob,
		ExpectedErr:  errors.ErrorType_UNAUTHORIZED,
	}
	a1.Run(ctx)
}

func TestSetRelationship_ManagerActorCanDelegateAccessToAnotherActor(t *testing.T) {
	ctx := setupSetRel(t)
	defer ctx.Cleanup()

	// Given object foo and Bob as a manager
	bob := ctx.GetActor("bob").DID
	a1 := test.SetRelationshipAction{
		PolicyId:     ctx.State.PolicyId,
		Relationship: coretypes.NewActorRelationship("file", "foo", "admin", bob),
		Actor:        ctx.GetActor("alice"),
	}
	a1.Run(ctx)

	// when bob shares foo with charlie
	charlie := ctx.GetActor("charlie").DID
	action := test.SetRelationshipAction{
		PolicyId:     ctx.State.PolicyId,
		Relationship: coretypes.NewActorRelationship("file", "foo", "reader", charlie),
		Actor:        ctx.GetActor("bob"),
		Expected: &types.SetRelationshipCmdResult{
			RecordExisted: false,
			Record: &types.RelationshipRecord{
				PolicyId:     ctx.State.PolicyId,
				Relationship: coretypes.NewActorRelationship("file", "foo", "reader", charlie),
				Archived:     false,
			},
		},
	}
	action.Run(ctx)
}

func TestSetRelationship_ManagerActorCannotSetRelationshipToRelationshipsTheyDoNotManage(t *testing.T) {
	ctx := setupSetRel(t)
	defer ctx.Cleanup()

	// Given object foo and Bob as a admin
	bob := ctx.GetActor("bob")
	a1 := test.SetRelationshipAction{
		PolicyId:     ctx.State.PolicyId,
		Relationship: coretypes.NewActorRelationship("file", "foo", "admin", bob.DID),
		Actor:        ctx.GetActor("alice"),
	}
	a1.Run(ctx)

	// when bob attemps to make charlie an admin
	// then operation is not authorized
	charlie := ctx.GetActor("charlie").DID
	action := test.SetRelationshipAction{
		PolicyId:     ctx.State.PolicyId,
		Relationship: coretypes.NewActorRelationship("file", "foo", "admin", charlie),
		Actor:        ctx.GetActor("bob"),
		ExpectedErr:  errors.ErrorType_UNAUTHORIZED,
	}
	action.Run(ctx)
}

func TestSetRelationship_AdminIsNotAllowedToSetAnOwnerRelationship(t *testing.T) {
	ctx := setupSetRel(t)
	defer ctx.Cleanup()

	// Given object foo and Bob as a admin
	bob := ctx.GetActor("bob").DID
	a1 := test.SetRelationshipAction{
		PolicyId:     ctx.State.PolicyId,
		Relationship: coretypes.NewActorRelationship("file", "foo", "admin", bob),
		Actor:        ctx.GetActor("alice"),
	}
	a1.Run(ctx)

	// when bob attemps to make himself an owner
	// then operation is not authorized
	action := test.SetRelationshipAction{
		PolicyId:     ctx.State.PolicyId,
		Relationship: coretypes.NewActorRelationship("file", "foo", "owner", bob),
		Actor:        ctx.GetActor("bob"),
		ExpectedErr:  errors.ErrorType_OPERATION_FORBIDDEN,
	}
	action.Run(ctx)
}
