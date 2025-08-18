package types

import (
	"bytes"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// ModuleName defines the module name
	ModuleName = "tier"

	// DeveloperPoolName defines the developer pool module account
	DeveloperPoolName = "developer_pool"

	// InsurancePoolName defines the developer insurance pool module account
	InsurancePoolName = "insurance_pool"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey is the message route for the module
	RouterKey = ModuleName

	// QuerierRoute is the querier route for the module
	QuerierRoute = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_tier"

	// LockupKeyPrefix is the prefix to retrieve all Lockups
	LockupKeyPrefix = "lockup/"

	// UnlockingLockupKeyPrefix is the prefix to retrieve all UnlockingLockups
	UnlockingLockupKeyPrefix = "unlockingLockup/"

	// InsuranceLockupKeyPrefix is the prefix to retrieve all insurance Lockups
	InsuranceLockupKeyPrefix = "insuranceLockup/"

	// UserSubscriptionKeyPrefix is the prefix to store user subscriptions
	UserSubscriptionKeyPrefix = "userSub/"

	// TotalDevGrantedKeyPrefix is the prefix to store developer total granted amounts
	TotalDevGrantedKeyPrefix = "totalDevGranted/"

	// DeveloperKeyPrefix is the prefix to store developer configurations
	DeveloperKeyPrefix = "developer/"
)

var (
	ParamsKey       = []byte("p_tier")
	TotalLockupsKey = []byte("total_lockups")
	TotalCreditsKey = []byte("total_credits")
)

func KeyPrefix(unlocking bool) []byte {
	if unlocking {
		return []byte(UnlockingLockupKeyPrefix)
	}
	return []byte(LockupKeyPrefix)
}

// LockupKey builds and returns a key to store/retrieve the Lockup.
func LockupKey(delAddr sdk.AccAddress, valAddr sdk.ValAddress) []byte {
	// Calculate the size of the buffer in advance
	size := len(delAddr.Bytes()) + 1 + len(valAddr.Bytes()) + 1
	buf := make([]byte, 0, size)

	// Append bytes to the buffer
	buf = append(buf, delAddr.Bytes()...)
	buf = append(buf, '/')
	buf = append(buf, valAddr.Bytes()...)
	buf = append(buf, '/')

	return buf
}

// UnlockingLockupKey builds and returns the key to store/retrieve the UnlockingLockup.
func UnlockingLockupKey(delAddr sdk.AccAddress, valAddr sdk.ValAddress, creationHeight int64) []byte {
	// Calculate the size of the buffer in advance, allocating 20 more bytes for creationHeight.
	creationHeightLength := 20
	size := len(delAddr.Bytes()) + 1 + len(valAddr.Bytes()) + 1 + creationHeightLength + 1
	buf := make([]byte, 0, size)

	// Append bytes to the buffer
	buf = append(buf, delAddr.Bytes()...)
	buf = append(buf, '/')
	buf = append(buf, valAddr.Bytes()...)
	buf = append(buf, '/')
	buf = strconv.AppendInt(buf, creationHeight, 10)
	buf = append(buf, '/')

	return buf
}

// LockupKeyToAddresses retrieves delAddr and valAddr from provided Lockup key.
func LockupKeyToAddresses(key []byte) (sdk.AccAddress, sdk.ValAddress) {
	// Find the positions of the delimiters
	parts := bytes.Split(key, []byte{'/'})
	if len(parts) != 3 {
		panic("invalid key format: expected format delAddr/valAddr/")
	}

	// Reconstruct the addresses
	delAddr := sdk.AccAddress(parts[0])
	valAddr := sdk.ValAddress(parts[1])

	return delAddr, valAddr
}

// UnlockingLockupKeyToAddressesAtHeight retrieves delAddr, valAddr, and creationHeight from provided unlocking Lockup key.
func UnlockingLockupKeyToAddressesAtHeight(key []byte) (sdk.AccAddress, sdk.ValAddress, int64) {
	// Find the positions of the delimiters
	parts := bytes.Split(key, []byte{'/'})
	if len(parts) != 4 {
		panic("invalid key format: expected format delAddr/valAddr/creationHeight/")
	}

	// Reconstruct the addresses and creation height
	delAddr := sdk.AccAddress(parts[0])
	valAddr := sdk.ValAddress(parts[1])
	creationHeight, err := strconv.ParseInt(string(parts[2]), 10, 64)
	if err != nil {
		panic("unexpected creation height")
	}

	return delAddr, valAddr, creationHeight
}

// UserSubscriptionKey builds and returns a key to store user subscription data.
// Key format: UserSubscriptionKeyPrefix | developer_addr | user_addr
func UserSubscriptionKey(developerAddr sdk.AccAddress, userAddr sdk.AccAddress) []byte {
	// Calculate the size of the buffer in advance
	size := len(developerAddr.Bytes()) + 1 + len(userAddr.Bytes()) + 1
	buf := make([]byte, 0, size)

	// Append bytes to the buffer
	buf = append(buf, developerAddr.Bytes()...)
	buf = append(buf, '/')
	buf = append(buf, userAddr.Bytes()...)
	buf = append(buf, '/')

	return buf
}

// UserSubscriptionKeyToAddresses retrieves developerAddr and userAddr from provided UserSubscriptionKey.
func UserSubscriptionKeyToAddresses(key []byte) (sdk.AccAddress, sdk.AccAddress) {
	// Find the positions of the delimiters
	parts := bytes.Split(key, []byte{'/'})
	if len(parts) != 3 {
		panic("invalid key format: expected format developerAddr/userAddr/")
	}

	// Reconstruct the addresses
	developerAddr := sdk.AccAddress(parts[0])
	userAddr := sdk.AccAddress(parts[1])

	return developerAddr, userAddr
}

// DeveloperKey builds and returns a key to store developer configuration.
// Key format: DeveloperKeyPrefix | developer_addr
func DeveloperKey(developerAddr sdk.AccAddress) []byte {
	// Calculate the size of the buffer in advance
	size := len(DeveloperKeyPrefix) + len(developerAddr.Bytes()) + 1
	buf := make([]byte, 0, size)

	// Append prefix and developer address
	buf = append(buf, []byte(DeveloperKeyPrefix)...)
	buf = append(buf, developerAddr.Bytes()...)
	buf = append(buf, '/')

	return buf
}

// TotalDevGrantedKey builds and returns a key to store developer total granted amount.
// Key format: TotalDevGrantedKeyPrefix | developer_addr
func TotalDevGrantedKey(developerAddr sdk.AccAddress) []byte {
	// Calculate the size of the buffer in advance
	size := len(TotalDevGrantedKeyPrefix) + len(developerAddr.Bytes()) + 1
	buf := make([]byte, 0, size)

	// Append prefix and developer address
	buf = append(buf, []byte(TotalDevGrantedKeyPrefix)...)
	buf = append(buf, developerAddr.Bytes()...)
	buf = append(buf, '/')

	return buf
}
