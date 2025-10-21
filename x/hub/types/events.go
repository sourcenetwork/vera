package types

// JWS token event types
const (
	EventTypeInvalidateJWS = "invalidate_jws"
	EventTypeJWSTokenUsed  = "jws_token_used"

	AttributeKeyTokenHash         = "token_hash"
	AttributeKeyInvalidatedBy     = "invalidated_by"
	AttributeKeyIssuerDID         = "issuer_did"
	AttributeKeyAuthorizedAccount = "authorized_account"
)
