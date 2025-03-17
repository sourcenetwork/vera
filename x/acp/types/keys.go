package types

const (
	// ModuleName defines the module name
	ModuleName = "acp"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_acp"

	// AccessDecisionRepositoryKeyPrefix defines the namespace for Access Decisions
	AccessDecisionRepositoryKeyPrefix = "access_decision/"

	// RegistrationsCommitmentKeyPrefix defines a key prefix for RegistrationsCommitments
	RegistrationsCommitmentKeyPrefix = "commitments/"

	// ObjectEventsKeyPrefix defines a key prefix for ObjectEvents
	ObjectEventsKeyPrefix = "object_events/"
)

var (
	ParamsKey = []byte("p_acp")
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}
