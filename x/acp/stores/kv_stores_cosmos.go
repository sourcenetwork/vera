package stores

import (
	storetypes "cosmossdk.io/store/types"
	rcdb "github.com/sourcenetwork/raccoondb"
)

// RaccoonKVFromCosmos adapts a cosmossdk KVStore
// into a raccondb v1 KVStore
func RaccoonKVFromCosmos(store storetypes.KVStore) rcdb.KVStore {
	return &cosmosKvWrapper{
		store: store,
	}
}

type cosmosKvWrapper struct {
	store storetypes.KVStore
}

func (s *cosmosKvWrapper) Get(key []byte) ([]byte, error) {
	return s.store.Get(key), nil
}

func (s *cosmosKvWrapper) Has(key []byte) (bool, error) {
	return s.store.Has(key), nil
}

func (s *cosmosKvWrapper) Set(key []byte, val []byte) error {
	s.store.Set(key, val)
	return nil
}

func (s *cosmosKvWrapper) Delete(key []byte) error {
	s.store.Delete(key)
	return nil
}

func (s *cosmosKvWrapper) Iterator(start, end []byte) rcdb.Iterator {
	return s.store.Iterator(start, end)
}
