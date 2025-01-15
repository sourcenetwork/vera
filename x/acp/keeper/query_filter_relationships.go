package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

func (q Querier) FilterRelationships(goCtx context.Context, req *types.QueryFilterRelationshipsRequest) (*types.QueryFilterRelationshipsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	engine, err := q.GetACPEngine(ctx)
	if err != nil {
		return nil, err
	}

	records, err := engine.FilterRelationships(goCtx, &coretypes.FilterRelationshipsRequest{
		PolicyId: req.PolicyId,
		Selector: req.Selector,
	})
	if err != nil {
		return nil, err
	}

	return &types.QueryFilterRelationshipsResponse{
		Records: records.Records,
	}, nil
}
