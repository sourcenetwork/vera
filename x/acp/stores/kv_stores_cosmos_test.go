package stores

import (
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/stretchr/testify/require"
)

func setupTestStore(t *testing.T) storetypes.KVStore {
	storeKey := storetypes.NewKVStoreKey("test")
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeDB, db)
	require.NoError(t, stateStore.LoadLatestVersion())
	return stateStore.GetKVStore(storeKey)
}

func TestRaccoonKVFromCosmos(t *testing.T) {
	cosmosStore := setupTestStore(t)
	wrapped := RaccoonKVFromCosmos(cosmosStore)
	require.NotNil(t, wrapped)
}

func TestCosmosKvWrapperSetGet(t *testing.T) {
	cosmosStore := setupTestStore(t)
	kv := RaccoonKVFromCosmos(cosmosStore)

	err := kv.Set([]byte("key1"), []byte("value1"))
	require.NoError(t, err)

	val, err := kv.Get([]byte("key1"))
	require.NoError(t, err)
	require.Equal(t, []byte("value1"), val)
}

func TestCosmosKvWrapperHas(t *testing.T) {
	cosmosStore := setupTestStore(t)
	kv := RaccoonKVFromCosmos(cosmosStore)

	has, err := kv.Has([]byte("missing"))
	require.NoError(t, err)
	require.False(t, has)

	err = kv.Set([]byte("exists"), []byte("val"))
	require.NoError(t, err)

	has, err = kv.Has([]byte("exists"))
	require.NoError(t, err)
	require.True(t, has)
}

func TestCosmosKvWrapperDelete(t *testing.T) {
	cosmosStore := setupTestStore(t)
	kv := RaccoonKVFromCosmos(cosmosStore)

	err := kv.Set([]byte("key"), []byte("val"))
	require.NoError(t, err)

	err = kv.Delete([]byte("key"))
	require.NoError(t, err)

	has, err := kv.Has([]byte("key"))
	require.NoError(t, err)
	require.False(t, has)
}

func TestCosmosKvWrapperGetMissing(t *testing.T) {
	cosmosStore := setupTestStore(t)
	kv := RaccoonKVFromCosmos(cosmosStore)

	val, err := kv.Get([]byte("nonexistent"))
	require.NoError(t, err)
	require.Nil(t, val)
}

func TestCosmosKvWrapperIterator(t *testing.T) {
	cosmosStore := setupTestStore(t)
	kv := RaccoonKVFromCosmos(cosmosStore)

	err := kv.Set([]byte("a"), []byte("1"))
	require.NoError(t, err)
	err = kv.Set([]byte("b"), []byte("2"))
	require.NoError(t, err)
	err = kv.Set([]byte("c"), []byte("3"))
	require.NoError(t, err)

	iter := kv.Iterator([]byte("a"), []byte("d"))
	defer iter.Close()

	count := 0
	for ; iter.Valid(); iter.Next() {
		count++
	}
	require.Equal(t, 3, count)
}
