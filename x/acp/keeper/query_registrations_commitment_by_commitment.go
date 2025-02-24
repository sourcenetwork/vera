package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sourcenetwork/raccoondb/v2/iterator"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (q Querier) RegistrationsCommitmentByCommitment(goCtx context.Context, req *types.QueryRegistrationsCommitmentByCommitmentRequest) (*types.QueryRegistrationsCommitmentByCommitmentResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	repo := q.GetRegistrationsCommitmentRepository(ctx)

	iter, err := repo.FilterByCommitment(ctx, req.Commitment)
	if err != nil {
		return nil, err
	}

	commitments, err := iterator.Consume(ctx, iter)
	if err != nil {
		return nil, err
	}

	return &types.QueryRegistrationsCommitmentByCommitmentResponse{
		RegistrationsCommitments: commitments,
	}, nil
}
