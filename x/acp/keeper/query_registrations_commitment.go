package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sourcenetwork/sourcehub/x/acp/registration"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) RegistrationsCommitment(goCtx context.Context, req *types.QueryRegistrationsCommitmentRequest) (*types.QueryRegistrationsCommitmentResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	var repo registration.CommitmentRepository = nil

	commitment, err := repo.GetById(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	return &types.QueryRegistrationsCommitmentResponse{
		RegistrationsCommitment: commitment,
	}, nil
}
