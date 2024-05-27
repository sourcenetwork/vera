package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/sourcenetwork/sourcehub/x/acp/access_decision"
	"github.com/sourcenetwork/sourcehub/x/acp/did"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

func (k Keeper) VerifyAccessRequest(goCtx context.Context, req *types.QueryVerifyAccessRequestRequest) (*types.QueryVerifyAccessRequestResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	engine, err := k.GetZanziEngine(ctx)
	if err != nil {
		return nil, err
	}

	rec, err := engine.GetPolicy(goCtx, req.PolicyId)
	if err != nil {
		return nil, err
	}

	actorId := req.AccessRequest.Actor.Id
	addr, err := sdk.AccAddressFromBech32(actorId)
	if err == nil {
		// this means the actor ID is a cosmos account
		// so convert it to a did
		acc := k.accountKeeper.GetAccount(ctx, addr)
		did, err := did.IssueDID(acc)
		if err != nil {
			return nil, fmt.Errorf("verify access request: could not produce did for actor %v: %v: %w", actorId, err, types.ErrAcpInput)
		}
		req.AccessRequest.Actor.Id = did
	}

	cmd := access_decision.VerifyAccessRequestQuery{
		Policy:        rec.Policy,
		AccessRequest: req.AccessRequest,
	}
	valid, err := cmd.Execute(ctx, engine)
	if err != nil {
		return nil, err
	}

	return &types.QueryVerifyAccessRequestResponse{
		Valid: valid,
	}, nil
}
