package keeper

import (
	"github.com/sourcenetwork/immutable"

	"github.com/sourcenetwork/vera/x/orbis/types"
)

func optionalCreateRingNonce(msg *types.MsgCreateRing) immutable.Option[string] {
	if msg.XNonce == nil {
		return immutable.None[string]()
	}
	value := msg.GetNonce()
	return immutable.Some(value)
}

func optionalStartRingReshareNewThreshold(msg *types.MsgStartRingReshareByAcp) immutable.Option[uint32] {
	if msg.XNewThreshold == nil {
		return immutable.None[uint32]()
	}
	value := msg.GetNewThreshold()
	return immutable.Some(value)
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
