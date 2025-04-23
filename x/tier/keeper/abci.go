package keeper

import (
	"context"
	"time"

	"github.com/cosmos/cosmos-sdk/telemetry"
	"github.com/sourcenetwork/sourcehub/app/metrics"
	"github.com/sourcenetwork/sourcehub/x/tier/types"
)

// BeginBlocker handles slashing events and processes tier module staking rewards.
func (k *Keeper) BeginBlocker(ctx context.Context) error {
	defer telemetry.ModuleMeasureSince(types.ModuleName, time.Now(), telemetry.MetricKeyBeginBlocker)

	k.handleSlashingEvents(ctx)

	err := k.processRewards(ctx)
	if err != nil {
		metrics.ModuleIncrInternalErrorCounter(types.ModuleName, telemetry.MetricKeyBeginBlocker, err)
		k.Logger().Error("Failed to process rewards", "error", err)
	}

	return nil
}
