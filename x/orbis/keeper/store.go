package keeper

import (
	"context"
	"encoding/binary"
	"fmt"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"

	"github.com/sourcenetwork/sourcehub/x/orbis/types"
)

func (k *Keeper) SetRing(ctx context.Context, ring types.Ring) {
	ring.PeerNodeKeys = canonicalStrings(ring.PeerNodeKeys)
	ring.NewPeerNodeKeys = canonicalStrings(ring.NewPeerNodeKeys)
	ring.TrustedAuthRelayDids = canonicalStrings(ring.TrustedAuthRelayDids)

	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.RingKeyPrefix))
	bz := k.cdc.MustMarshal(&ring)
	store.Set([]byte(ring.Id), bz)
}

func (k *Keeper) GetRing(ctx context.Context, ringID string) *types.Ring {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.RingKeyPrefix))
	bz := store.Get([]byte(ringID))
	if bz == nil {
		return nil
	}

	var ring types.Ring
	k.cdc.MustUnmarshal(bz, &ring)
	return &ring
}

func (k *Keeper) DeleteRing(ctx context.Context, ringID string) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.RingKeyPrefix))
	store.Delete([]byte(ringID))
}

func (k *Keeper) SetAcceptedReportPair(ctx context.Context, reportID, sessionID string, expiresAt uint64) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	reportStore := prefix.NewStore(storeAdapter, types.KeyPrefix(types.AcceptedReportKeyPrefix))
	reportStore.Set([]byte(reportID), []byte{1})
	sessionStore := prefix.NewStore(storeAdapter, types.KeyPrefix(types.AcceptedReportSessionKeyPrefix))
	sessionStore.Set([]byte(sessionID), []byte{1})
	expiryStore := prefix.NewStore(storeAdapter, types.KeyPrefix(types.AcceptedReportExpiryKeyPrefix))
	expiryStore.Set(acceptedExpiryKey(expiresAt, reportID), []byte(sessionID))
}

func (k *Keeper) HasAcceptedReport(ctx context.Context, reportID string) bool {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.AcceptedReportKeyPrefix))
	return store.Has([]byte(reportID))
}

func (k *Keeper) HasAcceptedReportSession(ctx context.Context, sessionID string) bool {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.AcceptedReportSessionKeyPrefix))
	return store.Has([]byte(sessionID))
}

func (k *Keeper) DeleteExpiredAcceptedReportPairs(ctx context.Context, now uint64) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	reportStore := prefix.NewStore(storeAdapter, types.KeyPrefix(types.AcceptedReportKeyPrefix))
	sessionStore := prefix.NewStore(storeAdapter, types.KeyPrefix(types.AcceptedReportSessionKeyPrefix))
	expiryStore := prefix.NewStore(storeAdapter, types.KeyPrefix(types.AcceptedReportExpiryKeyPrefix))

	type expiredPair struct {
		expiryKey []byte
		reportID  string
		sessionID string
	}
	var expired []expiredPair

	iterator := storetypes.KVStorePrefixIterator(expiryStore, []byte{})
	for ; iterator.Valid(); iterator.Next() {
		key := iterator.Key()
		if len(key) < 8 {
			continue
		}
		if binary.BigEndian.Uint64(key[:8]) > now {
			break
		}
		expired = append(expired, expiredPair{
			expiryKey: append([]byte{}, key...),
			reportID:  string(key[8:]),
			sessionID: string(iterator.Value()),
		})
	}
	iterator.Close()

	for _, p := range expired {
		expiryStore.Delete(p.expiryKey)
		reportStore.Delete([]byte(p.reportID))
		sessionStore.Delete([]byte(p.sessionID))
	}
}

func (k *Keeper) GetAllAcceptedReportPairs(ctx context.Context) []types.AcceptedReportPairEntry {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	expiryStore := prefix.NewStore(storeAdapter, types.KeyPrefix(types.AcceptedReportExpiryKeyPrefix))

	var pairs []types.AcceptedReportPairEntry
	iterator := storetypes.KVStorePrefixIterator(expiryStore, []byte{})
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		key := iterator.Key()
		if len(key) < 8 {
			continue
		}
		pairs = append(pairs, types.AcceptedReportPairEntry{
			ReportId:  string(key[8:]),
			SessionId: string(iterator.Value()),
			ExpiresAt: binary.BigEndian.Uint64(key[:8]),
		})
	}
	return pairs
}

func acceptedExpiryKey(expiresAt uint64, id string) []byte {
	key := make([]byte, 8+len(id))
	binary.BigEndian.PutUint64(key[:8], expiresAt)
	copy(key[8:], id)
	return key
}

