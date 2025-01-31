package registration

import (
	"context"

	"github.com/sourcenetwork/acp_core/pkg/errors"
	"github.com/sourcenetwork/raccoondb/v2/iterator"
	"github.com/sourcenetwork/raccoondb/v2/marshal"
	"github.com/sourcenetwork/raccoondb/v2/store"
	"github.com/sourcenetwork/raccoondb/v2/table"
	rctypes "github.com/sourcenetwork/raccoondb/v2/types"
	"github.com/sourcenetwork/sourcehub/x/acp/stores"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

// NewAmendmentEventRepository returns a repository which manages AmendmentEvent records
func NewAmendmentEventRepository(kv store.KVStore) (*AmendmentEventRepository, error) {
	marshaler := stores.NewGogoProtoMarshaler(func() *types.AmendmentEvent { return &types.AmendmentEvent{} })
	t := table.NewTable(kv, marshaler)

	getter := func(ev **types.AmendmentEvent) uint64 {
		return (*ev).Id
	}
	setter := func(ev **types.AmendmentEvent, id uint64) {
		(*ev).Id = id
	}
	incrementer := table.NewAutoIncrementer(t, getter, setter)

	extractor := func(ev **types.AmendmentEvent) string {
		return (*ev).PolicyId
	}
	polIdx, err := table.NewIndex(t, "policy", extractor, marshal.StringMarshaler{})
	if err != nil {
		return nil, err
	}

	return &AmendmentEventRepository{
		t:           t,
		incrementer: incrementer,
		polIdx:      polIdx,
	}, nil
}

// RegistrationEventRepsository exposes operations
// to access and store AmendmentEvent records
type AmendmentEventRepository struct {
	t           *table.Table[*types.AmendmentEvent]
	polIdx      table.IndexReader[*types.AmendmentEvent, string]
	incrementer *table.Autoincrementer[*types.AmendmentEvent]
}

func (r *AmendmentEventRepository) wrapErr(err error) error {
	if err == nil {
		return err
	}
	return errors.NewFromBaseError(err, errors.ErrorType_INTERNAL, "amendment event repository")
}

// create uses Raccoon's autoincrement to
// insert a new event with an autoincremented ID
func (r *AmendmentEventRepository) create(ctx context.Context, reg *types.AmendmentEvent) error {
	err := r.incrementer.Insert(ctx, &reg)
	return r.wrapErr(err)
}

// update updates the data in a record
func (r *AmendmentEventRepository) update(ctx context.Context, reg *types.AmendmentEvent) error {
	err := r.incrementer.Update(ctx, &reg)
	return r.wrapErr(err)
}

// GetById returns a record with the given ID
func (r *AmendmentEventRepository) GetById(ctx context.Context, id uint64) (rctypes.Option[*types.AmendmentEvent], error) {
	opt, err := r.incrementer.GetByID(ctx, id)
	if err != nil {
		return rctypes.None[*types.AmendmentEvent](), r.wrapErr(err)
	}
	return opt, nil
}

// ListHijackEventsByPolicy returns all flagged AmendmentEvents for a Policy
func (r *AmendmentEventRepository) ListHijackEventsByPolicy(ctx context.Context, policyId string) (iterator.Iterator[*types.AmendmentEvent], error) {
	iter, err := r.ListEventsByPolicy(ctx, policyId)
	if err != nil {
		return nil, err
	}
	iter = iterator.Filter(iter, func(ev *types.AmendmentEvent) bool { return ev.HijackFlag })
	return iter, nil
}

// ListHijackEventsByPolicy returns all AmendmentEvents for a Policy
func (r *AmendmentEventRepository) ListEventsByPolicy(ctx context.Context, policyId string) (iterator.Iterator[*types.AmendmentEvent], error) {
	keysIter, err := r.polIdx.IterateKeys(ctx, &policyId, store.NewOpenIterator())
	if err != nil {
		return nil, r.wrapErr(err)
	}
	iter := table.MaterializeObjects(ctx, r.t, keysIter)
	return iter, nil
}
