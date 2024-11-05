package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sourcenetwork/acp_core/pkg/errors"
	"github.com/sourcenetwork/sourcehub/x/acp/registration"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

func (k *Keeper) EndBlocker(goCtx context.Context) ([]*types.RegistrationsCommitment, error) {
	return nil, nil
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
		return nil, errors.Wrap("end blocker failed", err)
	}

	return commitments, nil
}