func (k *Keeper) GetNodeDemerits(ctx context.Context, ringID, nodeKey string) uint64 {
	state, found := k.getNodeDemeritState(ctx, ringID, nodeKey)
	if !found {
		return 0
	}
	return state.points
}

func (k *Keeper) GetEffectiveNodeDemerits(ctx context.Context, ringID, nodeKey string, now, resetIntervalSeconds uint64) uint64 {
	state, found := k.getNodeDemeritState(ctx, ringID, nodeKey)
	if !found || nodeDemeritWindowExpired(state.windowStartedAt, now, resetIntervalSeconds) {
		return 0
	}
	return state.points
}

func (k *Keeper) IncrementNodeDemerits(ctx context.Context, ringID, nodeKey string, amount, now, resetIntervalSeconds uint64) uint64 {
	state, found := k.getNodeDemeritState(ctx, ringID, nodeKey)
	if !found || nodeDemeritWindowExpired(state.windowStartedAt, now, resetIntervalSeconds) {
		state = nodeDemeritState{windowStartedAt: now}
	}
	state.points += amount
	k.setNodeDemeritState(ctx, ringID, nodeKey, state)
	return state.points
}

func (k *Keeper) SetNodeDemerits(ctx context.Context, ringID, nodeKey string, points, windowStartedAt uint64) {
	k.setNodeDemeritState(ctx, ringID, nodeKey, nodeDemeritState{
		points:          points,
		windowStartedAt: windowStartedAt,
	})
}

type nodeDemeritState struct {
	points          uint64
	windowStartedAt uint64
}

const nodeDemeritStateSize = 16

func (k *Keeper) getNodeDemeritState(ctx context.Context, ringID, nodeKey string) (nodeDemeritState, bool) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.NodeDemeritKeyPrefix))
	bz := store.Get(nodeDemeritStoreKey(ringID, nodeKey))
	if len(bz) != nodeDemeritStateSize {
		return nodeDemeritState{}, false
	}
	return nodeDemeritState{
		points:          binary.BigEndian.Uint64(bz[:8]),
		windowStartedAt: binary.BigEndian.Uint64(bz[8:]),
	}, true
}

func (k *Keeper) setNodeDemeritState(ctx context.Context, ringID, nodeKey string, state nodeDemeritState) {
	var buf [nodeDemeritStateSize]byte
	binary.BigEndian.PutUint64(buf[:8], state.points)
	binary.BigEndian.PutUint64(buf[8:], state.windowStartedAt)
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.NodeDemeritKeyPrefix))
	store.Set(nodeDemeritStoreKey(ringID, nodeKey), buf[:])
}

func nodeDemeritWindowExpired(windowStartedAt, now, resetIntervalSeconds uint64) bool {
	return now >= windowStartedAt && now-windowStartedAt >= resetIntervalSeconds
}

func nodeDemeritStoreKey(ringID, nodeKey string) []byte {
	key := make([]byte, 4+len(ringID)+len(nodeKey))
	binary.BigEndian.PutUint32(key[:4], uint32(len(ringID)))
	copy(key[4:], ringID)
	copy(key[4+len(ringID):], nodeKey)
	return key
}

func decodeNodeDemeritStoreKey(key []byte) (ringID, nodeKey string, ok bool) {
	if len(key) < 4 {
		return "", "", false
	}
	ringIDLength := int(binary.BigEndian.Uint32(key[:4]))
	if len(key) < 4+ringIDLength {
		return "", "", false
	}
	return string(key[4 : 4+ringIDLength]), string(key[4+ringIDLength:]), true
}

func (k *Keeper) GetAllNodeDemerits(ctx context.Context) []types.NodeDemeritEntry {
	var entries []types.NodeDemeritEntry
	k.mustIterateNodeDemerits(ctx, func(entry types.NodeDemeritEntry) {
		entries = append(entries, entry)
	})
	return entries
}

func (k *Keeper) GetAllRings(ctx context.Context) []types.Ring {
	var rings []types.Ring
	k.mustIterateRings(ctx, func(ring types.Ring) {
		rings = append(rings, ring)
	})
	return rings
}

func (k *Keeper) SetDocument(ctx context.Context, document types.Document) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.DocumentKeyPrefix))
	bz := k.cdc.MustMarshal(&document)
	store.Set([]byte(document.Id), bz)
}

