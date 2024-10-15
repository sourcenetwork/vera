package registration

import (
	"bytes"
	"context"

	"cosmossdk.io/store/prefix"

	storetypes "cosmossdk.io/store/types"
	"github.com/sourcenetwork/acp_core/pkg/errors"
	raccoon "github.com/sourcenetwork/raccoondb"
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
	store   raccoon.ObjectStore[*types.RegistrationsCommitment]
	counter raccoon.CounterStore
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
	err := r.store.SetObject(reg)
	return r.wrapErr(err)
}

func (r *KVRegistrationRepository) GetById(ctx context.Context, id string) (*types.RegistrationsCommitment, error) {
	opt, err := r.store.GetObject([]byte(id))
	if err != nil {
		return nil, r.wrapErr(err)
	}
	if opt.IsEmpty() {
		return nil, nil
	}
	return opt.Value(), nil
}

func (r *KVRegistrationRepository) FilterByCommitment(ctx context.Context, commitment []byte) ([]*types.RegistrationsCommitment, error) {
	// FIXME update raccoon store schema
	recs, err := r.store.Filter(func(rc *types.RegistrationsCommitment) bool {
		return bytes.Equal(rc.Commitment, commitment)
	})
	if err != nil {
		return nil, r.wrapErr(err)
	}
	return recs, nil
}

func (r *KVRegistrationRepository) GetExpiredCommitments(ctx context.Context, now *types.Timestamp) ([]*types.RegistrationsCommitment, error) {
	var filterErr error = nil
	records, err := r.store.Filter(func(c *types.RegistrationsCommitment) bool {
		expired, err := types.IsAfter(c.CreationTs, c.Validity, now)
		if err != nil {
			filterErr = errors.NewFromBaseError(err, errors.ErrorType_INTERNAL,
				"comparing timestamp for commitment failed",
				errors.Pair("commitment", c.Id),
			)
		}
		return expired
	})
	if filterErr != nil {
		return nil, filterErr
	}
	if err != nil {
		return nil, err
	}
	return records, nil
}
