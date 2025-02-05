package object

import (
	stderrors "errors"
	"testing"

	"github.com/sourcenetwork/acp_core/pkg/errors"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"

	test "github.com/sourcenetwork/sourcehub/tests/integration/acp"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

var policyDef string = `
name: policy
resources:
  resource:
    relations:
      owner:
        types:
          - actor
`

func TestRegisterObject_RegisteringNewObjectIsSucessful(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	a1 := test.CreatePolicyAction{
		Policy:  policyDef,
		Creator: ctx.TxSigner,
	}
	pol := a1.Run(ctx)

	bob := ctx.GetActor("bob")
	a2 := test.RegisterObjectAction{
		PolicyId: pol.Id,
		Object:   coretypes.NewObject("resource", "foo"),
		Actor:    bob,
		Expected: &types.RelationshipRecord{
			PolicyId:     ctx.State.PolicyId,
			Relationship: coretypes.NewActorRelationship("resource", "foo", "owner", bob.DID),
			Archived:     false,
		},
	}
	a2.Run(ctx)

	/*
		event := &types.EventObjectRegistered{
			Actor:          did,
			PolicyId:       pol.Id,
			ObjectId:       "foo",
			ObjectResource: "resource",
		}
		testutil.AssertEventEmmited(t, ctx, event)
	*/
}

func TestRegisterObject_RegisteringObjectRegisteredToAnotherUserErrors(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	// Given Bob as owner of foo
	a1 := test.CreatePolicyAction{
		Policy:  policyDef,
		Creator: ctx.TxSigner,
	}
	pol := a1.Run(ctx)

	a2 := test.RegisterObjectAction{
		PolicyId: pol.Id,
		Object:   coretypes.NewObject("resource", "foo"),
		Actor:    ctx.GetActor("bob"),
	}
	a2.Run(ctx)

	// when Alice tries to register foo
	// then she is denied
	a2 = test.RegisterObjectAction{
		PolicyId:    pol.Id,
		Object:      coretypes.NewObject("resource", "foo"),
		Actor:       ctx.GetActor("alice"),
		ExpectedErr: errors.ErrorType_OPERATION_FORBIDDEN,
	}
	a2.Run(ctx)
}

func TestRegisterObject_ReregisteringObjectOwnedByUser_Errors(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	// Given Bob as owner of foo
	a1 := test.CreatePolicyAction{
		Policy:  policyDef,
		Creator: ctx.TxSigner,
	}
	pol := a1.Run(ctx)

	a2 := test.RegisterObjectAction{
		PolicyId: pol.Id,
		Object:   coretypes.NewObject("resource", "foo"),
		Actor:    ctx.GetActor("bob"),
	}
	a2.Run(ctx)

	// when Bob tries to reregister foo
	// then forbidden op
	a2 = test.RegisterObjectAction{
		PolicyId:    pol.Id,
		Object:      coretypes.NewObject("resource", "foo"),
		Actor:       ctx.GetActor("bob"),
		ExpectedErr: errors.ErrorType_OPERATION_FORBIDDEN,
	}
	a2.Run(ctx)
}

func TestRegisterObject_RegisteringAnotherUsersArchivedObject_Errors(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	// Given Bob as previous owner of foo
	a1 := test.CreatePolicyAction{
		Policy:  policyDef,
		Creator: ctx.TxSigner,
	}
	pol := a1.Run(ctx)

	a2 := test.RegisterObjectAction{
		PolicyId: pol.Id,
		Object:   coretypes.NewObject("resource", "foo"),
		Actor:    ctx.GetActor("bob"),
	}
	a2.Run(ctx)

	a3 := test.ArchiveObjectAction{
		PolicyId: pol.Id,
		Actor:    ctx.GetActor("bob"),
		Object:   coretypes.NewObject("resource", "foo"),
	}
	a3.Run(ctx)

	// when Alice tries to register foo op forbidden
	action := test.RegisterObjectAction{
		PolicyId:    pol.Id,
		Object:      coretypes.NewObject("resource", "foo"),
		Actor:       ctx.GetActor("alice"),
		ExpectedErr: errors.ErrorType_OPERATION_FORBIDDEN,
	}
	action.Run(ctx)
}

func TestRegisterObject_RegisteringArchivedUserObject_ReturnsOperationForbidden(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	// Given Bob as previous owner of foo
	a1 := test.CreatePolicyAction{
		Policy:  policyDef,
		Creator: ctx.TxSigner,
	}
	pol := a1.Run(ctx)

	a2 := test.RegisterObjectAction{
		PolicyId: pol.Id,
		Object:   coretypes.NewObject("resource", "foo"),
		Actor:    ctx.GetActor("bob"),
	}
	a2.Run(ctx)
	a3 := test.ArchiveObjectAction{
		PolicyId: pol.Id,
		Actor:    ctx.GetActor("bob"),
		Object:   coretypes.NewObject("resource", "foo"),
	}
	a3.Run(ctx)

	// when Bob tries to register the archived foo
	action := test.RegisterObjectAction{
		PolicyId:    pol.Id,
		Object:      coretypes.NewObject("resource", "foo"),
		Actor:       ctx.GetActor("bob"),
		ExpectedErr: errors.ErrorType_OPERATION_FORBIDDEN,
	}
	action.Run(ctx)

	/*
		event := &types.EventObjectRegistered{
			Actor:          bobDID,
			PolicyId:       pol.Id,
			ObjectId:       "foo",
			ObjectResource: "resource",
		}
		testutil.AssertEventEmmited(t, ctx, event)
	*/
}

func TestRegisterObject_RegisteringObjectInAnUndefinedResourceErrors(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	// Given Bob as previous owner of foo
	a1 := test.CreatePolicyAction{
		Policy:  policyDef,
		Creator: ctx.TxSigner,
	}
	pol := a1.Run(ctx)

	a2 := test.RegisterObjectAction{
		PolicyId:    pol.Id,
		Object:      coretypes.NewObject("abc", "foo"),
		Actor:       ctx.GetActor("bob"),
		ExpectedErr: stderrors.New("resource not found"), // FIXME update once zanzi errors are sorted
	}
	a2.Run(ctx)
}

func TestRegisterObject_RegisteringToUnknownPolicyReturnsError(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	a2 := test.RegisterObjectAction{
		PolicyId:    "abc1234",
		Object:      coretypes.NewObject("resource", "foo"),
		Actor:       ctx.GetActor("bob"),
		ExpectedErr: errors.ErrorType_NOT_FOUND,
	}
	a2.Run(ctx)
}

func TestRegisterObject_BlankResourceErrors(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	a1 := test.CreatePolicyAction{
		Policy:  policyDef,
		Creator: ctx.TxSigner,
	}
	pol := a1.Run(ctx)

	a2 := test.RegisterObjectAction{
		PolicyId:    pol.Id,
		Object:      coretypes.NewObject("abc", "foo"),
		Actor:       ctx.GetActor("bob"),
		ExpectedErr: stderrors.New("resource not found"), //FIXME once zanzi errors are sorted, change this to the correct type
	}
	a2.Run(ctx)
}

func TestRegisterObject_BlankObjectIdErrors(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	a1 := test.CreatePolicyAction{
		Policy:  policyDef,
		Creator: ctx.TxSigner,
	}
	pol := a1.Run(ctx)

	a2 := test.RegisterObjectAction{
		PolicyId:    pol.Id,
		Object:      coretypes.NewObject("resource", ""),
		Actor:       ctx.GetActor("bob"),
		ExpectedErr: errors.ErrorType_BAD_INPUT,
	}
	a2.Run(ctx)
}
