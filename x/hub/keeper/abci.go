package keeper

import (
	"context"
	"time"

	"github.com/cosmos/cosmos-sdk/telemetry"
	"github.com/sourcenetwork/sourcehub/app/metrics"
	"github.com/sourcenetwork/sourcehub/x/hub/types"
)

// EndBlocker checks for expired JWS tokens and marks them as invalid.
func (k *Keeper) EndBlocker(ctx context.Context) error {
	defer telemetry.ModuleMeasureSince(types.ModuleName, time.Now(), telemetry.MetricKeyEndBlocker)

	if err := k.CheckAndUpdateExpiredTokens(ctx); err != nil {
		metrics.ModuleIncrInternalErrorCounter(types.ModuleName, telemetry.MetricKeyEndBlocker, err)
		k.Logger().Error("Failed to check and update expired JWS tokens", "error", err)
	}

	return nil
}
