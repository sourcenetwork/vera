package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sourcenetwork/acp_core/pkg/errors"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"

	"github.com/sourcenetwork/sourcehub/x/acp/access_decision"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

func (k *Keeper) CheckAccess(goCtx context.Context, msg *types.MsgCheckAccess) (*types.MsgCheckAccessResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	eventManager := ctx.EventManager()

	repository := k.getAccessDecisionRepository(ctx)
	paramsRepository := access_decision.StaticParamsRepository{}
	engine := k.getACPEngine(ctx)

	record, err := engine.GetPolicy(goCtx, &coretypes.GetPolicyRequest{Id: msg.PolicyId})
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, errors.ErrPolicyNotFound(msg.PolicyId)
	}

	creatorAddr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, types.NewErrInvalidAccAddrErr(err, msg.Creator)
	}
	creatorAcc := k.accountKeeper.GetAccount(ctx, creatorAddr)
	if creatorAcc == nil {
		return nil, types.NewAccNotFoundErr(msg.Creator)
	}

	ts, err := types.TimestampFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	cmd := access_decision.EvaluateAccessRequestsCommand{
		Policy:        record.Record.Policy,
		Operations:    msg.AccessRequest.Operations,
		Actor:         msg.AccessRequest.Actor.Id,
		CreationTime:  ts,
		Creator:       creatorAcc,
		CurrentHeight: uint64(ctx.BlockHeight()),
	}
	decision, err := cmd.Execute(goCtx, engine, repository, &paramsRepository)
	if err != nil {
		return nil, err
	}

	err = eventManager.EmitTypedEvent(&coretypes.EventAccessDecisionCreated{
		Creator:    msg.Creator,
		PolicyId:   msg.PolicyId,
		DecisionId: decision.Id,
		Actor:      decision.Actor,
	})
	if err != nil {
		return nil, err
	}

	return &types.MsgCheckAccessResponse{
		Decision: decision,
	}, nil
}
