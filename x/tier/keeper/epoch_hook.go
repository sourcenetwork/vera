package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"

	epochstypes "github.com/sourcenetwork/sourcehub/x/epochs/types"
	"github.com/sourcenetwork/sourcehub/x/tier/types"
)

func (k *Keeper) EpochHooks() epochstypes.EpochHooks {
	return EpochHooks{k}
}

type EpochHooks struct {
	keeper *Keeper
}

var _ epochstypes.EpochHooks = EpochHooks{}

// GetModuleName implements types.EpochHooks.
func (EpochHooks) GetModuleName() string {
	return types.ModuleName
}

// BeforeEpochStart is the epoch start hook.
func (h EpochHooks) BeforeEpochStart(ctx context.Context, epochIdentifier string, epochNumber int64) error {
	if epochIdentifier != types.EpochIdentifier {
		return nil
	}

	h.keeper.Logger().Info("resetting all credits", "epochID", epochIdentifier, "epochNumber", epochNumber)

	err := h.keeper.burnAllCredits(ctx, epochNumber)
	if err != nil {
		return errorsmod.Wrapf(err, "burn all credits")
	}

	err = h.keeper.resetAllCredits(ctx, epochNumber)
	if err != nil {
		return errorsmod.Wrapf(err, "reset all credits")
	}

	return nil
}

func (h EpochHooks) AfterEpochEnd(ctx context.Context, epochIdentifier string, epochNumber int64) error {
	if epochIdentifier != types.EpochIdentifier {
		return nil
	}

	h.keeper.Logger().Info("completing unlocking stakes", "epochID", epochIdentifier, "epochNumber", epochNumber)

	err := h.keeper.CompleteUnlocking(ctx)
	if err != nil {
		return errorsmod.Wrapf(err, "complete unlocking stakes")
	}

	return nil
}
