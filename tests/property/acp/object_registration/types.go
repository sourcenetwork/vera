package object_registration

import (
	"github.com/google/uuid"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	test "github.com/sourcenetwork/sourcehub/tests/integration/acp"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
	"github.com/stretchr/testify/require"
)

// ObjectModel models the status of the object under test
type ObjectModel struct {
	RegistrationTs types.Timestamp
	Owner          string
	Archived       bool
}

// State models the test state which mutates between operations
type State struct {
	Model       ObjectModel
	Registered  bool
	PolicyId    string
	Object      coretypes.Object
	Commitments []*types.RegistrationsCommitment
	Actors      []*test.TestActor
	LastTs      types.Timestamp
}

// MustCommitmentById returns one of the commitments created during the test
// with the given id
func (s *State) MustCommitmentById(id uint64) *types.RegistrationsCommitment {
	for _, commitment := range s.Commitments {
		if commitment.Id == id {
			return commitment
		}
	}
	panic("commitment not found")
}

// InitialState returns the initial state for the model state machine
func InitialState(ctx *test.TestCtx, policyId string) State {
	actors := make([]*test.TestActor, 0, ActorCount)
	for i := 0; i < ActorCount; i++ {
		id, err := uuid.NewUUID()
		require.NoError(ctx.T, err)

		actor := ctx.GetActor(id.String())
		actors = append(actors, actor)
	}

	return State{
		Model:       ObjectModel{},
		Registered:  false,
		PolicyId:    policyId,
		Object:      *coretypes.NewObject(ResourceName, ObjectId),
		Commitments: nil,
		Actors:      actors,
	}
}

// OperationKind models the set of supported operations which transition the model
// to a new state
type OperationKind int

const (
	Register OperationKind = iota
	Archive
	Commit
	Reveal
	Unarchive
)

func (k OperationKind) String() string {
	switch k {
	case Register:
		return "Register"
	case Archive:
		return "Archive"
	case Commit:
		return "Commit"
	case Reveal:
		return "Reveal"
	case Unarchive:
		return "Unarchive"
	default:
		return "UNKNOWN"
	}
}

// GetOperations returns all available OperationKinds
func ListOperationKinds() []OperationKind {
	return []OperationKind{
		Register,
		Archive,
		Commit,
		Reveal,
		Unarchive,
	}

}

// Operation models an operation that has been or will be executed against the
// reference implementation
type Operation struct {
	Kind         OperationKind
	Actor        *test.TestActor
	Request      types.PolicyCmd
	Result       types.PolicyCmdResult
	ResultErr    error
	ResultRecord types.RelationshipRecord
}
