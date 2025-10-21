package types

const (
	// ModuleName defines the module name
	ModuleName = "hub"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_hub"

	// ICAConnectionKeyPrefix defines a key prefix for ICA connections
	ICAConnectionKeyPrefix = "ica_connection/"

	// JWSTokenKeyPrefix defines a key prefix for JWS token records
	JWSTokenKeyPrefix = "jws_token/"

	// JWSTokenByDIDKeyPrefix defines a key prefix for JWS tokens indexed by DID
	JWSTokenByDIDKeyPrefix = "jws_token_by_did/"

	// JWSTokenByAccountKeyPrefix defines a key prefix for JWS tokens indexed by authorized account
	JWSTokenByAccountKeyPrefix = "jws_token_by_account/"
)

var (
	ParamsKey = []byte("p_hub")

	// AllowZeroFeeTxsKey stores an immutable flag for whether zero-fee transactions are allowed.
	// Set during genesis initialization and never changed.
	AllowZeroFeeTxsKey = []byte("app_config/allow_zero_fee_txs")

	// IgnoreBearerAuthKey stores an immutable flag for whether bearer auth should be ignored.
	// Set during genesis initialization and never changed.
	IgnoreBearerAuthKey = []byte("app_config/ignore_bearer_auth")
)
