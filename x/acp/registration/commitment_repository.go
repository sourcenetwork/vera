package registration

import (
	"context"

	"cosmossdk.io/store/prefix"

	storetypes "cosmossdk.io/store/types"
	"github.com/sourcenetwork/acp_core/pkg/errors"
	raccoon "github.com/sourcenetwork/raccoondb"
	"github.com/sourcenetwork/raccoondb/v2/iterator"
	"github.com/sourcenetwork/raccoondb/v2/store"
	"github.com/sourcenetwork/raccoondb/v2/stores"
	"github.com/sourcenetwork/raccoondb/v2/table"
	"github.com/sourcenetwork/sourcehub/x/acp/stores"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

const commitmentObjsPrefix = "/objs"
const commitmentCounterPrefix = "/counter"

var _ raccoon.Ider[*types.RegistrationsCommitment] = (*registrationIder)(nil)

type registrationIder struct{}

func (i *registrationIder) Id(obj *types.RegistrationsCommitment) []byte {
	return []byte(obj.Id)
}

var _ CommitmentRepository = (*KVRegistrationRepository)(nil)

func NewKVRegistrationRepository(kv storetypes.KVStore) CommitmentRepository {
	table.New
	objsKv := prefix.NewStore(kv, []byte(commitmentObjsPrefix))
	counterKv := prefix.NewStore(kv, []byte(eventsPrefix))

	objsRCKV := stores.RaccoonKVFromCosmos(objsKv)
	counterRCKV := stores.RaccoonKVFromCosmos(counterKv)

	factory := func() *types.RegistrationsCommitment { return &types.RegistrationsCommitment{} }
	objs := raccoon.NewObjStore(objsRCKV, stores.NewGogoProtoMarshaler(factory), &registrationIder{})
	return &KVRegistrationRepository{
		store:   objs,
		counter: raccoon.NewCounterStore("", counterRCKV, raccoon.NoopLogger()),
	}
}

type KVRegistrationRepository struct {
	store     raccoon.ObjectStore[*types.RegistrationsCommitment]
	counter   raccoon.CounterStore
	table     table.Table[types.RegistrationsCommitment]
	commIndex table.IndexReader[types.RegistrationsCommitment, []byte]
	timeIdx   table.IndexReader[types.RegistrationsCommitment, string]
}

func (r *KVRegistrationRepository) wrapErr(err error) error {
	if err == nil {
		return err
	}

	return errors.NewFromBaseError(err, errors.ErrorType_INTERNAL, "registration repository")
}

func (r *KVRegistrationRepository) IncrementId(ctx context.Context) (uint64, error) {
	return r.counter.GetNextAndIncrement(ctx)
}

func (r *KVRegistrationRepository) Set(ctx context.Context, reg *types.RegistrationsCommitment) error {
	r.table.Set(ctx, []byte(reg.Id), *reg)
	err := r.store.SetObject(reg)
	return r.wrapErr(err)
}

func (r *KVRegistrationRepository) GetById(ctx context.Context, id string) (*types.RegistrationsCommitment, error) {
	opt, err := r.table.Get(ctx, []byte(id))
	if err != nil {
		return nil, r.wrapErr(err)
	}
	if opt.Empty() {
		return nil, nil
	}
	reg := opt.GetValue()
	return &reg, nil
}

func (r *KVRegistrationRepository) FilterByCommitment(ctx context.Context, commitment []byte) ([]types.RegistrationsCommitment, error) {
	keyIter, err := r.commIndex.IterateKeys(ctx, &commitment, stores.NewOpenIterator())
	if err != nil {
		return nil, err
	}
	iter := table.MaterializeObjects(ctx, &r.table, keyIter)
	vals, errs := iterator.Consume(ctx, iter)
	if err != nil {
		return nil, errs[0]
	}
	return vals, nil
}

func (r *KVRegistrationRepository) GetExpiredCommitments(ctx context.Context, now *types.Timestamp) ([]types.RegistrationsCommitment, error) {
	var end []byte // todo marshal now
	param := store.NewBoundIterator(nil, end)
	keysIter, err := r.timeIdx.Iterate(ctx, param)

	iter := table.MaterializeObjects(ctx, &r.table, keysIter)
	vals, errs := iterator.Consume(ctx, iter)
	if err != nil {
		return nil, errs[0]
	}
	return vals, nil
}
