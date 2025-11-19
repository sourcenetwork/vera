package feegrant

import (
	time "time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
)

const (
	// ModuleName is the module name constant used in many places
	ModuleName = "feegrant"

	// StoreKey is the store key string for supply
	StoreKey = ModuleName

	// RouterKey is the message route for supply
	RouterKey = ModuleName

	// QuerierRoute is the querier route for supply
	QuerierRoute = ModuleName
)

var (
	// FeeAllowanceKeyPrefix is the set of the kvstore for fee allowance data
	// - 0x00<allowance_key_bytes>: allowance
	FeeAllowanceKeyPrefix = []byte{0x00}

	// FeeAllowanceQueueKeyPrefix is the set of the kvstore for fee allowance keys data
	// - 0x01<allowance_prefix_queue_key_bytes>: <empty value>
	FeeAllowanceQueueKeyPrefix = []byte{0x01}

	// DIDFeeAllowanceKeyPrefix is the set of the kvstore for DID fee allowance data
	// - 0x02<did_allowance_key_bytes>: did_allowance
	DIDFeeAllowanceKeyPrefix = []byte{0x02}

	// DIDFeeAllowanceQueueKeyPrefix is the set of the kvstore for DID fee allowance expiration queue
	// - 0x03<did_allowance_prefix_queue_key_bytes>: <empty value>
	DIDFeeAllowanceQueueKeyPrefix = []byte{0x03}
)

// FeeAllowanceKey is the canonical key to store a grant from granter to grantee
// We store by grantee first to allow searching by everyone who granted to you
//
// Key format:
// - <0x00><len(grantee_address_bytes)><grantee_address_bytes><len(granter_address_bytes)><granter_address_bytes>
func FeeAllowanceKey(granter, grantee sdk.AccAddress) []byte {
	return append(FeeAllowancePrefixByGrantee(grantee), address.MustLengthPrefix(granter.Bytes())...)
}

// FeeAllowanceByDIDKey is the canonical key to store a grant from granter to DID
// We store by DID first to allow searching by DID
//
// Key format:
// - <0x02><len(did_bytes)><did_bytes><len(granter_address_bytes)><granter_address_bytes>
func FeeAllowanceByDIDKey(granter sdk.AccAddress, did string) []byte {
	return append(FeeAllowancePrefixByDID(did), address.MustLengthPrefix(granter.Bytes())...)
}

// FeeAllowancePrefixByGrantee returns a prefix to scan for all grants to this given address.
//
// Key format:
// - <0x00><len(grantee_address_bytes)><grantee_address_bytes>
func FeeAllowancePrefixByGrantee(grantee sdk.AccAddress) []byte {
	return append(FeeAllowanceKeyPrefix, address.MustLengthPrefix(grantee.Bytes())...)
}

// FeeAllowancePrefixByDID returns a prefix to scan for all grants to this given DID.
//
// Key format:
// - <0x02><len(did_bytes)><did_bytes>
func FeeAllowancePrefixByDID(did string) []byte {
	didBytes := []byte(did)
	return append(DIDFeeAllowanceKeyPrefix, address.MustLengthPrefix(didBytes)...)
}

// FeeAllowancePrefixQueue is the canonical key to store grant key.
//
// Key format:
// - <0x01><exp_bytes><len(grantee_address_bytes)><grantee_address_bytes><len(granter_address_bytes)><granter_address_bytes>
func FeeAllowancePrefixQueue(exp *time.Time, key []byte) []byte {
	allowanceByExpTimeKey := AllowanceByExpTimeKey(exp)
	return append(allowanceByExpTimeKey, key...)
}

// AllowanceByExpTimeKey returns a key with `FeeAllowanceQueueKeyPrefix`, expiry
//
// Key format:
// - <0x01><exp_bytes>
func AllowanceByExpTimeKey(exp *time.Time) []byte {
	// no need of appending len(exp_bytes) here, `FormatTimeBytes` gives const length everytime.
	return append(FeeAllowanceQueueKeyPrefix, sdk.FormatTimeBytes(*exp)...)
}

// DIDFeeAllowancePrefixQueue is the canonical key to store DID grant key.
//
// Key format:
// - <0x03><exp_bytes><len(did_bytes)><did_bytes><len(granter_address_bytes)><granter_address_bytes>
func DIDFeeAllowancePrefixQueue(exp *time.Time, key []byte) []byte {
	allowanceByExpTimeKey := DIDAllowanceByExpTimeKey(exp)
	return append(allowanceByExpTimeKey, key...)
}

