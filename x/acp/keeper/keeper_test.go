package keeper

import (
	"encoding/binary"
	"testing"

	"github.com/sourcenetwork/raccoondb/v2/primitives"
	"github.com/stretchr/testify/require"

	cosmosadapter "github.com/sourcenetwork/vera/x/acp/stores/cosmos"
	"github.com/sourcenetwork/vera/x/acp/types"
)

// Test that an expired replay key is lazily removed and does not block resubmission.
func TestHasSeenSignedPolicyCmd_ExpiredKeyIsPruned(t *testing.T) {
	ctx, k, _ := setupKeeper(t)
	ctx = ctx.WithBlockHeight(10)

	payloadID := []byte("test-id")
	current := uint64(ctx.BlockHeight())
	cmtkv := k.storeService.OpenKVStore(ctx)
	kv := cosmosadapter.NewFromCoreKVStore(cmtkv)
	kv = primitives.NewPrefixedKV(kv, []byte(types.SignedPolicyCmdSeenKeyPrefix))

	var bz [8]byte
	binary.BigEndian.PutUint64(bz[:], uint64(current-1))
	kv.Set(ctx, payloadID, bz[:])

	seen := k.hasSeenSignedPolicyCmd(ctx, payloadID, uint64(ctx.BlockHeight()))
	require.False(t, seen)

	has, err := kv.Has(ctx, payloadID)
	require.NoError(t, err)
	require.False(t, has)
}

// Test that malformed stored values are deleted and treated as unseen.
func TestHasSeenSignedPolicyCmd_MalformedValueDeleted(t *testing.T) {
	ctx, k, _ := setupKeeper(t)

	payloadID := []byte("test-id")
	cmtkv := k.storeService.OpenKVStore(ctx)
	kv := cosmosadapter.NewFromCoreKVStore(cmtkv)
	kv = primitives.NewPrefixedKV(kv, []byte(types.SignedPolicyCmdSeenKeyPrefix))
	kv.Set(ctx, payloadID, []byte{0x01})

	seen := k.hasSeenSignedPolicyCmd(ctx, payloadID, uint64(ctx.BlockHeight()))
	require.False(t, seen)

	has, err := kv.Has(ctx, payloadID)
	require.NoError(t, err)
	require.False(t, has)
}

// Test that markSignedPolicyCmdSeen stores the expiration and prevents duplicate marks.
func TestMarkSignedPolicyCmdSeen_StoresAndBlocksReplay(t *testing.T) {
	ctx, k, _ := setupKeeper(t)
	ctx = ctx.WithBlockHeight(10)

	payloadID := []byte("test-id")
	expire := uint64(25)

	err := k.markSignedPolicyCmdSeen(ctx, payloadID, expire)
	require.NoError(t, err)

	seen := k.hasSeenSignedPolicyCmd(ctx, payloadID, uint64(ctx.BlockHeight()))
	require.True(t, seen)

	cmtkv := k.storeService.OpenKVStore(ctx)
	kv := cosmosadapter.NewFromCoreKVStore(cmtkv)
	kv = primitives.NewPrefixedKV(kv, []byte(types.SignedPolicyCmdSeenKeyPrefix))
	opt, _ := kv.Get(ctx, payloadID)
	require.False(t, opt.Empty())

	bz := opt.GetValue()
	require.Len(t, bz, 8)

	got := binary.BigEndian.Uint64(bz)
	require.Equal(t, expire, got)

	err = k.markSignedPolicyCmdSeen(ctx, payloadID, expire)
	require.Error(t, err)
}

// Test that marking with an already expired height results in immediate prune on check.
func TestMarkSignedPolicyCmdSeen_ExpiredImmediatelyPruned(t *testing.T) {
	ctx, k, _ := setupKeeper(t)
	ctx = ctx.WithBlockHeight(10)

	payloadID := []byte("test-id")
	expire := uint64(ctx.BlockHeight() - 1)
	err := k.markSignedPolicyCmdSeen(ctx, payloadID, expire)
	require.NoError(t, err)

	seen := k.hasSeenSignedPolicyCmd(ctx, payloadID, uint64(ctx.BlockHeight()))
	require.False(t, seen)

	cmtkv := k.storeService.OpenKVStore(ctx)
	kv := cosmosadapter.NewFromCoreKVStore(cmtkv)
	kv = primitives.NewPrefixedKV(kv, []byte(types.SignedPolicyCmdSeenKeyPrefix))
	has, err := kv.Has(ctx, payloadID)
	require.NoError(t, err)
	require.False(t, has)
}
