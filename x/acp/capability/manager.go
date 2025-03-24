package capability

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
)

// Overall this would work and is a nice abstraction wrt to the capability name,
// however it's not great because this wrapper essentially requries that acp
// code has access to an external module's capability keeper -
// consequently any capability that acp knows the name of -
// which callers may not like.

// NewPolicyCapabilityManager returns a PolicyCapabilityManager scoped to the calling module.
//
// Requires a scoped capability keeper, which authenticates the caller and limits
// the capabilities they have access to.
func NewPolicyCapabilityManager(keeper capabilitykeeper.ScopedKeeper) *PolicyCapabilityManager {
	return &PolicyCapabilityManager{
		scopedKeeper: keeper,
	}
}

// PolicyCapabilityManager models a manager for PolicyCapabilities.
//
// The manager provides methods to claim and fetch capabilities returned by the acp keeper.
type PolicyCapabilityManager struct {
	scopedKeeper capabilitykeeper.ScopedKeeper
}

// Fetch looks up a PolicyCapability based on a policyId.
// This capability will be an exact replica of the capability returned by the acp keeper,
// upon policy registration.
//
// The capability will only be returned if it was previously registered with `Claim`.
func (m *PolicyCapabilityManager) Fetch(ctx sdk.Context, policyId string) (*PolicyCapability, error) {
	panic("todo")
	return nil, nil
}

// Claim register the current module as one of the owners of capability.
// Callers which have received a capability are responsible for Claiming it.
//
// The registration is bound to the module's scoped capability keeper,
// which binds the capability to the caller module.
//
// This step is necessary in order to retrieve the capability in the future.
func (m *PolicyCapabilityManager) Claim(ctx sdk.Context, capability PolicyCapability) error {
	panic("todo")
	return nil
}

func (m *PolicyCapabilityManager) Register(ctx sdk.Context, policyId string) (*PolicyCapability, error) {
	polCap := &PolicyCapability{
		policyId: policyId,
	}
	cap, err := m.scopedKeeper.NewCapability(ctx, polCap.GetCapabilityName())
	if err != nil {
		return nil, err
	}
	polCap.capability = *cap
	return polCap, nil
}

// Validate verifies whether the given capability is valid
func (m *PolicyCapabilityManager) Validate(ctx sdk.Context, capability *PolicyCapability) error {
	ok := m.scopedKeeper.AuthenticateCapability(ctx, &capability.capability, capability.GetCapabilityName())
	if !ok {
		return fmt.Errorf("invalid capability")
	}
	return nil
}
