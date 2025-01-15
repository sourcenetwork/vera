package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"

	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

func (q Querier) ValidatePolicy(goCtx context.Context, req *types.QueryValidatePolicyRequest) (*types.QueryValidatePolicyResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	engine, err := q.GetACPEngine(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := engine.ValidatePolicy(ctx, &coretypes.ValidatePolicyRequest{
		Policy:      req.Policy,
		MarshalType: req.MarshalType,
	})
	if err != nil {
		return nil, err
	}

	return &types.QueryValidatePolicyResponse{
		Valid:    resp.Valid,
		ErrorMsg: resp.ErrorMsg,
	}, nil
}
