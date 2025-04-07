package keeper

import (
	"testing"

	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/sourcenetwork/sourcehub/x/acp/capability"
	"github.com/sourcenetwork/sourcehub/x/acp/did"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
	"github.com/stretchr/testify/require"
)

func Test_CreateModulePolicy_ModuleCanCreatePolicy(t *testing.T) {
	ctx, k, _, _ := setupKeeperWithCapability(t)

	pol := "name: test"
	record, capability, err := k.CreateModulePolicy(ctx, pol, coretypes.PolicyMarshalingType_SHORT_YAML, "external")

	require.NoError(t, err)
	require.Equal(t, pol, record.RawPolicy)
	require.Equal(t, record.Policy.Id, capability.GetPolicyId())
	require.Equal(t, "/acp/module_policies/"+record.Policy.Id, capability.GetCapabilityName())
	require.NotNil(t, capability.GetCosmosCapability())
}

func Test_EditModulePolicy_CannotEditWithoutClaimingCapability(t *testing.T) {
	ctx, k, _, _ := setupKeeperWithCapability(t)

	// Given Policy created by module without a claimed capability
	pol := "name: test"
	_, cap, err := k.CreateModulePolicy(ctx, pol, coretypes.PolicyMarshalingType_SHORT_YAML, "external")
	require.NoError(t, err)

	// When the module attempts to edit the policy
	pol = "name: new-name"
	result, _, err := k.EditModulePolicy(ctx, cap, pol, coretypes.PolicyMarshalingType_SHORT_YAML)

	// Then cmd is reject due to invalid capability
	require.Nil(t, result)
	require.ErrorIs(t, err, capability.ErrInvalidCapability)
}

func Test_EditModulePolicy_ModuleCanEditPolicyTiedToClaimedCapability(t *testing.T) {
	ctx, k, _, capK := setupKeeperWithCapability(t)

	moduleName := "test_module"
	scopedKeeper := capK.ScopeToModule(moduleName)

	// Given policy create by test_module
	pol := "name: test"
	_, cap, err := k.CreateModulePolicy(ctx, pol, coretypes.PolicyMarshalingType_SHORT_YAML, moduleName)
	require.NoError(t, err)
	// And capability claimed by the module
	manager := capability.NewPolicyCapabilityManager(&scopedKeeper)
	err = manager.Claim(ctx, cap)
	require.NoError(t, err)

	// When the module edits the policy
	pol = "name: new-name"
	record, _, err := k.EditModulePolicy(ctx, cap, pol, coretypes.PolicyMarshalingType_SHORT_YAML)

	// Then policy record was edited with no error
	require.NoError(t, err)
	require.Equal(t, pol, record.RawPolicy)
}

func Test_ModulePolicyCmdForActorDID_ModuleCanAddRelationshipsToTheirPolicy(t *testing.T) {
	ctx, k, _, capK := setupKeeperWithCapability(t)

	// Given policy created
	pol := `
name: test
resources:
  file:
`
	moduleName := "mod1"
	_, cap, err := k.CreateModulePolicy(ctx, pol, coretypes.PolicyMarshalingType_SHORT_YAML, moduleName)
	require.NoError(t, err)
	// And claimed by mod1
	scopedKeeper := capK.ScopeToModule(moduleName)
	manager := capability.NewPolicyCapabilityManager(&scopedKeeper)
	err = manager.Claim(ctx, cap)
	require.NoError(t, err)

	// When module issues a policy cmd
	cmd := types.NewRegisterObjectCmd(coretypes.NewObject("file", "foo"))
	signer := "source1twjwexwrsvflt9nv9xwk27e0f2defa9fdjaeus"
	result, err := k.ModulePolicyCmdForActorDID(ctx, cap, cmd, "did:example:bob", signer)

	// Then it is accepted
	require.NoError(t, err)
	resultCmd := result.GetRegisterObjectResult()
	require.Equal(t, "did:example:bob", resultCmd.Record.Metadata.OwnerDid)
	require.Equal(t, signer, resultCmd.Record.Metadata.TxSigner)
}

func Test_ModulePolicyCmdForActorAccount_ModuleCanAddRelationshipsToTheirPolicy(t *testing.T) {
	ctx, k, accKeep, capK := setupKeeperWithCapability(t)

	// Given policy created
	pol := `
name: test
resources:
  file:
`
	moduleName := "mod1"
	_, cap, err := k.CreateModulePolicy(ctx, pol, coretypes.PolicyMarshalingType_SHORT_YAML, moduleName)
	require.NoError(t, err)
	// And claimed by mod1
	scopedKeeper := capK.ScopeToModule(moduleName)
	manager := capability.NewPolicyCapabilityManager(&scopedKeeper)
	err = manager.Claim(ctx, cap)
	require.NoError(t, err)

	// When module issues a policy cmd to an actor acc
	cmd := types.NewRegisterObjectCmd(coretypes.NewObject("file", "foo"))
	signer := "source1twjwexwrsvflt9nv9xwk27e0f2defa9fdjaeus"
	accAddr := accKeep.FirstAcc().GetAddress().String()
	result, err := k.ModulePolicyCmdForActorAccount(ctx, cap, cmd, accAddr, signer)

	// Then it is accepted
	require.NoError(t, err)
	resultCmd := result.GetRegisterObjectResult()
	accDID, err := did.IssueDID(accKeep.FirstAcc())
	require.NoError(t, err)
	require.Equal(t, accDID, resultCmd.Record.Metadata.OwnerDid)
	require.Equal(t, signer, resultCmd.Record.Metadata.TxSigner)
}

func Test_ModulePolicyCmdForActorAccount_ModuleCannotUsePolicyWithoutClaimingCapability(t *testing.T) {
	ctx, k, accKeep, _ := setupKeeperWithCapability(t)

	// Given Policy created by module without a claimed capability
	pol := "name: test"
	_, cap, err := k.CreateModulePolicy(ctx, pol, coretypes.PolicyMarshalingType_SHORT_YAML, "external")
	require.NoError(t, err)

	// When module issues a policy cmd to an actor acc
	cmd := types.NewRegisterObjectCmd(coretypes.NewObject("file", "foo"))
	signer := "source1twjwexwrsvflt9nv9xwk27e0f2defa9fdjaeus"
	accAddr := accKeep.FirstAcc().GetAddress().String()
	result, err := k.ModulePolicyCmdForActorAccount(ctx, cap, cmd, accAddr, signer)

	// Then cmd is reject due to invalid capability
	require.Nil(t, result)
	require.ErrorIs(t, err, capability.ErrInvalidCapability)
}
