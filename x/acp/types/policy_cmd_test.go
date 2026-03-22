package types

import (
	"testing"

	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/stretchr/testify/require"
)

func TestNewSetRelationshipCmd(t *testing.T) {
	rel := coretypes.NewActorRelationship("resource", "obj1", "reader", "did:example:alice")
	cmd := NewSetRelationshipCmd(rel)
	require.NotNil(t, cmd)
	setCmd, ok := cmd.Cmd.(*PolicyCmd_SetRelationshipCmd)
	require.True(t, ok)
	require.Equal(t, rel, setCmd.SetRelationshipCmd.Relationship)
}

func TestNewDeleteRelationshipCmd(t *testing.T) {
	rel := coretypes.NewActorRelationship("resource", "obj1", "reader", "did:example:alice")
	cmd := NewDeleteRelationshipCmd(rel)
	require.NotNil(t, cmd)
	delCmd, ok := cmd.Cmd.(*PolicyCmd_DeleteRelationshipCmd)
	require.True(t, ok)
	require.Equal(t, rel, delCmd.DeleteRelationshipCmd.Relationship)
}

func TestNewRegisterObjectCmd(t *testing.T) {
	obj := coretypes.NewObject("resource", "obj1")
	cmd := NewRegisterObjectCmd(obj)
	require.NotNil(t, cmd)
	regCmd, ok := cmd.Cmd.(*PolicyCmd_RegisterObjectCmd)
	require.True(t, ok)
	require.Equal(t, obj, regCmd.RegisterObjectCmd.Object)
}

func TestNewArchiveObjectCmd(t *testing.T) {
	obj := coretypes.NewObject("resource", "obj1")
	cmd := NewArchiveObjectCmd(obj)
	require.NotNil(t, cmd)
	archCmd, ok := cmd.Cmd.(*PolicyCmd_ArchiveObjectCmd)
	require.True(t, ok)
	require.Equal(t, obj, archCmd.ArchiveObjectCmd.Object)
}

func TestNewCommitRegistrationCmd(t *testing.T) {
	commitment := []byte("commitment-hash")
	cmd := NewCommitRegistrationCmd(commitment)
	require.NotNil(t, cmd)
	commitCmd, ok := cmd.Cmd.(*PolicyCmd_CommitRegistrationsCmd)
	require.True(t, ok)
	require.Equal(t, commitment, commitCmd.CommitRegistrationsCmd.Commitment)
}

func TestNewRevealRegistrationCmd(t *testing.T) {
	proof := &RegistrationProof{
		Object: coretypes.NewObject("resource", "obj1"),
	}
	cmd := NewRevealRegistrationCmd(42, proof)
	require.NotNil(t, cmd)
	revealCmd, ok := cmd.Cmd.(*PolicyCmd_RevealRegistrationCmd)
	require.True(t, ok)
	require.Equal(t, proof, revealCmd.RevealRegistrationCmd.Proof)
	require.Equal(t, uint64(42), revealCmd.RevealRegistrationCmd.RegistrationsCommitmentId)
}

func TestNewFlagHijackAttemptCmd(t *testing.T) {
	cmd := NewFlagHijackAttemptCmd(99)
	require.NotNil(t, cmd)
	flagCmd, ok := cmd.Cmd.(*PolicyCmd_FlagHijackAttemptCmd)
	require.True(t, ok)
	require.Equal(t, uint64(99), flagCmd.FlagHijackAttemptCmd.EventId)
}

func TestNewUnarchiveObjectCmd(t *testing.T) {
	obj := coretypes.NewObject("resource", "obj1")
	cmd := NewUnarchiveObjectCmd(obj)
	require.NotNil(t, cmd)
	unarchCmd, ok := cmd.Cmd.(*PolicyCmd_UnarchiveObjectCmd)
	require.True(t, ok)
	require.Equal(t, obj, unarchCmd.UnarchiveObjectCmd.Object)
}
