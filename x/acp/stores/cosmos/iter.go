package cosmos

import (
	"context"

	cosmosstore "cosmossdk.io/core/store"
	"github.com/sourcenetwork/raccoondb/v2/iterator"
	"github.com/sourcenetwork/raccoondb/v2/store"
)

var _ iterator.Iterator[[]byte] = (*iterAdapter)(nil)

type iterAdapter struct {
	iter     cosmosstore.Iterator
	params   store.IterationParam
	finished bool
}

func (i *iterAdapter) Next(ctx context.Context) error {
	i.iter.Next()

	if !i.iter.Valid() {
		i.finished = true
	}

	err := i.iter.Error()
	if err != nil {
		return wrapErr(err)
	}
	return nil
}

func (i *iterAdapter) Value() ([]byte, error) {
	if i.finished {
		return nil, nil
	}
	return i.iter.Value(), nil
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
	if i.finished {
		return nil
	}
	return i.iter.Key()
}
