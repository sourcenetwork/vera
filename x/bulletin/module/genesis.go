package bulletin

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/sourcenetwork/sourcehub/x/acp/did"
	"github.com/sourcenetwork/sourcehub/x/acp/keeper/policy_cmd"
	acptypes "github.com/sourcenetwork/sourcehub/x/acp/types"
	acputils "github.com/sourcenetwork/sourcehub/x/acp/utils"
	"github.com/sourcenetwork/sourcehub/x/bulletin/keeper"
	"github.com/sourcenetwork/sourcehub/x/bulletin/types"
)

// basePolicy defines base policy for the bulletin module namespaces.
func basePolicy() string {
	policyStr := `
	name: Bulletin Policy
	description: Base policy that defines permissions for bulletin namespaces
	resources:
		namespace:
			relations:
				owner:
					types:
						- actor
				collaborator:
					types: 
						- actor
			permissions:
				create_post:
					expr: owner + collaborator
	`
	return policyStr
}

// createBasePolicy creates base bulletin module policy.
func createBasePolicy(ctx sdk.Context, k keeper.Keeper, modAcc sdk.ModuleAccountI, actorDID string) (string, error) {
	metadata, err := acptypes.BuildACPSuppliedMetadata(ctx, actorDID, modAcc.GetAddress().String())
	if err != nil {
		return "", err
	}

	ctx, err = acputils.InjectPrincipal(ctx, actorDID)
	if err != nil {
		return "", err
	}

	engine := k.GetAcpKeeper().GetACPEngine(ctx)

	coreResult, err := engine.CreatePolicy(ctx, &coretypes.CreatePolicyRequest{
		Policy:      basePolicy(),
		MarshalType: coretypes.PolicyMarshalingType_SHORT_YAML,
		Metadata:    metadata,
	})
	if err != nil {
		return "", err
	}

	rec, err := acptypes.MapPolicy(coreResult.Record)
	if err != nil {
		return "", err
	}

	return rec.Policy.Id, nil
}

// registerBulletinNamespace registers bulletin namespace with bulletin module DID as the owner.
func registerBulletinNamespace(ctx sdk.Context, k keeper.Keeper, modAcc sdk.ModuleAccountI, actorDID string, policyId string) error {
	cmdCtx, err := policy_cmd.NewPolicyCmdCtx(ctx, policyId, actorDID, modAcc.GetAddress().String(), k.GetAcpKeeper().GetParams(ctx))
	if err != nil {
		return err
	}

	policyCmd := acptypes.NewRegisterObjectCmd(coretypes.NewObject(types.NamespaceResource, types.ModuleName))
	policyCmdHandler := k.GetAcpKeeper().GetPolicyCmdHandler(ctx)

	_, err = policyCmdHandler.Dispatch(&cmdCtx, policyCmd)
	if err != nil {
		return err
	}

	return nil
}

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	if err := k.SetParams(ctx, genState.Params); err != nil {
		panic(err)
	}

	modAcc := k.GetAccountKeeper().GetModuleAccount(ctx, types.ModuleName)
	actorDID, err := did.IssueModuleDID(modAcc)
	if err != nil {
		panic(err)
	}

	policyId, err := createBasePolicy(ctx, k, modAcc, actorDID)
	if err != nil {
		panic(err)
	}
	k.SetPolicyId(ctx, policyId)

	err = registerBulletinNamespace(ctx, k, modAcc, actorDID, policyId)
	if err != nil {
		panic(err)
	}

	for _, namespace := range genState.Namespaces {
		err = keeper.RegisterNamespace(ctx, &k, namespace.Id, namespace.OwnerDid, namespace.Creator)
		if err != nil {
			panic(err)
		}
		k.SetNamespace(ctx, namespace)
	}

	for _, collaborator := range genState.Collaborators {
		namespace := k.GetNamespace(ctx, collaborator.Namespace)
		err = keeper.AddCollaborator(ctx, &k, collaborator.Namespace, collaborator.Did, namespace.OwnerDid, collaborator.Address)
		if err != nil {
			panic(err)
		}
		k.SetCollaborator(ctx, collaborator)
	}

	for _, post := range genState.Posts {
		k.SetPost(ctx, post)
	}
}

// ExportGenesis returns the module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)

	if policyId := k.GetPolicyId(ctx); policyId != "" {
		genesis.PolicyId = policyId
	}

	genesis.Namespaces = k.GetAllNamespaces(ctx)
	genesis.Collaborators = k.GetAllCollaborators(ctx)
	genesis.Posts = k.GetAllPosts(ctx)

	return genesis
}
