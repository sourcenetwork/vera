package types

import (
	"crypto/sha256"
	"encoding/hex"
)

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

// HashJWSToken returns the SHA256 hash of a JWS token string.
func HashJWSToken(jwsToken string) string {
	hash := sha256.Sum256([]byte(jwsToken))
	return hex.EncodeToString(hash[:])
}

// JWSTokenKey returns the store key for a JWS token by its hash.
func JWSTokenKey(tokenHash string) []byte {
	return append([]byte(JWSTokenKeyPrefix), []byte(tokenHash)...)
}

// JWSTokenByDIDKey returns the store key for a JWS token indexed by DID and hash.
func JWSTokenByDIDKey(did, tokenHash string) []byte {
	didKey := append([]byte(JWSTokenByDIDKeyPrefix), []byte(did)...)
	didKey = append(didKey, '/')
	return append(didKey, []byte(tokenHash)...)
}

// JWSTokenByAccountKey returns the store key for a JWS token indexed by account and hash.
func JWSTokenByAccountKey(account, tokenHash string) []byte {
	accountKey := append([]byte(JWSTokenByAccountKeyPrefix), []byte(account)...)
	accountKey = append(accountKey, '/')
	return append(accountKey, []byte(tokenHash)...)
}

// JWSTokenDIDPrefix returns the prefix for all tokens belonging to a DID.
func JWSTokenDIDPrefix(did string) []byte {
	prefix := append([]byte(JWSTokenByDIDKeyPrefix), []byte(did)...)
	return append(prefix, '/')
}

// JWSTokenAccountPrefix returns the prefix for all tokens belonging to an account.
func JWSTokenAccountPrefix(account string) []byte {
	prefix := append([]byte(JWSTokenByAccountKeyPrefix), []byte(account)...)
	return append(prefix, '/')
}
