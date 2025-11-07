package types

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
)

const (
	// ModuleName defines the module name
	ModuleName = "hub"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_hub"
)

var (
	// ICAConnectionKeyPrefix is the key prefix for ICA connections
	ICAConnectionKeyPrefix = []byte{0x00}

	// JWSTokenKeyPrefix is the key prefix for JWS token records
	JWSTokenKeyPrefix = []byte{0x01}

	// JWSTokenByDIDKeyPrefix is the key prefix for JWS tokens indexed by DID
	JWSTokenByDIDKeyPrefix = []byte{0x02}

	// JWSTokenByAccountKeyPrefix is the key prefix for JWS tokens indexed by authorized account
	JWSTokenByAccountKeyPrefix = []byte{0x03}

	ParamsKey = []byte("p_hub")

	// ChainConfigKey stores Hub's chain config params
	// set at genesis
	ChainConfigKey = []byte("chain_config")
)

// HashJWSToken returns the SHA256 hash of a JWS token string.
func HashJWSToken(jwsToken string) string {
	hash := sha256.Sum256([]byte(jwsToken))
	return hex.EncodeToString(hash[:])
}

// JWSTokenKey returns the store key for a JWS token by its hash using length-prefixed encoding.
func JWSTokenKey(tokenHash string) []byte {
	return append(JWSTokenKeyPrefix, address.MustLengthPrefix([]byte(tokenHash))...)
}

// JWSTokenByDIDKey returns the store key for a JWS token indexed by DID and hash using length-prefixed encoding.
func JWSTokenByDIDKey(did, tokenHash string) []byte {
	key := append(JWSTokenByDIDKeyPrefix, address.MustLengthPrefix([]byte(did))...)
	return append(key, address.MustLengthPrefix([]byte(tokenHash))...)
}

// JWSTokenByAccountKey returns the store key for a JWS token indexed by account and hash using length-prefixed encoding.
func JWSTokenByAccountKey(account, tokenHash string) []byte {
	key := append(JWSTokenByAccountKeyPrefix, address.MustLengthPrefix([]byte(account))...)
	return append(key, address.MustLengthPrefix([]byte(tokenHash))...)
}

// JWSTokenDIDPrefix returns the prefix for all tokens belonging to a DID.
func JWSTokenDIDPrefix(did string) []byte {
	return append(JWSTokenByDIDKeyPrefix, address.MustLengthPrefix([]byte(did))...)
}

// JWSTokenAccountPrefix returns the prefix for all tokens belonging to an account.
func JWSTokenAccountPrefix(account string) []byte {
	return append(JWSTokenByAccountKeyPrefix, address.MustLengthPrefix([]byte(account))...)
}

// ParseJWSTokenKey parses a JWS token key and returns the token hash.
func ParseJWSTokenKey(key []byte) (tokenHash string, err error) {
	if len(key) == 0 {
		return "", fmt.Errorf("empty key")
	}

	tokenHashLen, tokenHashLenEndIndex := sdk.ParseLengthPrefixedBytes(key, 0, 1)
	if len(tokenHashLen) == 0 {
		return "", fmt.Errorf("invalid key: missing token hash length")
	}

	tokenHashBz, _ := sdk.ParseLengthPrefixedBytes(key, tokenHashLenEndIndex+1, int(tokenHashLen[0]))
	return string(tokenHashBz), nil
}

// ParseJWSTokenByDIDKey parses a JWS token by DID key and returns the DID and token hash.
func ParseJWSTokenByDIDKey(key []byte) (did, tokenHash string, err error) {
	if len(key) == 0 {
		return "", "", fmt.Errorf("empty key")
	}

	didLen, didLenEndIndex := sdk.ParseLengthPrefixedBytes(key, 0, 1)
	if len(didLen) == 0 {
		return "", "", fmt.Errorf("invalid key: missing DID length")
	}

	didBz, tokenHashStartIndex := sdk.ParseLengthPrefixedBytes(key, didLenEndIndex+1, int(didLen[0]))

	tokenHashLen, tokenHashLenEndIndex := sdk.ParseLengthPrefixedBytes(key, tokenHashStartIndex+1, 1)
	if len(tokenHashLen) == 0 {
		return "", "", fmt.Errorf("invalid key: missing token hash length")
	}

	tokenHashBz, _ := sdk.ParseLengthPrefixedBytes(key, tokenHashLenEndIndex+1, int(tokenHashLen[0]))

	return string(didBz), string(tokenHashBz), nil
}

// ParseJWSTokenByAccountKey parses a JWS token by account key and returns the account and token hash.
func ParseJWSTokenByAccountKey(key []byte) (account, tokenHash string, err error) {
	if len(key) == 0 {
		return "", "", fmt.Errorf("empty key")
	}

	accountLen, accountLenEndIndex := sdk.ParseLengthPrefixedBytes(key, 0, 1)
	if len(accountLen) == 0 {
		return "", "", fmt.Errorf("invalid key: missing account length")
	}

	accountBz, tokenHashStartIndex := sdk.ParseLengthPrefixedBytes(key, accountLenEndIndex+1, int(accountLen[0]))

	tokenHashLen, tokenHashLenEndIndex := sdk.ParseLengthPrefixedBytes(key, tokenHashStartIndex+1, 1)
	if len(tokenHashLen) == 0 {
		return "", "", fmt.Errorf("invalid key: missing token hash length")
	}

	tokenHashBz, _ := sdk.ParseLengthPrefixedBytes(key, tokenHashLenEndIndex+1, int(tokenHashLen[0]))

	return string(accountBz), string(tokenHashBz), nil
}
