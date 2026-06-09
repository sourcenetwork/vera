package keeper

import (
	"github.com/sourcenetwork/immutable"

	"github.com/sourcenetwork/sourcehub/x/orbis/types"
)

func optionalCreateRingPSSInterval(msg *types.MsgCreateRing) immutable.Option[uint64] {
	if msg.XPssInterval == nil {
		return immutable.None[uint64]()
	}
	value := msg.GetPssInterval()
	return immutable.Some(value)
}

func optionalCreateRingNonce(msg *types.MsgCreateRing) immutable.Option[string] {
	if msg.XNonce == nil {
		return immutable.None[string]()
	}
	value := msg.GetNonce()
	return immutable.Some(value)
}

func optionalUpdateRingNewThreshold(msg *types.MsgUpdateRingByAcp) immutable.Option[uint32] {
	if msg.XNewThreshold == nil {
		return immutable.None[uint32]()
	}
	value := msg.GetNewThreshold()
	return immutable.Some(value)
}

func optionalUpdateRingPSSInterval(msg *types.MsgUpdateRingByAcp) immutable.Option[uint64] {
	if msg.XPssInterval == nil {
		return immutable.None[uint64]()
	}
	value := msg.GetPssInterval()
	return immutable.Some(value)
}

func optionalUpdateRingNextVersion(msg *types.MsgUpdateRingByAcp) immutable.Option[uint64] {
	if msg.XNextVersion == nil {
		return immutable.None[uint64]()
	}
	return immutable.Some(msg.GetNextVersion())
}

func optionalUpdateRingActivationHeight(msg *types.MsgUpdateRingByAcp) immutable.Option[int64] {
	if msg.XActivationHeight == nil {
		return immutable.None[int64]()
	}
	return immutable.Some(msg.GetActivationHeight())
}

func optionalStoreDocumentTier(msg *types.MsgStoreDocument) immutable.Option[string] {
	if msg.XTier == nil {
		return immutable.None[string]()
	}
	value := msg.GetTier()
	return immutable.Some(value)
}

func optionalStoreDocumentTimestamp(msg *types.MsgStoreDocument) immutable.Option[uint64] {
	if msg.XTimestamp == nil {
		return immutable.None[uint64]()
	}
	value := msg.GetTimestamp()
	return immutable.Some(value)
}

func setRingNewThreshold(ring *types.Ring, value immutable.Option[uint32]) {
	if !value.HasValue() {
		ring.XNewThreshold = nil
		return
	}
	ring.XNewThreshold = &types.Ring_NewThreshold{NewThreshold: value.Value()}
}

func setRingPSSInterval(ring *types.Ring, value immutable.Option[uint64]) {
	if !value.HasValue() {
		ring.XPssInterval = nil
		return
	}
	ring.XPssInterval = &types.Ring_PssInterval{PssInterval: value.Value()}
}

func setDocumentTier(document *types.Document, value immutable.Option[string]) {
	if !value.HasValue() {
		document.XTier = nil
		return
	}
	document.XTier = &types.Document_Tier{Tier: value.Value()}
}

func setDocumentTimestamp(document *types.Document, value immutable.Option[uint64]) {
	if !value.HasValue() {
		document.XTimestamp = nil
		return
	}
	document.XTimestamp = &types.Document_Timestamp{Timestamp: value.Value()}
}
