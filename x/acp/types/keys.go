package types

const (
	// ModuleName defines the module name
	ModuleName = "acp"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// AccessDecisionRepositoryKeyPrefix defines the namespace for Access Decisions
	AccessDecisionRepositoryKeyPrefix = "access_decision/"

	// RegistrationsCommitmentKeyPrefix defines a key prefix for RegistrationsCommitments
	RegistrationsCommitmentKeyPrefix = "commitment/"

	// AmendmentEventKeyPrefix defines a key prefix for Amendment Events
	AmendmentEventKeyPrefix = "amendment_event/"

	// SignedPolicyCmdSeenKeyPrefix defines a key prefix for seen signed policy command payloads
	SignedPolicyCmdSeenKeyPrefix = "spc_seen/"
)

var (
	ParamsKey = []byte("p_acp")
)
