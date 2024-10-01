package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sourcenetwork/sourcehub/x/acp/registration"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) ListObjectEvents(goCtx context.Context, req *types.QueryListObjectEventsRequest) (*types.QueryListObjectEventsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	var repo registration.RegistrationEventRepository
	events, err := repo.GetObjectEvents(ctx, req.PolicyId, req.Object)
	if err != nil {
		return nil, err
	}

	return &types.QueryListObjectEventsResponse{
		Events: events,
	}, nil
}
