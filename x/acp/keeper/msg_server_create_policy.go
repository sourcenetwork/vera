package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/sourcenetwork/sourcehub/x/acp/utils"

	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

func (k msgServer) CreatePolicy(goCtx context.Context, msg *types.MsgCreatePolicy) (*types.MsgCreatePolicyResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	engine := k.GetACPEngine(ctx)

	actorID, err := k.issueDIDFromAccountAddr(ctx, msg.Creator)
	if err != nil {
		return nil, fmt.Errorf("CreatePolicy: %w", err)
	}

	metadata, err := types.BuildACPSuppliedMetadata(ctx, actorID, msg.Creator)
	if err != nil {
		return nil, err
	}

	ctx, err = utils.InjectPrincipal(ctx, actorID)
	if err != nil {
		return nil, err
	}

	coreResult, err := engine.CreatePolicy(ctx, &coretypes.CreatePolicyRequest{
		Policy:      msg.Policy,
		MarshalType: msg.MarshalType,
		Metadata:    metadata,
	})
	if err != nil {
		return nil, fmt.Errorf("CreatePolicy: %w", err)
	}

	rec, err := types.MapPolicy(coreResult.Record)
	if err != nil {
		return nil, fmt.Errorf("CreatePolicy: %w", err)
	}
	// TODO event

	return &types.MsgCreatePolicyResponse{
		Record: rec,
	}, nil
}
