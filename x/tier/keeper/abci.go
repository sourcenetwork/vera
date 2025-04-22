package keeper

import (
	"context"
)

// BeginBlocker handles slashing events and processes tier module staking rewards.
func (k *Keeper) BeginBlocker(ctx context.Context) error {
	k.handleSlashingEvents(ctx)

	err := k.processRewards(ctx)
	if err != nil {
		k.Logger().Error("Failed to process rewards", "error", err)
	}

	return nil
}
