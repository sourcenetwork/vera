package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"

	"github.com/sourcenetwork/sourcehub/utils"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

func (k Keeper) PolicyIds(goCtx context.Context, req *types.QueryPolicyIdsRequest) (*types.QueryPolicyIdsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	engine := k.GetACPEngine(ctx)

	resp, err := engine.ListPolicies(ctx, &coretypes.ListPoliciesRequest{})
	if err != nil {
		return nil, err
	}

	// Use MapNullableSlice instead of MapSlice to filter out 'nil' policies.
	return &types.QueryPolicyIdsResponse{
		Ids: utils.MapNullableSlice(resp.Policies, func(p *coretypes.Policy) string { return p.Id }),
	}, nil
}
