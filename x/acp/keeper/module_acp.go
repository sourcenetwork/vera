package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sourcenetwork/acp_core/pkg/errors"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/sourcenetwork/sourcehub/x/acp/capability"
	"github.com/sourcenetwork/sourcehub/x/acp/did"
	"github.com/sourcenetwork/sourcehub/x/acp/keeper/policy_cmd"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
	"github.com/sourcenetwork/sourcehub/x/acp/utils"
)

// CreateModulePolicy creates a new Policy within the ACP module, bound to some calling module.
// Returns the created Policy and a Capability, which authorizes the presenter to operate over this policy.
//
// Callers must Claim the capability, as it is a unique instance which cannot be recreated after dropped.
// Claiming can be done using the callers capability keeper directly or the policy capability manager provided in the capability package.
func (k *Keeper) CreateModulePolicy(goCtx context.Context, policy string, marshalType coretypes.PolicyMarshalingType, module string) (*types.PolicyRecord, *capability.PolicyCapability, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	engine := k.getACPEngine(ctx)

	moduleDID := did.IssuedModuleDID(module)
	metadata, err := types.BuildACPSuppliedMetadata(ctx, moduleDID, module)
	if err != nil {
		return nil, nil, err
	}

	ctx, err = utils.InjectPrincipal(ctx, moduleDID)
	if err != nil {
		return nil, nil, err
	}

	coreResult, err := engine.CreatePolicy(goCtx, &coretypes.CreatePolicyRequest{
		Policy:      policy,
		MarshalType: marshalType,
		Metadata:    metadata,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("CreateModulePolicy: %w", err)
	}

	rec, err := types.MapPolicy(coreResult.Record)
	if err != nil {
		return nil, nil, fmt.Errorf("CreateModulePolicy: %w", err)
	}

	capMananager := k.getPolicyCapabilityManager(ctx)
	cap, err := capMananager.Issue(ctx, rec.Policy.Id)
	if err != nil {
		return nil, nil, err
	}

	return rec, cap, nil
}

// EditModulePolicy updates the policy definition attached to the given PolicyCapability
func (k *Keeper) EditModulePolicy(goCtx context.Context, cap *capability.PolicyCapability, policy string, marshalType coretypes.PolicyMarshalingType) (*types.PolicyRecord, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	engine := k.getACPEngine(ctx)

	capManager := k.getPolicyCapabilityManager(ctx)

	err := capManager.Validate(ctx, cap)
	if err != nil {
		return nil, err
	}

	module, err := capManager.GetOwnerModule(ctx, cap)
	if err != nil {
		return nil, err
	}

	moduleDID := did.IssuedModuleDID(module)

	ctx, err = utils.InjectPrincipal(ctx, moduleDID)
	if err != nil {
		return nil, err
	}

	coreResult, err := engine.EditPolicy(goCtx, &coretypes.EditPolicyRequest{
		PolicyId:    cap.GetPolicyId(),
		Policy:      policy,
		MarshalType: marshalType,
	})
	if err != nil {
		return nil, fmt.Errorf("EditModulePolicy: %w", err)
	}

	rec, err := types.MapPolicy(coreResult.Record)
	if err != nil {
		return nil, fmt.Errorf("EditModulePolicy: %w", err)
	}

	return rec, nil
}

// ModulePolicyCmdForActorAccount issues a policy command for the policy bound to the provided capability.
// The command skips authentication and is assumed to be issued by actorAcc, which must be a valid sourcehub account address.
func (k *Keeper) ModulePolicyCmdForActorAccount(goCtx context.Context, cap *capability.PolicyCapability, cmd *types.PolicyCmd, actorAcc string, txSigner string) (*types.PolicyCmdResult, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	actorDID, err := k.issueDIDFromAccountAddr(ctx, actorAcc)
	if err != nil {
		return nil, errors.Wrap("DirectPolicyCmd: could not issue did to creator",
			errors.ErrorType_BAD_INPUT, errors.Pair("address", actorAcc))
	}

	return k.ModulePolicyCmdForActorDID(goCtx, cap, cmd, actorDID, txSigner)
}

// ModulePolicyCmdForActorDID issues a policy command for the policy bound to the provided capability.
// The command skips authentication and is assumed to be issued by the actor given by actorID, which must be a valid DID.
func (k *Keeper) ModulePolicyCmdForActorDID(goCtx context.Context, capability *capability.PolicyCapability, cmd *types.PolicyCmd, actorDID string, txSigner string) (*types.PolicyCmdResult, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := k.getPolicyCapabilityManager(ctx).Validate(ctx, capability)
	if err != nil {
		return nil, err
	}

	polId := capability.GetPolicyId()
	cmdCtx, err := policy_cmd.NewPolicyCmdCtx(ctx, polId, actorDID, txSigner, k.GetParams(ctx))
	if err != nil {
		return nil, err
	}

	handler := k.getPolicyCmdHandler(ctx)
	result, err := handler.Dispatch(&cmdCtx, cmd)
	if err != nil {
		return nil, err
	}

	return result, nil
}
