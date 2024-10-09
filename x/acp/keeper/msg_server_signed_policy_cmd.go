package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/sourcenetwork/sourcehub/x/acp/did"
	"github.com/sourcenetwork/sourcehub/x/acp/signed_policy_cmd"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

func (k msgServer) SignedPolicyCmd(goCtx context.Context, msg *types.MsgSignedPolicyCmd) (*types.MsgSignedPolicyCmdResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	resolver := &did.KeyResolver{}
	params := k.GetParams(ctx)

	payload, err := signed_policy_cmd.ValidateAndExtractCmd(ctx, params, resolver, msg.Payload, msg.Type, uint64(ctx.BlockHeight()))
	if err != nil {
		return nil, fmt.Errorf("PolicyCmd: %w", err)
	}

	result, err := dispatchPolicyCmd(ctx, &k.Keeper, payload.PolicyId, payload.Actor, payload.CreationTime, payload.Cmd)

	if err != nil {
		return nil, err
	}

	return &types.MsgSignedPolicyCmdResponse{
		Result: result,
	}, nil
}
