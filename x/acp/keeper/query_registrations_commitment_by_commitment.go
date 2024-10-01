package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sourcenetwork/sourcehub/x/acp/registration"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) RegistrationsCommitmentByCommitment(goCtx context.Context, req *types.QueryRegistrationsCommitmentByCommitmentRequest) (*types.QueryRegistrationsCommitmentByCommitmentResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var repo registration.CommitmentRepository = nil

	ctx := sdk.UnwrapSDKContext(goCtx)

	commitments, err := repo.FilterByCommitment(ctx, req.Commitment)
	if err != nil {
		return nil, err
	}

	return &types.QueryRegistrationsCommitmentByCommitmentResponse{
		RegistrationsCommitments: commitments,
	}, nil
}
