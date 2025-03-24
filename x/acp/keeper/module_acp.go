package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sourcenetwork/acp_core/pkg/errors"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	hubtypes "github.com/sourcenetwork/sourcehub/types"
	"github.com/sourcenetwork/sourcehub/x/acp/capability"
	"github.com/sourcenetwork/sourcehub/x/acp/did"
	"github.com/sourcenetwork/sourcehub/x/acp/keeper/policy_cmd"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
	"github.com/sourcenetwork/sourcehub/x/acp/utils"
)

func (k *Keeper) CreateModulePolicy(goCtx context.Context, policy string, marshalType coretypes.PolicyMarshalingType, module string, capability any) (*types.PolicyRecord, *capability.PolicyCapability, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	engine := k.getACPEngine(ctx)

	modDID := k.deriveModuleDID(ctx, module)
	metadata, err := types.BuildACPSuppliedMetadata(ctx, modDID, modDID)
	if err != nil {
		return nil, nil, err
	}

	ctx, err = utils.InjectPrincipal(ctx, modDID)
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
	cap, err := capMananager.Register(ctx, rec.Policy.Id)
	if err != nil {
		return nil, nil, err
	}

	return rec, cap, nil
}

func (k *Keeper) ModulePolicyCmdForActorAccount(goCtx context.Context, cap *capability.PolicyCapability, cmd *types.PolicyCmd, actorAcc string) (*types.PolicyCmdResult, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	addr, err := hubtypes.AccAddressFromBech32(actorAcc)
	if err != nil {
		return nil, fmt.Errorf("DirectPolicyCmd: %v: %w", err, types.NewErrInvalidAccAddrErr(err, actorAcc))
	}

	acc := k.accountKeeper.GetAccount(ctx, addr)
	if acc == nil {
		return nil, fmt.Errorf("DirectPolicyCmd: %w", types.NewAccNotFoundErr(actorAcc))
	}

	actorID, err := did.IssueDID(acc)
	if err != nil {
		return nil, errors.Wrap("DirectPolicyCmd: could not issue did to creator",
			errors.ErrorType_BAD_INPUT, errors.Pair("address", actorAcc))
	}

	return k.ModulePolicyCmdForActorDID(goCtx, cap, cmd, actorID)
}

func (k *Keeper) ModulePolicyCmdForActorDID(goCtx context.Context, capability *capability.PolicyCapability, cmd *types.PolicyCmd, actorID string) (*types.PolicyCmdResult, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := k.getPolicyCapabilityManager(ctx).Validate(ctx, capability)
	if err != nil {
		return nil, err
	}

	mod := capability.GetOwnerModule()
	polId := capability.GetPolicyId()

	modDID := k.deriveModuleDID(ctx, mod)
	cmdCtx, err := policy_cmd.NewPolicyCmdCtx(ctx, polId, actorID, modDID, k.GetParams(ctx))
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

func (k *Keeper) deriveModuleDID(ctx context.Context, module string) string {
	panic("todo")
	return ""
}
