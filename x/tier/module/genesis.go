package tier

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sourcenetwork/sourcehub/x/tier/keeper"
	"github.com/sourcenetwork/sourcehub/x/tier/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx context.Context, k keeper.Keeper, genState types.GenesisState) {
	// this line is used by starport scaffolding # genesis/module/init
	if err := k.SetParams(ctx, genState.Params); err != nil {
		panic(err)
	}

	for _, lockup := range genState.Lockups {
		delAddr := sdk.MustAccAddressFromBech32(lockup.DelegatorAddress)
		valAddr := types.MustValAddressFromBech32(lockup.ValidatorAddress)
		if k.HasLockup(ctx, delAddr, valAddr) {
			k.AddLockup(ctx, delAddr, valAddr, lockup.Amount)
		} else {
			k.SaveLockup(ctx, lockup.UnlockTime != nil, delAddr, valAddr, lockup.Amount, lockup.CreationHeight, lockup.UnbondTime, lockup.UnlockTime)
		}
	}
}

// ExportGenesis returns the module's exported genesis.
func ExportGenesis(ctx context.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)

	// this line is used by starport scaffolding # genesis/module/export
	genesis.Lockups = k.GetAllLockups(ctx)

	return genesis
}
