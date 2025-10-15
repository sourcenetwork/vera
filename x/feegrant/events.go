package feegrant

// evidence module events
const (
	EventTypeUseFeeGrant       = "use_feegrant"
	EventTypeUseDIDFeeGrant    = "use_did_feegrant"
	EventTypeRevokeFeeGrant    = "revoke_feegrant"
	EventTypeRevokeDIDFeeGrant = "revoke_did_feegrant"
	EventTypeSetFeeGrant       = "set_feegrant"
	EventTypeSetDIDFeeGrant    = "set_did_feegrant"
	EventTypeUpdateFeeGrant    = "update_feegrant"
	EventTypeUpdateDIDFeeGrant = "update_did_feegrant"
	EventTypePruneFeeGrant     = "prune_feegrant"
	EventTypePruneDIDFeeGrant  = "prune_did_feegrant"

	AttributeKeyGranter    = "granter"
	AttributeKeyGrantee    = "grantee"
	AttributeKeyGranteeDid = "grantee_did"
	AttributeKeyPruner     = "pruner"
)
