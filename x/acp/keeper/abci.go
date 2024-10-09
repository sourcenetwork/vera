package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sourcenetwork/sourcehub/x/acp/registration"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

// EndBlocker called at every block, update validator set
func (k *Keeper) EndBlocker(goCtx context.Context) ([]*types.RegistrationsCommitment, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	objRepo := k.GetObjectEventRepository(ctx)
	evService := registration.NewEventService(objRepo)
	commitRepo := k.GetRegistrationsCommitmentRepository(ctx)
	engine, err := k.GetACPEngine(ctx)
	if err != nil {
		return nil, err
	}
	service := registration.NewRegistrationService(engine, evService, commitRepo)

	commitments, err := service.FlagExpiredCommitments(ctx)
	if err != nil {
		return nil, err
	}

	return commitments, nil
}
