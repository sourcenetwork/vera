package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sourcenetwork/acp_core/pkg/errors"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

func (q Querier) Policy(goCtx context.Context, req *types.QueryPolicyRequest) (*types.QueryPolicyResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	engine, err := q.GetACPEngine(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := engine.GetPolicy(ctx, &coretypes.GetPolicyRequest{
		Id: req.Id,
	})
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, errors.NewPolicyNotFound(req.Id)
	}

	return &types.QueryPolicyResponse{
		Policy: resp.Policy,
	}, nil
}