func (k *Keeper) GetDocument(ctx context.Context, documentID string) *types.Document {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.DocumentKeyPrefix))
	bz := store.Get([]byte(documentID))
	if bz == nil {
		return nil
	}

	var document types.Document
	k.cdc.MustUnmarshal(bz, &document)
	return &document
}

func (k *Keeper) GetAllDocuments(ctx context.Context) []types.Document {
	var documents []types.Document
	k.mustIterateDocuments(ctx, func(document types.Document) {
		documents = append(documents, document)
	})
	return documents
}

func (k *Keeper) SetKeyDerivation(ctx context.Context, keyDerivation types.KeyDerivation) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.KeyDerivationKeyPrefix))
	bz := k.cdc.MustMarshal(&keyDerivation)
	store.Set([]byte(keyDerivation.Id), bz)
}

func (k *Keeper) GetKeyDerivation(ctx context.Context, keyDerivationID string) *types.KeyDerivation {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.KeyDerivationKeyPrefix))
	bz := store.Get([]byte(keyDerivationID))
	if bz == nil {
		return nil
	}

	var keyDerivation types.KeyDerivation
	k.cdc.MustUnmarshal(bz, &keyDerivation)
	return &keyDerivation
}

func (k *Keeper) GetAllKeyDerivations(ctx context.Context) []types.KeyDerivation {
	var keyDerivations []types.KeyDerivation
	k.mustIterateKeyDerivations(ctx, func(keyDerivation types.KeyDerivation) {
		keyDerivations = append(keyDerivations, keyDerivation)
	})
	return keyDerivations
}

func (k *Keeper) mustIterateRings(ctx context.Context, cb func(ring types.Ring)) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.RingKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var ring types.Ring
		k.cdc.MustUnmarshal(iterator.Value(), &ring)
		cb(ring)
	}
}

func (k *Keeper) mustIterateDocuments(ctx context.Context, cb func(document types.Document)) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.DocumentKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var document types.Document
		k.cdc.MustUnmarshal(iterator.Value(), &document)
		cb(document)
	}
}

func (k *Keeper) mustIterateKeyDerivations(ctx context.Context, cb func(keyDerivation types.KeyDerivation)) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.KeyDerivationKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var keyDerivation types.KeyDerivation
		k.cdc.MustUnmarshal(iterator.Value(), &keyDerivation)
		cb(keyDerivation)
	}
}

func (k *Keeper) mustIterateNodeDemerits(ctx context.Context, cb func(entry types.NodeDemeritEntry)) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.NodeDemeritKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		ringID, nodeKey, ok := decodeNodeDemeritStoreKey(iterator.Key())
		if !ok {
			panic(fmt.Sprintf("invalid node demerit key %x", iterator.Key()))
		}
		bz := iterator.Value()
		if len(bz) != nodeDemeritStateSize {
			panic(fmt.Sprintf("invalid node demerit state size for ring %q node %q", ringID, nodeKey))
		}
		cb(types.NodeDemeritEntry{
			RingId:          ringID,
			NodeKey:         nodeKey,
			Points:          binary.BigEndian.Uint64(bz[:8]),
			WindowStartedAt: binary.BigEndian.Uint64(bz[8:]),
		})
	}
}

func (k *Keeper) SetNodeInfo(ctx context.Context, nodeKey string, nodeInfo types.NodeInfo) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.NodeInfoKeyPrefix))
	bz := k.cdc.MustMarshal(&nodeInfo)
	store.Set([]byte(nodeKey), bz)
}

func (k *Keeper) GetNodeInfo(ctx context.Context, nodeKey string) *types.NodeInfo {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.NodeInfoKeyPrefix))
	bz := store.Get([]byte(nodeKey))
	if bz == nil {
		return nil
	}

	var nodeInfo types.NodeInfo
	k.cdc.MustUnmarshal(bz, &nodeInfo)
	return &nodeInfo
}

func (k *Keeper) GetAllNodeInfos(ctx context.Context) []types.NodeInfoEntry {
	var entries []types.NodeInfoEntry
	k.mustIterateNodeInfos(ctx, func(nodeKey string, nodeInfo types.NodeInfo) {
		entries = append(entries, types.NodeInfoEntry{NodeKey: nodeKey, NodeInfo: &nodeInfo})
	})
	return entries
}

func (k *Keeper) mustIterateNodeInfos(ctx context.Context, cb func(nodeKey string, nodeInfo types.NodeInfo)) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.NodeInfoKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var nodeInfo types.NodeInfo
		k.cdc.MustUnmarshal(iterator.Value(), &nodeInfo)
		cb(string(iterator.Key()), nodeInfo)
	}
}
