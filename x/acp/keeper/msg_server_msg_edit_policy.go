package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/sourcenetwork/sourcehub/x/acp/utils"

	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

func (k msgServer) EditPolicy(goCtx context.Context, msg *types.MsgEditPolicy) (*types.MsgEditPolicyResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	engine := k.getACPEngine(ctx)

	did, err := k.issueDIDFromAccountAddr(ctx, msg.Creator)
	if err != nil {
		return nil, fmt.Errorf("EditPolicy: %w", err)
	}

	ctx, err = utils.InjectPrincipal(ctx, did)
	if err != nil {
		return nil, fmt.Errorf("EditPolicy: %w", err)
	}

	response, err := engine.EditPolicy(ctx, &coretypes.EditPolicyRequest{
		PolicyId:    msg.PolicyId,
		Policy:      msg.Policy,
		MarshalType: msg.MarshalType,
	})
	if err != nil {
		return nil, fmt.Errorf("EditPolicy: %w", err)
	}

	rec, err := types.MapPolicy(response.Record)
	if err != nil {
		return nil, fmt.Errorf("EditPolicy: %w", err)
	}

	return &types.MsgEditPolicyResponse{
		RelationshipsRemoved: response.RelatinshipsRemoved,
		Record:               rec,
	}, nil
}
