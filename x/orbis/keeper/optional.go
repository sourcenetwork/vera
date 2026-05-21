package keeper

import "github.com/sourcenetwork/sourcehub/x/orbis/types"

func optionalCreateRingPSSInterval(msg *types.MsgCreateRing) *uint64 {
	if msg.XPssInterval == nil {
		return nil
	}
	value := msg.GetPssInterval()
	return &value
}

func optionalUpdateRingNewThreshold(msg *types.MsgUpdateRingByAcp) *uint32 {
	if msg.XNewThreshold == nil {
		return nil
	}
	value := msg.GetNewThreshold()
	return &value
}

func optionalUpdateRingPSSInterval(msg *types.MsgUpdateRingByAcp) *uint64 {
	if msg.XPssInterval == nil {
		return nil
	}
	value := msg.GetPssInterval()
	return &value
}

func optionalStoreDocumentTier(msg *types.MsgStoreDocument) *string {
	if msg.XTier == nil {
		return nil
	}
	value := msg.GetTier()
	return &value
}

func optionalStoreDocumentTimestamp(msg *types.MsgStoreDocument) *uint64 {
	if msg.XTimestamp == nil {
		return nil
	}
	value := msg.GetTimestamp()
	return &value
}

func setRingNewThreshold(ring *types.Ring, value *uint32) {
	if value == nil {
		ring.XNewThreshold = nil
		return
	}
	ring.XNewThreshold = &types.Ring_NewThreshold{NewThreshold: *value}
}

func setRingPSSInterval(ring *types.Ring, value *uint64) {
	if value == nil {
		ring.XPssInterval = nil
		return
	}
	ring.XPssInterval = &types.Ring_PssInterval{PssInterval: *value}
}

func setDocumentTier(document *types.Document, value *string) {
	if value == nil {
		document.XTier = nil
		return
	}
	document.XTier = &types.Document_Tier{Tier: *value}
}

func setDocumentTimestamp(document *types.Document, value *uint64) {
	if value == nil {
		document.XTimestamp = nil
		return
	}
	document.XTimestamp = &types.Document_Timestamp{Timestamp: *value}
}
