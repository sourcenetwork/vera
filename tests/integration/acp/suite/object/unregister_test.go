package object

import (
	"testing"

	"github.com/sourcenetwork/acp_core/pkg/errors"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"

	test "github.com/sourcenetwork/sourcehub/tests/integration/acp"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

var unregisterTestPol = `
name: policy
resources:
  file:
    relations:
      owner:
        types:
          - actor
      reader:
        types:
          - actor
`

func setupUnregister(t *testing.T) *test.TestCtx {
	ctx := test.NewTestCtx(t)
	a1 := test.CreatePolicyAction{
		Creator: ctx.TxSigner,
		Policy:  unregisterTestPol,
	}
	a1.Run(ctx)
	a2 := test.RegisterObjectAction{
		PolicyId: ctx.State.PolicyId,
		Object:   coretypes.NewObject("file", "foo"),
		Actor:    ctx.GetActor("alice"),
	}
	a2.Run(ctx)
	a3 := test.SetRelationshipAction{
		Actor:        ctx.GetActor("alice"),
		PolicyId:     ctx.State.PolicyId,
		Relationship: coretypes.NewActorRelationship("file", "foo", "reader", ctx.GetActor("alice").DID),
	}
	a3.Run(ctx)
	return ctx
}

func TestUnregisterObject_RegisteredObjectCanBeUnregisteredByAuthor(t *testing.T) {
	ctx := setupUnregister(t)
	defer ctx.Cleanup()

	action := test.ArchiveObjectAction{
		PolicyId: ctx.State.PolicyId,
		Object:   coretypes.NewObject("file", "foo"),
		Actor:    ctx.GetActor("alice"),
		Expected: &types.ArchiveObjectCmdResult{
			Found:                true,
			RelationshipsRemoved: 2,
		},
	}
	action.Run(ctx)
}

func TestUnregisterObject_ActorCannotUnregisterObjectTheyDoNotOwn(t *testing.T) {
	ctx := setupUnregister(t)
	defer ctx.Cleanup()

	action := test.ArchiveObjectAction{
		PolicyId:    ctx.State.PolicyId,
		Object:      coretypes.NewObject("file", "foo"),
		Actor:       ctx.GetActor("bob"),
		ExpectedErr: errors.ErrorType_UNAUTHORIZED,
	}
	action.Run(ctx)
}

func TestUnregisterObject_UnregisteringAnObjectThatDoesNotExistReturnsFoundFalse(t *testing.T) {
	ctx := setupUnregister(t)
	defer ctx.Cleanup()

	action := test.ArchiveObjectAction{
		PolicyId:    ctx.State.PolicyId,
		Object:      coretypes.NewObject("file", "file-isnt-registerd"),
		Actor:       ctx.GetActor("bob"),
		ExpectedErr: errors.ErrorType_BAD_INPUT,
	}
	action.Run(ctx)
}

func TestUnregisterObject_UnregisteringAnAlreadyArchivedObjectIsANoop(t *testing.T) {
	ctx := setupUnregister(t)
	defer ctx.Cleanup()

	action := test.ArchiveObjectAction{
		PolicyId: ctx.State.PolicyId,
		Object:   coretypes.NewObject("file", "foo"),
		Actor:    ctx.GetActor("alice"),
	}
	action.Run(ctx)

	action = test.ArchiveObjectAction{
		PolicyId: ctx.State.PolicyId,
		Object:   coretypes.NewObject("file", "foo"),
		Actor:    ctx.GetActor("alice"),
		Expected: &types.ArchiveObjectCmdResult{
			Found: true,
		},
	}
	action.Run(ctx)
}

func TestUnregisterObject_SendingInvalidPolicyIdErrors(t *testing.T) {
	ctx := setupUnregister(t)
	defer ctx.Cleanup()

	action := test.ArchiveObjectAction{
		PolicyId:    "abc1234",
		Object:      coretypes.NewObject("file", "foo"),
		Actor:       ctx.GetActor("alice"),
		ExpectedErr: errors.ErrorType_NOT_FOUND,
	}
	action.Run(ctx)
}
