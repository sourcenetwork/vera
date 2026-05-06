package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestModuleName(t *testing.T) {
	require.Equal(t, "acp", ModuleName)
}

func TestStoreKey(t *testing.T) {
	require.Equal(t, ModuleName, StoreKey)
}

func TestMemStoreKey(t *testing.T) {
	require.Equal(t, "mem_acp", MemStoreKey)
}

func TestKeyPrefixes(t *testing.T) {
	require.Equal(t, "access_decision/", AccessDecisionRepositoryKeyPrefix)
	require.Equal(t, "commitment/", RegistrationsCommitmentKeyPrefix)
	require.Equal(t, "amendment_event/", AmendmentEventKeyPrefix)
	require.Equal(t, "spc_seen/", SignedPolicyCmdSeenKeyPrefix)
}

func TestParamsKey(t *testing.T) {
	require.Equal(t, []byte("p_acp"), ParamsKey)
}
