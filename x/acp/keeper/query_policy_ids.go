package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/sourcenetwork/sourcehub/utils"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

func (q Querier) PolicyIds(goCtx context.Context, req *types.QueryPolicyIdsRequest) (*types.QueryPolicyIdsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	engine, err := q.GetACPEngine(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := engine.ListPolicies(ctx, &coretypes.ListPoliciesRequest{})
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, status.Error(codes.NotFound, "no policies found")
	}

	// Use MapNullableSlice instead of MapSlice to filter out 'nil' policies.
	return &types.QueryPolicyIdsResponse{
		Ids: utils.MapNullableSlice(resp.Policies, func(p *coretypes.Policy) string { return p.Id }),
	}, nil
}
