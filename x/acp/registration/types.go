// package registration exposes types for the object registration protocol
package registration

import (
	"context"

	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/sourcenetwork/raccoondb/v2/iterator"
	rctypes "github.com/sourcenetwork/raccoondb/v2/types"

	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

type CommitmentRepository interface {
	// Create sets a new RegistrationCommitment using the next free up id.
	// Sets reg.Id with the effective record Id used.
	Create(ctx context.Context, reg *types.RegistrationsCommitment) error

	Set(ctx context.Context, reg *types.RegistrationsCommitment) error

	GetById(ctx context.Context, id uint64) (rctypes.Option[*types.RegistrationsCommitment], error)

	FilterByCommitment(ctx context.Context, commitment []byte) (iterator.Iterator[*types.RegistrationsCommitment], error)

	// GetNonExpiredCommitments returns all commitments whose expiration flag is false
	GetNonExpiredCommitments(ctx context.Context) (iterator.Iterator[*types.RegistrationsCommitment], error)
}

type RegistrationEventRepository interface {
	Set(ctx context.Context, event *types.ObjectRegistrationEvent) error
	Create(ctx context.Context, event *types.ObjectRegistrationEvent) error
	GetById(ctx context.Context, id uint64) (rctypes.Option[*types.ObjectRegistrationEvent], error)
	GetObjectEvents(ctx context.Context, policyId string, object *coretypes.Object) (iterator.Iterator[*types.ObjectRegistrationEvent], error)
	ListHijackEventsByPolicy(ctx context.Context, policyId string) (iterator.Iterator[*types.ObjectRegistrationEvent], error)
}
