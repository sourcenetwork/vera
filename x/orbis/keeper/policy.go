package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/sourcenetwork/sourcehub/x/acp/capability"
	"github.com/sourcenetwork/sourcehub/x/orbis/types"
)

// EnsurePolicy creates the module-level orbis policy and claims the capability if it does not exist.
// Returns the existing policy id if it was already created.
func (k *Keeper) EnsurePolicy(ctx sdk.Context) (string, error) {
	if id := k.GetPolicyId(ctx); id != "" {
		return id, nil
	}

	_, polCap, err := k.GetAcpKeeper().CreateModulePolicy(
		ctx,
		types.BasePolicy(),
		coretypes.PolicyMarshalingType_YAML,
		types.ModuleName,
	)
	if err != nil {
		return "", err
	}

	manager := capability.NewPolicyCapabilityManager(k.GetScopedKeeper())
	if err := manager.Claim(ctx, polCap); err != nil {
		return "", err
	}

	policyId := polCap.GetPolicyId()
	k.SetPolicyId(ctx, policyId)
	return policyId, nil
}
