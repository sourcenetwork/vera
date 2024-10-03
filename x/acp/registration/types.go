// package registration exposes types for the object registration protocol
package registration

import (
	"context"
	"time"

	coretypes "github.com/sourcenetwork/acp_core/pkg/types"

	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

type CommitmentRepository interface {
	Set(ctx context.Context, reg *types.RegistrationsCommitment) error

	GetById(ctx context.Context, id string) (*types.RegistrationsCommitment, error)

	FilterByCommitment(ctx context.Context, commitment []byte) ([]*types.RegistrationsCommitment, error)

	GetExpiredCommitments(ctx context.Context, currentTime time.Time) ([]*types.RegistrationsCommitment, error)
}

type RegistrationEventRepository interface {
	Set(ctx context.Context, event *types.ObjectRegistrationEvent) error
	GetById(ctx context.Context, id string) (*types.ObjectRegistrationEvent, error)
	GetObjectEvents(ctx context.Context, policyId string, object *coretypes.Object) ([]*types.ObjectRegistrationEvent, error)
}
