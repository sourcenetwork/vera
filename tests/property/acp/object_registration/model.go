package object_registration

import (
	"fmt"
	"reflect"
	"testing"

	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/sourcenetwork/sourcehub/tests/property"
	"github.com/sourcenetwork/sourcehub/utils"
	"github.com/sourcenetwork/sourcehub/x/acp/commitment"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
	"github.com/stretchr/testify/require"
)

// preconditions validate whether op should be run given the current state
func Precoditions(state State, kind OperationKind) bool {
	switch kind {
	case Reveal:
		// don't reveal unless there's at least one commitment registered
		return len(state.Commitments) > 0
	case Unarchive, Archive:
		return state.Registered
	default:
		// remaining ops can happen, doesn't matter if they error
		return true
	}
}

// GenerateOperation generates the request payload for some operation and state
func GenerateOperation(t *testing.T, state State, kind OperationKind) Operation {
	actor := property.PickAny(state.Actors)
	op := Operation{
		Kind:  kind,
		Actor: actor,
	}

	switch kind {
	case Reveal:
		id := property.PickAny(utils.MapSlice(state.Commitments, func(commitment *types.RegistrationsCommitment) uint64 { return commitment.Id }))

		proof, err := commitment.ProofForObject(state.PolicyId, coretypes.NewActor(actor.DID), 0, []*coretypes.Object{&state.Object})
		require.NoError(t, err)

		op.Request = types.PolicyCmd{
			Cmd: &types.PolicyCmd_RevealRegistrationCmd{
				RevealRegistrationCmd: &types.RevealRegistrationCmd{
					RegistrationsCommitmentId: id,
					Proof:                     proof,
				},
			},
		}
	case Register:
		op.Request = types.PolicyCmd{
			Cmd: &types.PolicyCmd_RegisterObjectCmd{
				RegisterObjectCmd: &types.RegisterObjectCmd{
					Object: &state.Object,
				},
			},
		}
	case Archive:
		op.Request = types.PolicyCmd{
			Cmd: &types.PolicyCmd_ArchiveObjectCmd{
				ArchiveObjectCmd: &types.ArchiveObjectCmd{
					Object: &state.Object,
				},
			},
		}
	case Commit:
		c, err := commitment.GenerateCommitmentWithoutValidation(state.PolicyId, coretypes.NewActor(actor.DID), []*coretypes.Object{&state.Object})
		require.NoError(t, err)
		op.Request = types.PolicyCmd{
			Cmd: &types.PolicyCmd_CommitRegistrationsCmd{
				CommitRegistrationsCmd: &types.CommitRegistrationsCmd{
					Commitment: c,
				},
			},
		}
	case Unarchive:
		op.Request = types.PolicyCmd{
			Cmd: &types.PolicyCmd_UnarchiveObjectCmd{
				UnarchiveObjectCmd: &types.UnarchiveObjectCmd{
					Object: &state.Object,
				},
			},
		}
	}
	return op
}

// NextState updates the state object based on the operation result
func NextState(t *testing.T, state State, op Operation) State {
	t.Logf("Operation: kind=%v; actor=%v...; payload=%v", op.Kind, op.Actor.DID[0:20], op.Request.Cmd)
	if op.ResultErr != nil {
		t.Logf("\tfailed: err=%v", op.ResultErr)
		return state
	}

	switch op.Kind {
	case Register:
		if !state.Registered {
			state.Model.RegistrationTs = *op.Result.GetRegisterObjectResult().Record.Metadata.CreationTs
			state.Model.Owner = op.Actor.DID
			state.Registered = true
		}
	case Archive:
		if op.Actor.DID == state.Model.Owner {
			state.Model.Archived = true
		}
	case Commit:
		result := op.Result.Result.(*types.PolicyCmdResult_CommitRegistrationsResult).CommitRegistrationsResult
		state.Commitments = append(state.Commitments, result.RegistrationsCommitment)
	case Unarchive:
		if op.Actor.DID == state.Model.Owner {
			state.Model.Archived = false
		}
	case Reveal:
		commitId := op.Request.GetRevealRegistrationCmd().RegistrationsCommitmentId
		c := state.MustCommitmentById(commitId)

		expirationHeight := c.Metadata.CreationTs.BlockHeight + c.Validity.GetBlockCount()
		blockHeight := state.LastTs.BlockHeight
		isValid := blockHeight < expirationHeight

		isCommitmentOwner := c.Metadata.OwnerDid == op.Actor.DID
		isCommitmentOlder := c.Metadata.CreationTs.BlockHeight < state.Model.RegistrationTs.BlockHeight

		if isCommitmentOwner && isCommitmentOlder && isValid {
			state.Model.Owner = op.Actor.DID
			state.Model.RegistrationTs = *c.Metadata.CreationTs
		}
	}
	return state
}

// Post validate whether the system invariants held after the execution of op
func Post(state State, op Operation) error {
	if !state.Registered {
		return nil
	}

	// compare model data with result record
	if state.Model.Owner != op.ResultRecord.Metadata.OwnerDid {
		return fmt.Errorf("model missmatch: owner: expected %v, got %v",
			state.Model.Owner,
			op.ResultRecord.Metadata.OwnerDid,
		)

	}
	if state.Model.Archived != op.ResultRecord.Archived {
		return fmt.Errorf("model missmatch: archived: expected %v, got %v",
			state.Model.Archived,
			op.ResultRecord.Archived,
		)
	}
	if reflect.DeepEqual(state.Model.RegistrationTs, op.ResultRecord.Metadata.CreationTs) {
		return fmt.Errorf("model missmatch: ts: expected %v, got %v",
			state.Model.RegistrationTs,
			op.ResultRecord.Metadata.CreationTs,
		)
	}
	return nil
}
