package bulletin

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/sourcenetwork/sourcehub/x/bulletin/keeper"
	"github.com/sourcenetwork/sourcehub/x/bulletin/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx sdk.Context, k *keeper.Keeper, genState types.GenesisState) {
	if err := k.SetParams(ctx, genState.Params); err != nil {
		panic(err)
	}

	policyId := genState.PolicyId
	if policyId != "" {
		// Reset the policy if initializing from the exported state
		policyId, err := k.EnsurePolicy(ctx)
		if err != nil {
			panic(err)
		}
		k.SetPolicyId(ctx, policyId)

		for _, namespace := range genState.Namespaces {
			err := keeper.RegisterNamespace(ctx, k, policyId, namespace.Id, namespace.OwnerDid, namespace.Creator)
			if err != nil {
				panic(err)
			}
			k.SetNamespace(ctx, namespace)
		}

		for _, collaborator := range genState.Collaborators {
			namespace := k.GetNamespace(ctx, collaborator.Namespace)
			err := keeper.AddCollaborator(ctx, k, policyId, collaborator.Namespace, collaborator.Did, namespace.OwnerDid, collaborator.Address)
			if err != nil {
				panic(err)
			}
			k.SetCollaborator(ctx, collaborator)
		}

		for _, post := range genState.Posts {
			k.SetPost(ctx, post)
		}
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
