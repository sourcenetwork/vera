package tier

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sourcenetwork/sourcehub/x/tier/keeper"
	"github.com/sourcenetwork/sourcehub/x/tier/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx context.Context, k *keeper.Keeper, genState types.GenesisState) {
	if err := k.SetParams(ctx, genState.Params); err != nil {
		panic(err)
	}

	for _, lockup := range genState.Lockups {
		delAddr := sdk.MustAccAddressFromBech32(lockup.DelegatorAddress)
		valAddr := types.MustValAddressFromBech32(lockup.ValidatorAddress)
		k.AddLockup(ctx, delAddr, valAddr, lockup.Amount)
	}

	for _, unlockingLockup := range genState.UnlockingLockups {
		delAddr := sdk.MustAccAddressFromBech32(unlockingLockup.DelegatorAddress)
		valAddr := types.MustValAddressFromBech32(unlockingLockup.ValidatorAddress)
		if !k.HasUnlockingLockup(ctx, delAddr, valAddr, unlockingLockup.CreationHeight) {
			k.SetUnlockingLockup(
				ctx,
				delAddr,
				valAddr,
				unlockingLockup.CreationHeight,
				unlockingLockup.Amount,
				unlockingLockup.CompletionTime,
				unlockingLockup.UnlockTime,
			)
		}
	}

	for _, insuranceLockup := range genState.InsuranceLockups {
		delAddr := sdk.MustAccAddressFromBech32(insuranceLockup.DelegatorAddress)
		valAddr := types.MustValAddressFromBech32(insuranceLockup.ValidatorAddress)
		k.AddInsuranceLockup(ctx, delAddr, valAddr, insuranceLockup.Amount)
	}
}

// ExportGenesis returns the module's exported genesis.
func ExportGenesis(ctx context.Context, k *keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)

	genesis.Lockups = k.GetAllLockups(ctx)
	genesis.UnlockingLockups = k.GetAllUnlockingLockups(ctx)
	genesis.InsuranceLockups = k.GetAllInsuranceLockups(ctx)

	return genesis
}
