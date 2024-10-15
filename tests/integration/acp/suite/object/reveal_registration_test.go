package object

import (
	"testing"

	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/stretchr/testify/require"

	test "github.com/sourcenetwork/sourcehub/tests/integration/acp"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

const revealPolicy string = `
name: pol
resources:
  file:
    relations:
	  owner:
	    types:
		  - actor
`

func TestRevealRegistration_UnregisteredObjectGetsRegistered_ReturnsNewRecord(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	a1 := test.CreatePolicyAction{
		Policy:  revealPolicy,
		Creator: ctx.GetActor("bob"),
	}
	pol := a1.Run(ctx)
	a2 := test.CommitRegistrationsAction{
		PolicyId: pol.Id,
		Actor:    ctx.GetActor("bob"),
		Objects: []*coretypes.Object{
			coretypes.NewObject("file", "foo.txt"),
			coretypes.NewObject("file", "bar.txt"),
		},
	}
	commitment := a2.Run(ctx)
	ctx.WaitBlock()

	a := test.RevealRegistrationAction{
		Actor:        ctx.GetActor("bob"),
		PolicyId:     pol.Id,
		CommitmentId: commitment.Id,
		Objects: []*coretypes.Object{
			coretypes.NewObject("file", "foo.txt"),
			coretypes.NewObject("file", "bar.txt"),
		},
		Index: 0,
	}
	a.Run(ctx)
}

func TestRevealRegistration_ObjectRegisteredToActor_ReturnOldRecord(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	a1 := test.CreatePolicyAction{
		Policy:  revealPolicy,
		Creator: ctx.GetActor("bob"),
	}
	pol := a1.Run(ctx)
	a2 := test.CommitRegistrationsAction{
		PolicyId: pol.Id,
		Actor:    ctx.GetActor("bob"),
		Objects: []*coretypes.Object{
			coretypes.NewObject("file", "foo.txt"),
			coretypes.NewObject("file", "bar.txt"),
		},
	}
	commitment := a2.Run(ctx)
	ctx.WaitBlock()

	a := test.RevealRegistrationAction{
		Actor:        ctx.GetActor("bob"),
		PolicyId:     pol.Id,
		CommitmentId: commitment.Id,
		Objects: []*coretypes.Object{
			coretypes.NewObject("file", "foo.txt"),
			coretypes.NewObject("file", "bar.txt"),
		},
		Index: 0,
	}
	a.Run(ctx)
	// TODO
}

func TestRevealRegistration_ObjectRegisteredAfterCommitment_RegistrationAmended(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	a1 := test.CreatePolicyAction{
		Policy:  revealPolicy,
		Creator: ctx.GetActor("bob"),
	}
	pol := a1.Run(ctx)
	a2 := test.CommitRegistrationsAction{
		PolicyId: pol.Id,
		Actor:    ctx.GetActor("bob"),
		Objects: []*coretypes.Object{
			coretypes.NewObject("file", "foo.txt"),
			coretypes.NewObject("file", "bar.txt"),
		},
	}
	commitment := a2.Run(ctx)
	ctx.WaitBlock()

	a3 := test.RegisterObjectAction{
		Actor:    ctx.GetActor("alice"),
		PolicyId: pol.Id,
		Object:   coretypes.NewObject("file", "foo.txt"),
	}
	a3.Run(ctx)
	ctx.WaitBlock()

	a := test.RevealRegistrationAction{
		Actor:        ctx.GetActor("bob"),
		PolicyId:     pol.Id,
		CommitmentId: commitment.Id,
		Objects: []*coretypes.Object{
			coretypes.NewObject("file", "foo.txt"),
			coretypes.NewObject("file", "bar.txt"),
		},
		Index: 0,
	}
	result := a.Run(ctx)

	require.Equal(ctx.T, result.Event.Type, types.ObjectRegistrationEventType_AMENDMENT)
	require.Equal(ctx.T, result.Record.OwnerDid, ctx.GetActor("bob").DID)
	require.Equal(ctx.T, result.Record.Relationship, coretypes.NewActorRelationship("file", "foo.txt", "owner", ctx.GetActor("bob").DID))
}

func TestRevealRegistration_ObjectRegisteredThroughNewerCommitment_RegistrationIsAmended(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	// Given Policy
	a1 := test.CreatePolicyAction{
		Policy:  revealPolicy,
		Creator: ctx.GetActor("bob"),
	}
	pol := a1.Run(ctx)
	// Given a commitment made by bob to foo.txt
	a2 := test.CommitRegistrationsAction{
		PolicyId: pol.Id,
		Actor:    ctx.GetActor("bob"),
		Objects: []*coretypes.Object{
			coretypes.NewObject("file", "foo.txt"),
		},
	}
	commitment := a2.Run(ctx)
	ctx.WaitBlock()
	// Given alice registers foo.txt through a commitment made after bob's
	a3 := test.CommitRegistrationsAction{
		Actor:    ctx.GetActor("alice"),
		PolicyId: pol.Id,
		Objects: []*coretypes.Object{
			coretypes.NewObject("file", "foo.txt"),
		},
	}
	aliceComm := a3.Run(ctx)
	ctx.WaitBlock()
	a4 := test.RevealRegistrationAction{
		Actor:        ctx.GetActor("alice"),
		PolicyId:     pol.Id,
		CommitmentId: aliceComm.Id,
		Objects: []*coretypes.Object{
			coretypes.NewObject("file", "foo.txt"),
		},
		Index: 0,
	}
	a4.Run(ctx)
	ctx.WaitBlock()

	// When Bob reveals foo.txt
	a := test.RevealRegistrationAction{
		Actor:        ctx.GetActor("bob"),
		PolicyId:     pol.Id,
		CommitmentId: commitment.Id,
		Objects: []*coretypes.Object{
			coretypes.NewObject("file", "foo.txt"),
		},
		Index: 0,
	}
	result := a.Run(ctx)

	// Then Bob is the owner of foo.txt
	require.Equal(ctx.T, result.Event.Type, types.ObjectRegistrationEventType_AMENDMENT)
	require.Equal(ctx.T, result.Record.OwnerDid, ctx.GetActor("bob").DID)
	require.Equal(ctx.T, result.Record.Relationship, coretypes.NewActorRelationship("file", "foo.txt", "owner", ctx.GetActor("bob").DID))
}

func TestRevealRegistration_ObjectOwnedByUserAfterCommitment_NoOp(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()
	//TODOt.FailNow()
}

func TestRevealRegistration_ObjectRegisteredToSomeoneElseAfterCommitment_ErrorsUnauthorized(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	// Given Policy
	a1 := test.CreatePolicyAction{
		Policy:  revealPolicy,
		Creator: ctx.GetActor("bob"),
	}
	pol := a1.Run(ctx)
	// Given alice as owner of foo.txt
	a2 := test.RegisterObjectAction{
		Actor:    ctx.GetActor("alice"),
		PolicyId: pol.Id,
		Object:   coretypes.NewObject("file", "foo.txt"),
	}
	a2.Run(ctx)
	ctx.WaitBlock()
	// Given a commitment made by bob to foo.txt
	a3 := test.CommitRegistrationsAction{
		PolicyId: pol.Id,
		Actor:    ctx.GetActor("bob"),
		Objects: []*coretypes.Object{
			coretypes.NewObject("file", "foo.txt"),
		},
	}
	commitment := a3.Run(ctx)
	ctx.WaitBlock()

	// When Bob reveals foo.txt then bob is forbidden from doing so
	a := test.RevealRegistrationAction{
		Actor:        ctx.GetActor("bob"),
		PolicyId:     pol.Id,
		CommitmentId: commitment.Id,
		Objects: []*coretypes.Object{
			coretypes.NewObject("file", "foo.txt"),
		},
		Index:       0,
		ExpectedErr: types.ErrorType_OPERATION_FORBIDDEN,
	}
	a.Run(ctx)
}

func TestRevealRegistration_InvalidProof_ReturnsError(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	// Given Policy
	a1 := test.CreatePolicyAction{
		Policy:  revealPolicy,
		Creator: ctx.GetActor("bob"),
	}
	pol := a1.Run(ctx)
	// Given alice as owner of foo.txt
	a2 := test.RegisterObjectAction{
		Actor:    ctx.GetActor("alice"),
		PolicyId: pol.Id,
		Object:   coretypes.NewObject("file", "foo.txt"),
	}
	a2.Run(ctx)
	ctx.WaitBlock()
	// Given a commitment made by bob to foo.txt
	a3 := test.CommitRegistrationsAction{
		PolicyId: pol.Id,
		Actor:    ctx.GetActor("bob"),
		Objects: []*coretypes.Object{
			coretypes.NewObject("file", "foo.txt"),
		},
	}
	commitment := a3.Run(ctx)
	ctx.WaitBlock()

	// When Bob reveals foo.txt then bob is forbidden from doing so
	a := test.RevealRegistrationAction{
		Actor:        ctx.GetActor("bob"),
		PolicyId:     pol.Id,
		CommitmentId: commitment.Id,
		Objects: []*coretypes.Object{
			coretypes.NewObject("file", "foo.txt"),
		},
		Index:       0,
		ExpectedErr: types.ErrorType_OPERATION_FORBIDDEN,
	}
	a.Run(ctx)
}

func TestRevealRegistration_ValidProofToExpiredCommitment_ReturnsProtocolError(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	// Given Policy
	a1 := test.CreatePolicyAction{
		Policy:  revealPolicy,
		Creator: ctx.GetActor("bob"),
	}
	pol := a1.Run(ctx)
	// Given a commitment made by bob to foo.txt
	a2 := test.CommitRegistrationsAction{
		PolicyId: pol.Id,
		Actor:    ctx.GetActor("bob"),
		Objects: []*coretypes.Object{
			coretypes.NewObject("file", "foo.txt"),
		},
	}
	commitment := a2.Run(ctx)
	ctx.WaitBlocks(ctx.Params.RegistrationsCommitmentValidity.GetBlockCount() + 1)

	// When Bob reveals foo.txt Bob is forbidden from doing so
	a := test.RevealRegistrationAction{
		Actor:        ctx.GetActor("bob"),
		PolicyId:     pol.Id,
		CommitmentId: commitment.Id,
		Objects: []*coretypes.Object{
			coretypes.NewObject("file", "foo.txt"),
		},
		Index:       0,
		ExpectedErr: types.ErrorType_OPERATION_FORBIDDEN,
	}
	a.Run(ctx)
}