// DIDAllowanceByExpTimeKey returns a key with `DIDFeeAllowanceQueueKeyPrefix`, expiry
//
// Key format:
// - <0x03><exp_bytes>
func DIDAllowanceByExpTimeKey(exp *time.Time) []byte {
	// no need of appending len(exp_bytes) here, `FormatTimeBytes` gives const length everytime.
	return append(DIDFeeAllowanceQueueKeyPrefix, sdk.FormatTimeBytes(*exp)...)
}

// ParseAddressesFromFeeAllowanceKey extracts and returns the granter, grantee from the given key.
func ParseAddressesFromFeeAllowanceKey(key []byte) (granter, grantee []byte) {
	// key is of format:
	// 0x00<granteeAddressLen (1 Byte)><granteeAddress_Bytes><granterAddressLen (1 Byte)><granterAddress_Bytes>
	granterAddrLen, granterAddrLenEndIndex := sdk.ParseLengthPrefixedBytes(key, 1, 1) // ignore key[0] since it is a prefix key
	grantee, granterAddrEndIndex := sdk.ParseLengthPrefixedBytes(key, granterAddrLenEndIndex+1, int(granterAddrLen[0]))

	granteeAddrLen, granteeAddrLenEndIndex := sdk.ParseLengthPrefixedBytes(key, granterAddrEndIndex+1, 1)
	granter, _ = sdk.ParseLengthPrefixedBytes(key, granteeAddrLenEndIndex+1, int(granteeAddrLen[0]))

	return granter, grantee
}

// ParseAddressesFromFeeAllowanceQueueKey extracts and returns the granter, grantee from the given key.
func ParseAddressesFromFeeAllowanceQueueKey(key []byte) (granter, grantee []byte) {
	lenTime := len(sdk.FormatTimeBytes(time.Now()))

	// key is of format:
	// <0x01><expiration_bytes(fixed length)><granteeAddressLen (1 Byte)><granteeAddress_Bytes><granterAddressLen (1 Byte)><granterAddress_Bytes>
	granterAddrLen, granterAddrLenEndIndex := sdk.ParseLengthPrefixedBytes(key, 1+lenTime, 1) // ignore key[0] since it is a prefix key
	grantee, granterAddrEndIndex := sdk.ParseLengthPrefixedBytes(key, granterAddrLenEndIndex+1, int(granterAddrLen[0]))

	granteeAddrLen, granteeAddrLenEndIndex := sdk.ParseLengthPrefixedBytes(key, granterAddrEndIndex+1, 1)
	granter, _ = sdk.ParseLengthPrefixedBytes(key, granteeAddrLenEndIndex+1, int(granteeAddrLen[0]))

	return granter, grantee
}

// ParseGranterDIDFromFeeAllowanceKey extracts and returns the granter address and DID from the given DID-based key.
func ParseGranterDIDFromFeeAllowanceKey(key []byte) (granter []byte, did string) {
	// key is of format:
	// 0x02<didLen (1 Byte)><did_Bytes><granterAddressLen (1 Byte)><granterAddress_Bytes>
	didLen, didLenEndIndex := sdk.ParseLengthPrefixedBytes(key, 1, 1) // ignore key[0] since it is a prefix key
	didBytes, granterAddrEndIndex := sdk.ParseLengthPrefixedBytes(key, didLenEndIndex+1, int(didLen[0]))

	granterAddrLen, granterAddrLenEndIndex := sdk.ParseLengthPrefixedBytes(key, granterAddrEndIndex+1, 1)
	granter, _ = sdk.ParseLengthPrefixedBytes(key, granterAddrLenEndIndex+1, int(granterAddrLen[0]))

	return granter, string(didBytes)
}

// ParseGranterDIDFromDIDAllowanceQueueKey extracts and returns the granter address and DID from the given DID expiration queue key.
func ParseGranterDIDFromDIDAllowanceQueueKey(key []byte) (granter []byte, did string) {
	lenTime := len(sdk.FormatTimeBytes(time.Now()))

	// key is of format:
	// <0x03><expiration_bytes(fixed length)><didLen (1 Byte)><did_Bytes><granterAddressLen (1 Byte)><granterAddress_Bytes>
	didLen, didLenEndIndex := sdk.ParseLengthPrefixedBytes(key, 1+lenTime, 1) // ignore key[0] since it is a prefix key
	didBytes, granterAddrEndIndex := sdk.ParseLengthPrefixedBytes(key, didLenEndIndex+1, int(didLen[0]))

	granterAddrLen, granterAddrLenEndIndex := sdk.ParseLengthPrefixedBytes(key, granterAddrEndIndex+1, 1)
	granter, _ = sdk.ParseLengthPrefixedBytes(key, granterAddrLenEndIndex+1, int(granterAddrLen[0]))

	return granter, string(didBytes)
}
