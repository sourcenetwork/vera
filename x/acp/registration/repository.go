package registration

import (
	"bytes"
	"context"

	storetypes "cosmossdk.io/store/types"
	"github.com/sourcenetwork/acp_core/pkg/errors"
	raccoon "github.com/sourcenetwork/raccoondb"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

var _ raccoon.Ider[*types.RegistrationsCommitment] = (*registrationIder)(nil)

type registrationIder struct{}

func (i *registrationIder) Id(obj *types.RegistrationsCommitment) []byte {
	return []byte(obj.Id)
}

var _ RegistrationsRepository = (*KVRegistrationRepository)(nil)

type KVRegistrationRepository struct {
	kv    storetypes.KVStore
	store raccoon.ObjectStore[*types.RegistrationsCommitment]
}

func (r *KVRegistrationRepository) wrapErr(err error) error {
	if err == nil {
		return err
	}

	return errors.NewFromBaseError(err, errors.ErrorType_INTERNAL, "registration repository")
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
