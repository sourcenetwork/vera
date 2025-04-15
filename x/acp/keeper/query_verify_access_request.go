package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sourcenetwork/acp_core/pkg/errors"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/sourcenetwork/sourcehub/x/acp/did"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

func (k *Keeper) VerifyAccessRequest(
	goCtx context.Context,
	req *types.QueryVerifyAccessRequestRequest,
) (*types.QueryVerifyAccessRequestResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	engine := k.getACPEngine(ctx)

	actorId := req.AccessRequest.Actor.Id
	addr, err := sdk.AccAddressFromBech32(actorId)
	if err == nil {
		// this means the actor ID is a cosmos account, so convert it to a did
		acc := k.accountKeeper.GetAccount(ctx, addr)
		if acc == nil {
			return nil, errors.Wrap(
				"verify access request: could not produce did for actor",
				errors.ErrorType_BAD_INPUT, errors.Pair("actorId", actorId),
			)
		}
		did, err := did.IssueDID(acc)
		if err != nil {
			return nil, errors.Wrap(
				"verify access request: could not produce did for actor",
				errors.ErrorType_BAD_INPUT, errors.Pair("actorId", actorId),
			)
		}
		req.AccessRequest.Actor.Id = did
	}

	result, err := engine.VerifyAccessRequest(ctx, &coretypes.VerifyAccessRequestRequest{
		PolicyId:      req.PolicyId,
		AccessRequest: req.AccessRequest,
	})
	if err != nil {
		return nil, err
	}

	return &types.QueryVerifyAccessRequestResponse{
		Valid: result.Valid,
	}, nil
}
