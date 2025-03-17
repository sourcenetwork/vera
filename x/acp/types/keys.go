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
	RegistrationsCommitmentKeyPrefix = "commitment/"

	// AmendmentEventKeyPrefix defines a key prefix for Amendment Events
	AmendmentEventKeyPrefix = "amendment_event/"
)

var (
	ParamsKey = []byte("p_acp")
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}
