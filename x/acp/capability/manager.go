package capability

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
	"github.com/sourcenetwork/acp_core/pkg/errors"
	"github.com/sourcenetwork/raccoondb/v2/utils"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
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
func NewPolicyCapabilityManager(keeper *capabilitykeeper.ScopedKeeper) *PolicyCapabilityManager {
	return &PolicyCapabilityManager{
		scopedKeeper: keeper,
	}
}

// PolicyCapabilityManager models a manager for PolicyCapabilities.
//
// The manager provides methods to claim and fetch capabilities returned by the acp keeper.
type PolicyCapabilityManager struct {
	scopedKeeper *capabilitykeeper.ScopedKeeper
}

// Fetch looks up a PolicyCapability based on a policyId.
// This capability will be an exact replica of the capability returned by the acp keeper,
// upon policy registration.
//
// The capability will only be returned if it was previously registered with `Claim`.
func (m *PolicyCapabilityManager) Fetch(ctx sdk.Context, policyId string) (*PolicyCapability, error) {
	cap := &PolicyCapability{
		policyId: policyId,
	}

	sdkCap, ok := m.scopedKeeper.GetCapability(ctx, cap.GetCapabilityName())
	if !ok {
		return nil, fmt.Errorf("fetching capability for policy %v", policyId)
	}
	cap.capability = sdkCap

	return cap, nil
}

// Claim register the current module as one of the owners of capability.
// Callers which have received a capability are responsible for Claiming it.
//
// The registration is bound to the module's scoped capability keeper,
// which binds the capability to the caller module.
//
// This step is necessary in order to retrieve the capability in the future.
func (m *PolicyCapabilityManager) Claim(ctx sdk.Context, capability *PolicyCapability) error {
	return m.scopedKeeper.ClaimCapability(ctx, capability.capability, capability.GetCapabilityName())
}

// Issue creates a new PolicyCapability from a policyId.
// The created capability is bound to the calling module's name.
func (m *PolicyCapabilityManager) Issue(ctx sdk.Context, policyId string) (*PolicyCapability, error) {
	polCap := &PolicyCapability{
		policyId: policyId,
	}
	cap, err := m.scopedKeeper.NewCapability(ctx, polCap.GetCapabilityName())
	if err != nil {
		return nil, err
	}
	polCap.capability = cap
	return polCap, nil
}

// Validate verifies whether the given capability is valid
func (m *PolicyCapabilityManager) Validate(ctx sdk.Context, capability *PolicyCapability) error {
	ok := m.scopedKeeper.AuthenticateCapability(ctx, capability.capability, capability.GetCapabilityName())
	if !ok {
		return errors.Wrap("authentication failed", ErrInvalidCapability)
	}

	// check if the capability is also owned by acp module.
	// this prevents a malicious module from creating a capability using a known policy id,
	// which would give the module full control over the policy.
	//
	// By verifying that a PolicyCapability is owned by the ACP module, we implicitly verify
	// that the Policy was created through the CreateModulePolicy call,
	// as opposed to the regular user flow.
	//
	// This verification depends on the invariant that the acp keeper
	// never claims any capability, it only issues them.
	ownedByAcp, err := m.isOwnedByAcpModule(ctx, capability)
	if err != nil {
		return err
	}
	if !ownedByAcp {
		return errors.Wrap("capability not issued by acp module", ErrInvalidCapability)
	}

	return nil
}

// GetOwnerModule returns the co-owner of a PolicyCapability.
// ie. the module which received the capability from the ACP module and claimed it.
func (m *PolicyCapabilityManager) GetOwnerModule(ctx sdk.Context, capability *PolicyCapability) (string, error) {
	mods, _, err := m.scopedKeeper.LookupModules(ctx, capability.GetCapabilityName())
	if err != nil {
		return "", fmt.Errorf("looking up capability owner: %v", err) //TODO
	}

	mods = utils.FilterSlice(mods, func(name string) bool {
		return name != types.ModuleName
	})

	if len(mods) == 0 {
		return "", errors.Wrap("capability not claimed by any module", ErrInvalidCapability)
	}
	return mods[0], nil
}

// isOwnedByAcpModule returns true if the acp module is an owner of the given PolicyCapability.
func (m *PolicyCapabilityManager) isOwnedByAcpModule(ctx sdk.Context, capability *PolicyCapability) (bool, error) {
	mods, _, err := m.scopedKeeper.LookupModules(ctx, capability.GetCapabilityName())
	if err != nil {
		return false, fmt.Errorf("looking up capability owner: %v", err)
	}

	mods = utils.FilterSlice(mods, func(name string) bool {
		return name != types.ModuleName
	})

	if len(mods) == 0 {
		return false, nil
	}
	return true, nil
}
