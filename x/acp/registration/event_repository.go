package registration

import (
	"context"

	coretypes "github.com/sourcenetwork/acp_core/pkg/types"

	"github.com/sourcenetwork/acp_core/pkg/errors"
	"github.com/sourcenetwork/raccoondb/v2/iterator"
	"github.com/sourcenetwork/raccoondb/v2/marshal"
	"github.com/sourcenetwork/raccoondb/v2/store"
	"github.com/sourcenetwork/raccoondb/v2/table"
	rctypes "github.com/sourcenetwork/raccoondb/v2/types"
	"github.com/sourcenetwork/sourcehub/x/acp/stores"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

func NewObjectEventRepository(kv store.KVStore) (RegistrationEventRepository, error) {
	marshaler := stores.NewGogoProtoMarshaler(func() *types.ObjectRegistrationEvent { return &types.ObjectRegistrationEvent{} })
	t := table.NewTable(kv, marshaler)

	getter := func(ev **types.ObjectRegistrationEvent) uint64 {
		return (*ev).Id
	}
	setter := func(ev **types.ObjectRegistrationEvent, id uint64) {
		(*ev).Id = id
	}
	incrementer := table.NewAutoIncrementer(t, getter, setter)

	extractor := func(ev **types.ObjectRegistrationEvent) string {
		return (*ev).PolicyId
	}
	polIdx, err := table.NewIndex(t, "policy", extractor, marshal.StringMarshaler{})
	if err != nil {
		return nil, err
	}

	return &KVEventRepository{
		t:           t,
		incrementer: incrementer,
		polIdx:      polIdx,
	}, nil
}

var _ RegistrationEventRepository = (*KVEventRepository)(nil)

type KVEventRepository struct {
	t           *table.Table[*types.ObjectRegistrationEvent]
	polIdx      table.IndexReader[*types.ObjectRegistrationEvent, string]
	incrementer *table.Autoincrementer[*types.ObjectRegistrationEvent]
}

func (r *KVEventRepository) wrapErr(err error) error {
	if err == nil {
		return err
	}

	return errors.NewFromBaseError(err, errors.ErrorType_INTERNAL, "registration event repository")
}

func (r *KVEventRepository) Create(ctx context.Context, reg *types.ObjectRegistrationEvent) error {
	err := r.incrementer.Insert(ctx, &reg)
	return r.wrapErr(err)
}

func (r *KVEventRepository) Set(ctx context.Context, reg *types.ObjectRegistrationEvent) error {
	err := r.incrementer.Update(ctx, &reg)
	return r.wrapErr(err)
}

func (r *KVEventRepository) GetById(ctx context.Context, id uint64) (rctypes.Option[*types.ObjectRegistrationEvent], error) {
	opt, err := r.incrementer.GetByID(ctx, id)
	if err != nil {
		return rctypes.None[*types.ObjectRegistrationEvent](), r.wrapErr(err)
	}
	return opt, nil
}

func (r *KVEventRepository) GetObjectEvents(ctx context.Context, policyId string, object *coretypes.Object) (iterator.Iterator[*types.ObjectRegistrationEvent], error) {
	keysIter, err := r.polIdx.IterateKeys(ctx, &policyId, store.NewOpenIterator())
	if err != nil {
		return nil, r.wrapErr(err)
	}

	iter := table.MaterializeObjects(ctx, r.t, keysIter)
	iter = iterator.Filter(iter, func(ev *types.ObjectRegistrationEvent) bool {
		return ev.Object.Resource == object.Resource && ev.Object.Id == ev.Object.Id
	})

	return iter, nil
}

func (r *KVEventRepository) ListHijackEventsByPolicy(ctx context.Context, policyId string) (iterator.Iterator[*types.ObjectRegistrationEvent], error) {
	keysIter, err := r.polIdx.IterateKeys(ctx, &policyId, store.NewOpenIterator())
	if err != nil {
		return nil, r.wrapErr(err)
	}
	iter := table.MaterializeObjects(ctx, r.t, keysIter)

	iter = iterator.Filter(iter, func(ev *types.ObjectRegistrationEvent) bool {
		return ev.Type == types.ObjectRegistrationEventType_AMENDMENT && ev.Detail.GetAmendmentEvent().HijackFlag
	})
	return iter, nil
}
