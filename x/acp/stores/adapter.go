package stores

import (
	"context"
	"fmt"

	cosmosstore "cosmossdk.io/core/store"
	"github.com/sourcenetwork/raccoondb/v2/errors"
	"github.com/sourcenetwork/raccoondb/v2/store"
	"github.com/sourcenetwork/raccoondb/v2/types"
)

var _ store.KVStore = (*kvAdapter)(nil)

var ErrCosmosKV = errors.New("cosmossdk store")

func NewRaccoonKV(cosmosKV cosmosstore.KVStore) store.KVStore {
	return &kvAdapter{
		db: cosmosKV,
	}
}

// wrapErr wraps an error with ErrCosmosKV
func wrapErr(err error) error {
	return fmt.Errorf("%w: %w", ErrCosmosKV, err)
}

type kvAdapter struct {
	db cosmosstore.KVStore
}

func (k *kvAdapter) Iterate(ctx context.Context, opt store.IterationParam) (store.StoreIterator[[]byte], error) {
	var iter cosmosstore.Iterator
	var err error
	if opt.IsReverse() {
		iter, err = k.db.ReverseIterator(opt.GetLeftBound(), opt.GetRightBound())
	} else {
		iter, err = k.db.Iterator(opt.GetLeftBound(), opt.GetRightBound())
	}

	if err != nil {
		return nil, wrapErr(err)
	}
	wrapped := &iterAdapter{
		iter:        iter,
		initialized: false,
		finished:    false,
		params:      opt,
	}

	return wrapped, nil
}

func (k *kvAdapter) Get(ctx context.Context, key []byte) (types.Option[[]byte], error) {
	if key == nil {
		return types.None[[]byte](), wrapErr(store.ErrKeyNil)
	}

	bytes, err := k.db.Get(key)
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

	has, err := k.db.Has(key)
	if err != nil {
		return false, wrapErr(err)
	}
	return has, nil
}

func (k *kvAdapter) Set(ctx context.Context, key, value []byte) (store.KeyCreated, error) {
	if key == nil {
		return false, wrapErr(store.ErrKeyNil)
	}

	has, err := k.db.Has(key)
	if err != nil {
		return false, wrapErr(err)
	}

	err = k.db.Set(key, value)
	if err != nil {
		return false, wrapErr(err)
	}
	return store.KeyCreated(!has), nil
}

func (k *kvAdapter) Delete(ctx context.Context, key []byte) (store.KeyRemoved, error) {
	if key == nil {
		return false, wrapErr(store.ErrKeyNil)
	}

	has, err := k.db.Has(key)
	if err != nil {
		return false, wrapErr(err)
	}

	err = k.db.Delete(key)
	if err != nil {
		return false, wrapErr(err)
	}
	return store.KeyRemoved(has), nil
}

type iterAdapter struct {
	iter        cosmosstore.Iterator
	params      store.IterationParam
	initialized bool
	finished    bool
}

func (i *iterAdapter) Next(ctx context.Context) error {
	if !i.initialized {
		i.initialized = true
		// cometbft-db's iterator is created ready to use (yields first value right away)
		// as such it may have an error set during creation
		// if it fails to yield the first value, therefore we check for it
		if i.iter.Error() != nil {
			return wrapErr(i.iter.Error())
		}
		return nil
	}

	i.iter.Next()
	err := i.iter.Error()
	if !i.iter.Valid() {
		i.finished = true
	}

	if err != nil {
		return wrapErr(err)
	}
	return nil
}

func (i *iterAdapter) Value() types.Option[[]byte] {
	if i.finished || !i.initialized {
		return types.None[[]byte]()
	}
	return types.Some(i.iter.Value())
}

func (i *iterAdapter) Finished() bool {
	return i.finished
}

func (i *iterAdapter) Close() error {
	err := i.iter.Close()
	if err != nil {
		return wrapErr(err)
	}
	return nil
}

func (i *iterAdapter) GetParams() store.IterationParam {
	return i.params
}

func (i *iterAdapter) CurrentKey() []byte {
	if i.finished || !i.initialized {
		return nil
	}
	return i.iter.Key()
}
