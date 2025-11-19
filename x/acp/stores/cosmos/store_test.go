package cosmos

import (
	"testing"

	cosmosdb "github.com/cosmos/cosmos-db"

	"github.com/sourcenetwork/raccoondb/v2/store"
	"github.com/sourcenetwork/raccoondb/v2/store/test"
)

func Test_CosmosCoreKVStore_Suite(t *testing.T) {
	factory := func() store.KVStore {
		db := cosmosdb.NewMemDB()
		return NewFromCoreKVStore(db)
	}
	test.RunSuite(t, factory)
}

func Test_CosmosDB_Suite(t *testing.T) {
	factory := func() store.KVStore {
		db := cosmosdb.NewMemDB()
		return NewFromCosmosDB(db)
	}
	test.RunSuite(t, factory)
}
