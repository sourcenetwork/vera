package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sourcenetwork/acp_core/pkg/errors"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k *Keeper) RegistrationsCommitment(
	goCtx context.Context,
	req *types.QueryRegistrationsCommitmentRequest,
) (*types.QueryRegistrationsCommitmentResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	repo := k.getRegistrationsCommitmentRepository(ctx)

	opt, err := repo.GetById(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if opt.Empty() {
		return nil, errors.Wrap("commitment not found", errors.ErrorType_NOT_FOUND,
			errors.Pair("commitment", req.Id))
	}

	return &types.QueryRegistrationsCommitmentResponse{
		RegistrationsCommitment: opt.GetValue(),
	}, nil
}
