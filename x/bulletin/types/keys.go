package types

const (
	// ModuleName defines the module name
	ModuleName = "bulletin"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_bulletin"

	PostKey = "Post/Value/"

	NamespaceResource = "namespace"
)

var (
	ParamsKey = []byte("p_bulletin")
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}
