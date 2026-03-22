package capability

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPolicyCapabilityGetCapabilityName(t *testing.T) {
	cap := &PolicyCapability{policyId: "test-policy-123"}
	require.Equal(t, "/acp/module_policies/test-policy-123", cap.GetCapabilityName())
}

func TestPolicyCapabilityGetPolicyId(t *testing.T) {
	cap := &PolicyCapability{policyId: "test-policy-123"}
	require.Equal(t, "test-policy-123", cap.GetPolicyId())
}

func TestPolicyCapabilityGetCosmosCapability(t *testing.T) {
	cap := &PolicyCapability{policyId: "p1", capability: nil}
	require.Nil(t, cap.GetCosmosCapability())
}
