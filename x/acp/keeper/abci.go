package keeper

import (
	"context"
	"time"

	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sourcenetwork/sourcehub/x/acp/commitment"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

// EndBlocker is a function which should be called after processing a proposal and before finalizing a block,
// as specified by the ABCI spec.
//
// Currently EndBlocker iterates over valid (non-expired) RegistrationCommitments and checks whether
// they are still valid, otherwise flags them as expired.
func (k *Keeper) EndBlocker(goCtx context.Context) []*types.RegistrationsCommitment {
	defer telemetry.ModuleMeasureSince(types.ModuleName, time.Now(), telemetry.MetricKeyEndBlocker)

	ctx := sdk.UnwrapSDKContext(goCtx)
	engine := k.GetACPEngine(ctx)
	repo := k.GetRegistrationsCommitmentRepository(ctx)
	service := commitment.NewCommitmentService(engine, repo)

	// If an error occurs, return nil commitments and log the error
	commitments, err := service.FlagExpiredCommitments(ctx)
	if err != nil {
		k.Logger().Error("EndBlocker failed", "error", err)
	}

	return commitments
}
