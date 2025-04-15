package bulletin

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/sourcenetwork/sourcehub/x/acp/capability"
	"github.com/sourcenetwork/sourcehub/x/bulletin/keeper"
	"github.com/sourcenetwork/sourcehub/x/bulletin/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx sdk.Context, k *keeper.Keeper, genState types.GenesisState) {
	if err := k.SetParams(ctx, genState.Params); err != nil {
		panic(err)
	}

	_, polCap, err := k.GetAcpKeeper().CreateModulePolicy(
		ctx,
		types.BasePolicy(),
		coretypes.PolicyMarshalingType_SHORT_YAML,
		types.ModuleName,
	)
	if err != nil {
		panic(err)
	}

	policyId := polCap.GetPolicyId()
	k.SetPolicyId(ctx, policyId)

	manager := capability.NewPolicyCapabilityManager(k.GetScopedKeeper())
	err = manager.Claim(ctx, polCap)
	if err != nil {
		panic(err)
	}

	for _, namespace := range genState.Namespaces {
		err = keeper.RegisterNamespace(ctx, k, policyId, namespace.Id, namespace.OwnerDid, namespace.Creator)
		if err != nil {
			panic(err)
		}
		k.SetNamespace(ctx, namespace)
	}

	for _, collaborator := range genState.Collaborators {
		namespace := k.GetNamespace(ctx, collaborator.Namespace)
		err = keeper.AddCollaborator(ctx, k, policyId, collaborator.Namespace, collaborator.Did, namespace.OwnerDid, collaborator.Address)
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
func ExportGenesis(ctx sdk.Context, k *keeper.Keeper) *types.GenesisState {
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
