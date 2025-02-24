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
	engine := q.GetACPEngine(ctx)

	resp, err := engine.ListPolicies(ctx, &coretypes.ListPoliciesRequest{})
	if err != nil {
		return nil, err
	}

	// Use MapNullableSlice instead of MapSlice to filter out 'nil' policies.
	return &types.QueryPolicyIdsResponse{
		Ids: utils.MapNullableSlice(resp.Records, func(p *coretypes.PolicyRecord) string { return p.Policy.Id }),
	}, nil
}
