package access_decision

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	"github.com/sourcenetwork/acp_core/pkg/errors"
	raccoon "github.com/sourcenetwork/raccoondb"

	"github.com/sourcenetwork/sourcehub/utils"
	"github.com/sourcenetwork/sourcehub/x/acp/stores"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

type AccessDecisionRepository struct {
	kv storetypes.KVStore
}

func NewAccessDecisionRepository(store storetypes.KVStore) *AccessDecisionRepository {
	return &AccessDecisionRepository{
		kv: store,
	}
}

func (r *AccessDecisionRepository) getStore(_ context.Context) raccoon.ObjectStore[*types.AccessDecision] {
	rcKV := stores.RaccoonKVFromCosmos(r.kv)
	marshaler := stores.NewGogoProtoMarshaler(func() *types.AccessDecision { return &types.AccessDecision{} })
	ider := &decisionIder{}
	return raccoon.NewObjStore(rcKV, marshaler, ider)
}

func (r *AccessDecisionRepository) wrapErr(err error) error {
	if err == nil {
		return err
	}

	return errors.New(err.Error(), errors.ErrorType_INTERNAL)
}

func (r *AccessDecisionRepository) Set(ctx context.Context, decision *types.AccessDecision) error {
	store := r.getStore(ctx)
	err := store.SetObject(decision)
	return r.wrapErr(err)
}

func (r *AccessDecisionRepository) Get(ctx context.Context, id string) (*types.AccessDecision, error) {
	store := r.getStore(ctx)
	opt, err := store.GetObject([]byte(id))
	var obj *types.AccessDecision
	if !opt.IsEmpty() {
		obj = opt.Value()
	}
	return obj, r.wrapErr(err)
}

func (r *AccessDecisionRepository) Delete(ctx context.Context, id string) error {
	store := r.getStore(ctx)
	err := store.DeleteById([]byte(id))
	return r.wrapErr(err)
}

func (r *AccessDecisionRepository) ListIds(ctx context.Context) ([]string, error) {
	store := r.getStore(ctx)
	bytesIds, err := store.ListIds()
	ids := utils.MapSlice(bytesIds, func(bytes []byte) string { return string(bytes) })
	return ids, r.wrapErr(err)
}

func (r *AccessDecisionRepository) List(ctx context.Context) ([]*types.AccessDecision, error) {
	store := r.getStore(ctx)
	objs, err := store.List()
	return objs, r.wrapErr(err)
}

type decisionIder struct{}

func (i *decisionIder) Id(decision *types.AccessDecision) []byte {
	return []byte(decision.Id)
}
