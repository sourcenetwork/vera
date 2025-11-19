package object

import (
	"testing"

	"github.com/sourcenetwork/acp_core/pkg/errors"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	test "github.com/sourcenetwork/sourcehub/tests/integration/acp"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
	"github.com/stretchr/testify/require"
)

var commitPolicy string = `
name: policy
resources:
  resource:
    relations:
      owner:
        types:
          - actor
`

func TestCommitRegistration_CreatingCommitmentReturnsID(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	// Given Policy
	a1 := test.CreatePolicyAction{
		Policy:  commitPolicy,
		Creator: ctx.TxSigner,
	}
	pol := a1.Run(ctx)

	// When bob commits to foo.txt
	a2 := test.CommitRegistrationsAction{
		PolicyId: pol.Id,
		Actor:    ctx.GetActor("bob"),
		Objects: []*coretypes.Object{
			coretypes.NewObject("file", "foo.txt"),
		},
	}
	commitment := a2.GetCommitment(ctx)
	metadata := ctx.GetRecordMetadataForActor("bob")
	metadata.CreationTs.BlockHeight++
	a2.Expected = &types.RegistrationsCommitment{
		Id:         1,
		PolicyId:   ctx.State.PolicyId,
		Commitment: commitment,
		Expired:    false,
		Validity:   ctx.Params.RegistrationsCommitmentValidity,
		Metadata:   metadata,
	}
	a2.Run(ctx)
}

func TestCommitRegistration_CreateAndGetCommitment(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	// Given Policy
	a1 := test.CreatePolicyAction{
		Policy:  commitPolicy,
		Creator: ctx.TxSigner,
	}
	pol := a1.Run(ctx)

	a2 := test.CommitRegistrationsAction{
		PolicyId: pol.Id,
		Actor:    ctx.GetActor("bob"),
		Objects: []*coretypes.Object{
			coretypes.NewObject("file", "foo.txt"),
		},
	}
	commitment := a2.Run(ctx)

	got, err := ctx.Executor.RegistrationsCommitment(ctx, &types.QueryRegistrationsCommitmentRequest{
		Id: commitment.Id,
	})
	require.NoError(t, err)
	require.Equal(t, commitment, got.RegistrationsCommitment)
}

func TestCommitRegistration_CommitmentsGenerateDifferentIds(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	// Given Policy
	a1 := test.CreatePolicyAction{
		Policy:  commitPolicy,
		Creator: ctx.TxSigner,
	}
	pol := a1.Run(ctx)

	a2 := test.CommitRegistrationsAction{
		PolicyId: pol.Id,
		Actor:    ctx.GetActor("bob"),
		Objects: []*coretypes.Object{
			coretypes.NewObject("file", "foo.txt"),
		},
	}
	c1 := a2.Run(ctx)

	a3 := test.CommitRegistrationsAction{
		PolicyId: pol.Id,
		Actor:    ctx.GetActor("bob"),
		Objects: []*coretypes.Object{
			coretypes.NewObject("file", "foo.txt"),
		},
	}
	c2 := a3.Run(ctx)

	require.NotEqual(ctx.T, c1.Id, c2.Id)
}

func TestCommitRegistration_CommitmentWithInvalidId_Errors(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	// Given Policy
	a1 := test.CreatePolicyAction{
		Policy:  commitPolicy,
		Creator: ctx.TxSigner,
	}
	pol := a1.Run(ctx)

	a2 := test.CommitRegistrationsAction{
		PolicyId:    pol.Id,
		Actor:       ctx.GetActor("bob"),
		Commitment:  []byte{0x0, 0x1, 0x2},
		ExpectedErr: errors.ErrorType_BAD_INPUT,
	}
	a2.Run(ctx)
}
