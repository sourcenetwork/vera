package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/sourcenetwork/vera/x/acp/types"
)

func (k *Keeper) VerifyAccessRequest(
	goCtx context.Context,
	req *types.QueryVerifyAccessRequestRequest,
) (*types.QueryVerifyAccessRequestResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	engine := k.getACPEngine(ctx)

	actorId := req.AccessRequest.Actor.Id
	if did, err := k.GetActorDID(ctx, actorId); err == nil {
		req.AccessRequest.Actor.Id = did
	}

	result, err := engine.VerifyAccessRequest(ctx, &coretypes.VerifyAccessRequestRequest{
		PolicyId:      req.PolicyId,
		AccessRequest: req.AccessRequest,
	})
	if err != nil {
		return nil, err
	}

	return &types.QueryVerifyAccessRequestResponse{
		Valid: result.Valid,
	}, nil
}
