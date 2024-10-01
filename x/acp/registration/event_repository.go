package registration

import (
	"context"

	coretypes "github.com/sourcenetwork/acp_core/pkg/types"

	storetypes "cosmossdk.io/store/types"
	"github.com/sourcenetwork/acp_core/pkg/errors"
	raccoon "github.com/sourcenetwork/raccoondb"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

var _ raccoon.Ider[*types.ObjectRegistrationEvent] = (*eventIder)(nil)

type eventIder struct{}

func (i *eventIder) Id(obj *types.ObjectRegistrationEvent) []byte {
	return []byte(obj.Id)
}

var _ RegistrationEventRepository = (*KVEventRepository)(nil)

type KVEventRepository struct {
	kv    storetypes.KVStore
	store raccoon.ObjectStore[*types.ObjectRegistrationEvent]
}

func (r *KVEventRepository) wrapErr(err error) error {
	if err == nil {
		return err
	}

	return errors.NewFromBaseError(err, errors.ErrorType_INTERNAL, "registration event repository")
}

func (r *KVEventRepository) Set(ctx context.Context, reg *types.ObjectRegistrationEvent) error {
	err := r.store.SetObject(reg)
	return r.wrapErr(err)
}

func (r *KVEventRepository) GetById(ctx context.Context, id string) (*types.ObjectRegistrationEvent, error) {
	opt, err := r.store.GetObject([]byte(id))
	if err != nil {
		return nil, r.wrapErr(err)
	}
	if opt.IsEmpty() {
		return nil, nil
	}
	return opt.Value(), nil
}

func (r *KVEventRepository) GetObjectEvents(ctx context.Context, policyId string, object coretypes.Object) ([]*types.ObjectRegistrationEvent, error) {
	recs, err := r.store.Filter(func(ev *types.ObjectRegistrationEvent) bool {
		return ev.PolicyId == policyId && ev.Object.Resource == object.Resource && ev.Object.Id == ev.Object.Id
	})
	if err != nil {
		return nil, r.wrapErr(err)
	}
	return recs, nil

}
