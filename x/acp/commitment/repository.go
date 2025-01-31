package commitment

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

const commitmentObjsPrefix = "/objs"
const commitmentCounterPrefix = "/counter"

// NewCommitmentRepository returns a CommitmentRepository from a kv
func NewCommitmentRepository(kv store.KVStore) (*CommitmentRepository, error) {
	marshaler := stores.NewGogoProtoMarshaler(func() *types.RegistrationsCommitment {
		return &types.RegistrationsCommitment{}
	})
	t := table.NewTable(kv, marshaler)

	tsExtractor := func(rec **types.RegistrationsCommitment) bool {
		return (*rec).Expired
	}
	expiredIdx, err := table.NewIndex(t, "expired", tsExtractor, marshal.BoolMarshaler{})
	if err != nil {
		return nil, err
	}

	commExtractor := func(rec **types.RegistrationsCommitment) []byte {
		return (*rec).Commitment
	}
	commIdx, err := table.NewIndex(t, "commitment", commExtractor, marshal.BytesMarshaler{})
	if err != nil {
		return nil, err
	}

	getter := func(rec **types.RegistrationsCommitment) uint64 {
		return (*rec).Id
	}
	setter := func(rec **types.RegistrationsCommitment, id uint64) {
		(*rec).Id = id
	}
	incrementer := table.NewAutoIncrementer(t, getter, setter)

	return &CommitmentRepository{
		table:       t,
		incrementer: incrementer,
		commIndex:   commIdx,
		expiredIdx:  expiredIdx,
	}, nil
}

// CommitmentRepository exposes an interface to manipulate RegistrationsCommitment records
type CommitmentRepository struct {
	table       *table.Table[*types.RegistrationsCommitment]
	incrementer *table.Autoincrementer[*types.RegistrationsCommitment]
	commIndex   table.IndexReader[*types.RegistrationsCommitment, []byte]
	expiredIdx  table.IndexReader[*types.RegistrationsCommitment, bool]
}

func (r *CommitmentRepository) wrapErr(err error) error {
	if err == nil {
		return err
	}
	return errors.NewFromBaseError(err, errors.ErrorType_INTERNAL, "registration repository")
}

// update stores the updated RegistrationCommitment record
func (r *CommitmentRepository) update(ctx context.Context, reg *types.RegistrationsCommitment) error {
	err := r.incrementer.Update(ctx, &reg)
	return r.wrapErr(err)
}

// create sets a new RegistrationCommitment using the next free up id.
// Sets reg.Id with the effective record Id used.
func (r *CommitmentRepository) create(ctx context.Context, reg *types.RegistrationsCommitment) error {
	err := r.incrementer.Insert(ctx, &reg)
	return r.wrapErr(err)
}

// GetById returns a RegistrationCommitment with the given id
func (r *CommitmentRepository) GetById(ctx context.Context, id uint64) (rctypes.Option[*types.RegistrationsCommitment], error) {
	comm := &types.RegistrationsCommitment{Id: id}
	opt, err := r.incrementer.GetByRecordID(ctx, &comm)
	if err != nil {
		return rctypes.None[*types.RegistrationsCommitment](), r.wrapErr(err)
	}
	return opt, nil
}

// FilterByCommitment returns all RegistrationCommitment records with the given commitment
func (r *CommitmentRepository) FilterByCommitment(ctx context.Context, commitment []byte) (iterator.Iterator[*types.RegistrationsCommitment], error) {
	keyIter, err := r.commIndex.IterateKeys(ctx, &commitment, store.NewOpenIterator())
	if err != nil {
		return nil, err
	}
	iter := table.MaterializeObjects(ctx, r.table, keyIter)
	return iter, nil
}

// GetNonExpiredCommitments returns all commitments whose expiration flag is false
func (r *CommitmentRepository) GetNonExpiredCommitments(ctx context.Context) (iterator.Iterator[*types.RegistrationsCommitment], error) {
	bkt := false
	keyIter, err := r.expiredIdx.IterateKeys(ctx, &bkt, store.NewOpenIterator())
	if err != nil {
		return nil, err
	}

	iter := table.MaterializeObjects(ctx, r.table, keyIter)
	return iter, nil
}
