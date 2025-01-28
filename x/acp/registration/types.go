// package registration exposes types for the object registration protocol
package registration

import (
	"context"

	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/sourcenetwork/raccoondb/v2/iterator"
	rctypes "github.com/sourcenetwork/raccoondb/v2/types"

	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

type RegistrationEventRepository interface {
	Set(ctx context.Context, event *types.ObjectRegistrationEvent) error
	Create(ctx context.Context, event *types.ObjectRegistrationEvent) error
	GetById(ctx context.Context, id uint64) (rctypes.Option[*types.ObjectRegistrationEvent], error)
	GetObjectEvents(ctx context.Context, policyId string, object *coretypes.Object) (iterator.Iterator[*types.ObjectRegistrationEvent], error)
	ListHijackEventsByPolicy(ctx context.Context, policyId string) (iterator.Iterator[*types.ObjectRegistrationEvent], error)
}
