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
func (k *Keeper) EndBlocker(goCtx context.Context) ([]*types.RegistrationsCommitment, error) {
	defer telemetry.ModuleMeasureSince(types.ModuleName, time.Now(), telemetry.MetricKeyEndBlocker)

	ctx := sdk.UnwrapSDKContext(goCtx)
	engine := k.getACPEngine(ctx)
	repo := k.getRegistrationsCommitmentRepository(ctx)
	service := commitment.NewCommitmentService(engine, repo)

	commitments, err := service.FlagExpiredCommitments(ctx)
	if err != nil {
		return nil, err
	}

	return commitments, nil
}
