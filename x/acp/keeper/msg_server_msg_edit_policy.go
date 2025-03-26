package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

func (k msgServer) MsgEditPolicy(goCtx context.Context, msg *types.MsgEditPolicy) (*types.MsgEditPolicyResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO: Handling the message
	_ = ctx

	return &types.MsgEditPolicyResponse{}, nil
}
