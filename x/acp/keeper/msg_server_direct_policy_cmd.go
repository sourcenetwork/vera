package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sourcenetwork/vera/x/acp/keeper/policy_cmd"
	"github.com/sourcenetwork/vera/x/acp/types"
)

func (k *Keeper) DirectPolicyCmd(goCtx context.Context, msg *types.MsgDirectPolicyCmd) (*types.MsgDirectPolicyCmdResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	actorID, err := k.GetActorDID(ctx, msg.Creator)
	if err != nil {
		return nil, fmt.Errorf("DirectPolicyCmd: %w", err)
	}

	cmdCtx, err := policy_cmd.NewPolicyCmdCtx(ctx, msg.PolicyId, actorID, msg.Creator, k.GetParams(ctx))
	if err != nil {
		return nil, err
	}

	handler := k.getPolicyCmdHandler(ctx)
	result, err := handler.Dispatch(&cmdCtx, msg.Cmd)
	if err != nil {
		return nil, err
	}

	return &types.MsgDirectPolicyCmdResponse{
		Result: result,
	}, nil
}
