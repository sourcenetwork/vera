package cosmos

import (
	"context"

	corestore "cosmossdk.io/core/store"
	cosmosdb "github.com/cosmos/cosmos-db"
	"github.com/sourcenetwork/raccoondb/v2/store"
	"github.com/sourcenetwork/raccoondb/v2/types"
)

var _ store.KVStore = (*kvAdapter)(nil)

// NewFromCoreKVStore returns a Raccoon KVStore from
// cosmossdk.io/core/store KVStore
func NewFromCoreKVStore(cosmosKV corestore.KVStore) store.KVStore {
	return &kvAdapter{
		store: cosmosKV,
	}
}

// NewFromCosmosDBStore returns a Raccoon KVStore from
// a cosmos-db DB
func NewFromCosmosDB(db cosmosdb.DB) store.KVStore {
	return &kvAdapter{
		store: db,
	}
}

// kvAdapter implements Raccoon KVStore interface
type kvAdapter struct {
	store corestore.KVStore
}

func (k *kvAdapter) Iterate(ctx context.Context, opt store.IterationParam) (store.StoreIterator[[]byte], error) {
	var iter corestore.Iterator
	var err error
	if opt.IsReverse() {
		iter, err = k.store.ReverseIterator(opt.GetLeftBound(), opt.GetRightBound())
	} else {
		iter, err = k.store.Iterator(opt.GetLeftBound(), opt.GetRightBound())
	}
	if err != nil {
		return nil, wrapErr(err)
	}

	if iter.Error() != nil {
		return nil, wrapErr(iter.Error())
	}

	return &iterAdapter{
		iter:     iter,
		finished: false,
		params:   opt,
	}, nil
}

func (k *kvAdapter) Get(ctx context.Context, key []byte) (types.Option[[]byte], error) {
	if key == nil {
		return types.None[[]byte](), wrapErr(store.ErrKeyNil)
	}

	bytes, err := k.store.Get(key)
	if err != nil {
		return types.None[[]byte](), wrapErr(err)
	}
	if bytes == nil {
		return types.None[[]byte](), nil
	}
	return types.Some(bytes), nil

}

func (k *kvAdapter) Has(ctx context.Context, key []byte) (bool, error) {
	if key == nil {
		return false, wrapErr(store.ErrKeyNil)
	}

	has, err := k.store.Has(key)
	if err != nil {
		return false, wrapErr(err)
	}
	return has, nil
}

func (k *kvAdapter) Set(ctx context.Context, key, value []byte) (store.KeyCreated, error) {
	if key == nil {
		return false, wrapErr(store.ErrKeyNil)
	}

	has, err := k.store.Has(key)
	if err != nil {
		return false, wrapErr(err)
	}

	err = k.store.Set(key, value)
	if err != nil {
		return false, wrapErr(err)
	}
	return store.KeyCreated(!has), nil
}

func (k *kvAdapter) Delete(ctx context.Context, key []byte) (store.KeyRemoved, error) {
	if key == nil {
		return false, wrapErr(store.ErrKeyNil)
	}

	has, err := k.store.Has(key)
	if err != nil {
		return false, wrapErr(err)
	}

	err = k.store.Delete(key)
	if err != nil {
		return false, wrapErr(err)
	}
	return store.KeyRemoved(has), nil
}
