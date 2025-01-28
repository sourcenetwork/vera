package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sourcenetwork/acp_core/pkg/errors"
	"github.com/sourcenetwork/sourcehub/x/acp/commitment"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

func (k *Keeper) EndBlocker(goCtx context.Context) ([]*types.RegistrationsCommitment, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	engine := k.GetACPEngine(ctx)
	repo := k.GetRegistrationsCommitmentRepository(ctx)
	service := commitment.NewCommitmentService(engine, repo)

	commitments, err := service.FlagExpiredCommitments(ctx)
	if err != nil {
		return nil, errors.Wrap("end blocker failed", err)
	}

	return commitments, nil
}
