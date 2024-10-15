// package registration exposes types for the object registration protocol
package registration

import (
	"context"

	coretypes "github.com/sourcenetwork/acp_core/pkg/types"

	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

type CommitmentRepository interface {
	IncrementId(ctx context.Context) (uint64, error)

	Set(ctx context.Context, reg *types.RegistrationsCommitment) error

	GetById(ctx context.Context, id string) (*types.RegistrationsCommitment, error)

	FilterByCommitment(ctx context.Context, commitment []byte) ([]*types.RegistrationsCommitment, error)

	GetExpiredCommitments(ctx context.Context, currentTime *types.Timestamp) ([]*types.RegistrationsCommitment, error)
}

type RegistrationEventRepository interface {
	IncrementId(ctx context.Context) (uint64, error)
	Set(ctx context.Context, event *types.ObjectRegistrationEvent) error
	GetById(ctx context.Context, id string) (*types.ObjectRegistrationEvent, error)
	GetObjectEvents(ctx context.Context, policyId string, object *coretypes.Object) ([]*types.ObjectRegistrationEvent, error)
}
