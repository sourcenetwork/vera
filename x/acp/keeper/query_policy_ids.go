package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/sourcenetwork/acp_core/pkg/utils"

	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

func (k Keeper) PolicyIds(goCtx context.Context, req *types.QueryPolicyIdsRequest) (*types.QueryPolicyIdsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	engine := k.GetACPEngine(ctx)

	resp, err := engine.ListPolicies(ctx, &coretypes.ListPoliciesRequest{})
	if err != nil {
		return nil, err
	}

	return &types.QueryPolicyIdsResponse{
		Ids: utils.MapSlice(resp.Records, func(r *coretypes.PolicyRecord) string { return r.Policy.Id }),
	}, nil
}
