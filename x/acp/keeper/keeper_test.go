package keeper

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"

	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

// Test that an expired replay key is lazily removed and does not block resubmission.
func TestHasSeenSignedPolicyCmd_ExpiredKeyIsPruned(t *testing.T) {
	ctx, k, _ := setupKeeper(t)
	ctx = ctx.WithBlockHeight(10)

	id := "test-id"
	current := uint64(ctx.BlockHeight())
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	pref := prefix.NewStore(store, types.KeyPrefix(types.SignedPolicyCmdSeenKeyPrefix))
	var bz [8]byte
	binary.BigEndian.PutUint64(bz[:], uint64(current-1))
	pref.Set([]byte(id), bz[:])

	seen := k.hasSeenSignedPolicyCmd(ctx, id, uint64(ctx.BlockHeight()))
	require.False(t, seen)
	require.False(t, pref.Has([]byte(id)))
}

// Test that malformed stored values are deleted and treated as unseen.
func TestHasSeenSignedPolicyCmd_MalformedValueDeleted(t *testing.T) {
	ctx, k, _ := setupKeeper(t)

	id := "bad-id"
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	pref := prefix.NewStore(store, types.KeyPrefix(types.SignedPolicyCmdSeenKeyPrefix))
	pref.Set([]byte(id), []byte{0x01})

	seen := k.hasSeenSignedPolicyCmd(ctx, id, uint64(ctx.BlockHeight()))
	require.False(t, seen)
	require.False(t, pref.Has([]byte(id)))
}

// Test that markSignedPolicyCmdSeen stores the expiration and prevents duplicate marks.
func TestMarkSignedPolicyCmdSeen_StoresAndBlocksReplay(t *testing.T) {
	ctx, k, _ := setupKeeper(t)
	ctx = ctx.WithBlockHeight(10)

	id := "mark-id"
	expire := uint64(25)

	err := k.markSignedPolicyCmdSeen(ctx, id, expire)
	require.NoError(t, err)

	seen := k.hasSeenSignedPolicyCmd(ctx, id, uint64(ctx.BlockHeight()))
	require.True(t, seen)

	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	pref := prefix.NewStore(store, types.KeyPrefix(types.SignedPolicyCmdSeenKeyPrefix))
	bz := pref.Get([]byte(id))
	require.Len(t, bz, 8)
	got := binary.BigEndian.Uint64(bz)
	require.Equal(t, expire, got)

	err = k.markSignedPolicyCmdSeen(ctx, id, expire)
	require.Error(t, err)
}

// Test that marking with an already expired height results in immediate prune on check.
func TestMarkSignedPolicyCmdSeen_ExpiredImmediatelyPruned(t *testing.T) {
	ctx, k, _ := setupKeeper(t)
	ctx = ctx.WithBlockHeight(10)

	id := "expired-id"
	expire := uint64(ctx.BlockHeight() - 1)
	err := k.markSignedPolicyCmdSeen(ctx, id, expire)
	require.NoError(t, err)

	seen := k.hasSeenSignedPolicyCmd(ctx, id, uint64(ctx.BlockHeight()))
	require.False(t, seen)

	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	pref := prefix.NewStore(store, types.KeyPrefix(types.SignedPolicyCmdSeenKeyPrefix))
	require.False(t, pref.Has([]byte(id)))
}
