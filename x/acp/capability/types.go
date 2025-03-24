package capability

import (
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
)

// PolicyCapability models a capability which grants
// unrevokable access to any command within a policy
type PolicyCapability struct {
	policyId   string
	capability capabilitytypes.Capability
}

// GetCapabilityName returns the name of the given PolicyCapability
func (c *PolicyCapability) GetCapabilityName() string {
	return "/acp/module_policies/" + c.policyId
}

func (c *PolicyCapability) GetPolicyId() string {
	return c.policyId
}

func (c *PolicyCapability) GetOwnerModule() string {
	panic("todo")
	return ""
}

func (c *PolicyCapability) GetCosmosCapability() *capabilitytypes.Capability {
	return &c.capability
}
