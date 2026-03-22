package types

import (
	"testing"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	sdk "github.com/cosmos/cosmos-sdk/types"
	prototypes "github.com/cosmos/gogoproto/types"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/stretchr/testify/require"
)

func makeTestContext(t *testing.T) sdk.Context {
	storeKey := storetypes.NewKVStoreKey("test")
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeDB, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	blockTime := time.Date(2024, time.June, 1, 12, 0, 0, 0, time.UTC)
	ctx := sdk.NewContext(stateStore, cmtproto.Header{
		Height: 42,
		Time:   blockTime,
	}, false, log.NewNopLogger())
	return ctx
}

func TestBuildRecordMetadata(t *testing.T) {
	ctx := makeTestContext(t)
	md, err := BuildRecordMetadata(ctx, "did:example:actor", "cosmos1creator")
	require.NoError(t, err)
	require.NotNil(t, md)
	require.Equal(t, "did:example:actor", md.OwnerDid)
	require.Equal(t, "cosmos1creator", md.TxSigner)
	require.NotNil(t, md.CreationTs)
	require.Equal(t, uint64(42), md.CreationTs.BlockHeight)
}

func TestBuildACPSuppliedMetadata(t *testing.T) {
	ctx := makeTestContext(t)
	sm, err := BuildACPSuppliedMetadata(ctx, "did:example:actor", "cosmos1creator")
	require.NoError(t, err)
	require.NotNil(t, sm)
	require.NotEmpty(t, sm.Blob)
}

func TestBuildACPSuppliedMetadataWithTime(t *testing.T) {
	ctx := makeTestContext(t)
	protoTs, err := prototypes.TimestampProto(time.Date(2024, time.March, 1, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	ts := NewTimestamp(protoTs, 50)

	sm, err := BuildACPSuppliedMetadataWithTime(ctx, ts, "did:example:actor", "cosmos1creator")
	require.NoError(t, err)
	require.NotNil(t, sm)
	require.NotEmpty(t, sm.Blob)
}

func TestExtractRecordMetadata(t *testing.T) {
	ctx := makeTestContext(t)
	original, err := BuildRecordMetadata(ctx, "did:example:actor", "cosmos1creator")
	require.NoError(t, err)

	blob, err := original.Marshal()
	require.NoError(t, err)

	coreMd := &coretypes.RecordMetadata{
		Supplied: &coretypes.SuppliedMetadata{Blob: blob},
	}
	extracted, err := ExtractRecordMetadata(coreMd)
	require.NoError(t, err)
	require.Equal(t, original.OwnerDid, extracted.OwnerDid)
	require.Equal(t, original.TxSigner, extracted.TxSigner)
}

func TestExtractRecordMetadataInvalidBlob(t *testing.T) {
	coreMd := &coretypes.RecordMetadata{
		Supplied: &coretypes.SuppliedMetadata{Blob: []byte("not-valid-protobuf")},
	}
	_, err := ExtractRecordMetadata(coreMd)
	require.Error(t, err)
}
