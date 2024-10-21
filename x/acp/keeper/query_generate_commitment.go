package keeper

import (
	"context"
	"encoding/hex"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/jsonpb"
	"github.com/sourcenetwork/sourcehub/utils"
	"github.com/sourcenetwork/sourcehub/x/acp/registration"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GenerateCommitment(goCtx context.Context, req *types.QueryGenerateCommitmentRequest) (*types.QueryGenerateCommitmentResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	engine, err := k.GetACPEngine(ctx)
	if err != nil {
		return nil, err
	}

	eventRepo := k.GetObjectEventRepository(ctx)
	commitRepo := k.GetRegistrationsCommitmentRepository(ctx)
	eventService := registration.NewEventService(eventRepo)
	service := registration.NewRegistrationService(engine, eventService, commitRepo)

	commitment, err := service.GenerateCommitment(ctx, req.PolicyId, req.Actor, req.Objects)
	if err != nil {
		return nil, err
	}

	proofs := make([]*types.RegistrationProof, 0, len(req.Objects))
	for i := range req.Objects {
		proof, err := registration.ProofForObject(req.PolicyId, req.Actor, i, req.Objects)
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
		Commitment:    commitment,
		HexCommitment: hex.EncodeToString(commitment),
		Proofs:        proofs,
		ProofsJson:    proofsJson,
	}, nil
}
