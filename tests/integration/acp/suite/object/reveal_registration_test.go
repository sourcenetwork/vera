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
		Creator: ctx.TxSigner,
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
	t.Logf("commitment generated: %v", commitment.Commitment)
	t.Logf("registrations commitment id: %v", commitment.Id)
	ctx.WaitBlock()

	t.Logf("reveal registration for file:foo.txt")
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
	t.Logf("created relationship: %v", result.Record)
}

// This tests documents deals with the edge case where the object owner
// commits to the object, performs explicit registration and then reveals the same object.
// We assert that the record timestamp invariant holds, even for this odd scenario
func TestRevealRegistration_ObjectRegisteredToActor_ReturnRecordWithCommitmentTimestamp(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	a1 := test.CreatePolicyAction{
		Policy:  revealPolicy,
		Creator: ctx.TxSigner,
	}
	pol := a1.Run(ctx)
	// Given commitment and object registered after commitment
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
		PolicyId: pol.Id,
		Object:   coretypes.NewObject("file", "foo.txt"),
		Actor:    ctx.GetActor("bob"),
	}
	a3.Run(ctx)
	ctx.WaitBlock()

	// When Bob opens commitment for foo.txt
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

	// Then result contains relationship registered
	// at commit creation time
	require.Equal(ctx.T, commitment.Metadata.CreationTs, result.Record.Metadata.CreationTs)
}

func TestRevealRegistration_ObjectRegisteredAfterCommitment_RegistrationAmended(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	// Given a commitment by bob to foo.txt
	// followed by Alice's registration of foo.txt
	a1 := test.CreatePolicyAction{
		Policy:  revealPolicy,
		Creator: ctx.TxSigner,
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
		PolicyId: pol.Id,
		Object:   coretypes.NewObject("file", "foo.txt"),
		Actor:    ctx.GetActor("alice"),
	}
	a3.Run(ctx)
	ctx.WaitBlock()

	// When Bob reveals the commitment to foo.txt
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

	// Then foo.txt is transfered to bob
	require.Equal(ctx.T, uint64(1), result.Event.Id)
	require.Equal(ctx.T, result.Record.Metadata.OwnerDid, ctx.GetActor("bob").DID)
	require.Equal(ctx.T, result.Record.Relationship, coretypes.NewActorRelationship("file", "foo.txt", "owner", ctx.GetActor("bob").DID))
}

func TestRevealRegistration_ObjectRegisteredThroughNewerCommitment_RegistrationIsAmended(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	// Given Policy
	a1 := test.CreatePolicyAction{
		Policy:  revealPolicy,
		Creator: ctx.TxSigner,
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
	require.Equal(ctx.T, uint64(1), result.Event.Id)
	require.Equal(ctx.T, result.Record.Metadata.OwnerDid, ctx.GetActor("bob").DID)
	require.Equal(ctx.T, result.Record.Relationship, coretypes.NewActorRelationship("file", "foo.txt", "owner", ctx.GetActor("bob").DID))
}

func TestRevealRegistration_ObjectRegisteredToSomeoneElseAfterCommitment_ErrorsUnauthorized(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	// Given Policy
	a1 := test.CreatePolicyAction{
		Policy:  revealPolicy,
		Creator: ctx.TxSigner,
	}
	pol := a1.Run(ctx)
	// Given alice as owner of foo.txt
	a2 := test.RegisterObjectAction{
		PolicyId: pol.Id,
		Object:   coretypes.NewObject("file", "foo.txt"),
		Actor:    ctx.GetActor("alice"),
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
		Creator: ctx.TxSigner,
	}
	pol := a1.Run(ctx)
	// Given alice as owner of foo.txt
	a2 := test.RegisterObjectAction{
		PolicyId: pol.Id,
		Object:   coretypes.NewObject("file", "foo.txt"),
		Actor:    ctx.GetActor("alice"),
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
		Creator: ctx.TxSigner,
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

func TestRevealRegistration_RevealingRegistrationTwice_Errors(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	a1 := test.CreatePolicyAction{
		Policy:  revealPolicy,
		Creator: ctx.TxSigner,
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
	ctx.WaitBlock()

	// When bob reveals the same registartion twice
	a = test.RevealRegistrationAction{
		Actor:        ctx.GetActor("bob"),
		PolicyId:     pol.Id,
		CommitmentId: commitment.Id,
		Objects: []*coretypes.Object{
			coretypes.NewObject("file", "foo.txt"),
			coretypes.NewObject("file", "bar.txt"),
		},
		Index:       0,
		ExpectedErr: types.ErrorType_OPERATION_FORBIDDEN,
	}
}
