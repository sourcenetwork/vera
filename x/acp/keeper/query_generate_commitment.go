package keeper

import (
	"context"
	"encoding/hex"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/jsonpb"
	"github.com/sourcenetwork/sourcehub/utils"
	"github.com/sourcenetwork/sourcehub/x/acp/commitment"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (q Querier) GenerateCommitment(goCtx context.Context, req *types.QueryGenerateCommitmentRequest) (*types.QueryGenerateCommitmentResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	engine := q.GetACPEngine(ctx)

	commitRepo := q.GetRegistrationsCommitmentRepository(ctx)
	commitmentService := commitment.NewCommitmentService(engine, commitRepo)

	comm, err := commitmentService.BuildCommitment(ctx, req.PolicyId, req.Actor, req.Objects)
	if err != nil {
		return nil, err
	}

	proofs := make([]*types.RegistrationProof, 0, len(req.Objects))
	for i := range req.Objects {
		proof, err := commitment.ProofForObject(req.PolicyId, req.Actor, i, req.Objects)
		if err != nil {
			return nil, fmt.Errorf("generating proof for obj %v: %v", i, err)
		}
		proofs = append(proofs, proof)
	}
	proofsJson, err := utils.MapFailableSlice(proofs, func(p *types.RegistrationProof) (string, error) {
		marshaler := jsonpb.Marshaler{}
		return marshaler.MarshalToString(p)
	})
	if err != nil {
		return nil, err
	}

	return &types.QueryGenerateCommitmentResponse{
		Commitment:    comm,
		HexCommitment: hex.EncodeToString(comm),
		Proofs:        proofs,
		ProofsJson:    proofsJson,
	}, nil
}
