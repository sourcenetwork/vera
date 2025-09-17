package types

const (
	// ModuleName defines the module name
	ModuleName = "ica"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_ica"

	// ICAConnectionKeyPrefix defines a key prefix for ICA connections
	ICAConnectionKeyPrefix = "ica_connection/"
)

var (
	ParamsKey = []byte("p_ica")
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}
