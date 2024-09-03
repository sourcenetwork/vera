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

func (k Keeper) Policy(goCtx context.Context, req *types.QueryPolicyRequest) (*types.QueryPolicyResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	engine, err := k.GetACPEngine(ctx)
	if err != nil {
		return nil, err
	}

	rec, err := engine.GetPolicy(goCtx, &coretypes.GetPolicyRequest{
		Id: req.Id,
	})
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, errors.NewPolicyNotFound(req.Id)
	}

	return &types.QueryPolicyResponse{
		Policy:      rec.Policy,
		PolicyRaw:   rec.PolicyRaw,
		MarshalType: rec.MarshalType,
	}, nil
}
