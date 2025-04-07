package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sourcenetwork/acp_core/pkg/errors"

	hubtypes "github.com/sourcenetwork/sourcehub/types"
	"github.com/sourcenetwork/sourcehub/x/acp/did"
	"github.com/sourcenetwork/sourcehub/x/acp/keeper/policy_cmd"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

func (k msgServer) DirectPolicyCmd(goCtx context.Context, msg *types.MsgDirectPolicyCmd) (*types.MsgDirectPolicyCmdResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	addr, err := hubtypes.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, fmt.Errorf("DirectPolicyCmd: %v: %w", err, types.NewErrInvalidAccAddrErr(err, msg.Creator))
	}

	acc := k.accountKeeper.GetAccount(ctx, addr)
	if acc == nil {
		return nil, fmt.Errorf("DirectPolicyCmd: %w", types.NewAccNotFoundErr(msg.Creator))
	}

	actorID, err := did.IssueDID(acc)
	if err != nil {
		return nil, errors.Wrap("DirectPolicyCmd: could not issue did to creator",
			errors.ErrorType_BAD_INPUT, errors.Pair("address", msg.Creator))
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
