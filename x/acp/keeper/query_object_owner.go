package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

func (q Querier) ObjectOwner(goCtx context.Context, req *types.QueryObjectOwnerRequest) (*types.QueryObjectOwnerResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	engine, err := q.GetACPEngine(ctx)
	if err != nil {
		return nil, err
	}

	result, err := engine.GetObjectRegistration(ctx, &coretypes.GetObjectRegistrationRequest{
		PolicyId: req.PolicyId,
		Object:   req.Object,
	})
	if err != nil {
		return nil, err
	}

	return &types.QueryObjectOwnerResponse{
		IsRegistered: result.IsRegistered,
		OwnerId:      result.OwnerId,
	}, nil
}
