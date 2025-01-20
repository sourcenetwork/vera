package types

import (
	"bytes"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// ModuleName defines the module name
	ModuleName = "tier"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey is the message route for the module
	RouterKey = ModuleName

	// QuerierRoute is the querier route for the module
	QuerierRoute = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_tier"

	// LockupKeyPrefix is the prefix to retrieve all Lockup
	LockupKeyPrefix = "Lockup/"

	// UnlockingLockupKeyPrefix is the prefix to retrieve all UnlockingLockup
	UnlockingLockupKeyPrefix = "UnlockingLockup/"
)

var (
	ParamsKey = []byte("p_tier")
)

func KeyPrefix(unlocking bool) []byte {
	if unlocking {
		return []byte(UnlockingLockupKeyPrefix)
	}
	return []byte(LockupKeyPrefix)
}

// LockupKey returns the store key to retrieve a Lockup from the index fields.
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

// UnlockingLockupKey returns the store key to retrieve an unlocking Lockup from the index fields.
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

// LockupKeyToAddresses retreives delAddr and valAddr from provided Lockup key.
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

// UnlockingLockupKeyToAddressesAtHeight retreives delAddr, valAddr, and creationHeight from provided unlocking Lockup key.
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
