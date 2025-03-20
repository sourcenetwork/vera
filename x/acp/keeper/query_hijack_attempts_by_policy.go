package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sourcenetwork/raccoondb/v2/iterator"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) HijackAttemptsByPolicy(goCtx context.Context, req *types.QueryHijackAttemptsByPolicyRequest) (*types.QueryHijackAttemptsByPolicyResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	repo := k.GetAmendmentEventRepository(ctx)
	iter, err := repo.ListHijackEventsByPolicy(ctx, req.PolicyId)
	if err != nil {
		return nil, err
	}

	evs, err := iterator.Consume(ctx, iter)
	if err != nil {
		return nil, err
	}

	return &types.QueryHijackAttemptsByPolicyResponse{
		Events: evs,
	}, nil
}
