package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sourcenetwork/acp_core/pkg/errors"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	hubtypes "github.com/sourcenetwork/sourcehub/types"
	"github.com/sourcenetwork/sourcehub/x/acp/did"
	"github.com/sourcenetwork/sourcehub/x/acp/utils"

	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

const (
	txHashMapKey  = "tx_hash"
	creatorMapKey = "creator"
)

func (k msgServer) CreatePolicy(goCtx context.Context, msg *types.MsgCreatePolicy) (*types.MsgCreatePolicyResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	engine := k.GetACPEngine(ctx)

	addr, err := hubtypes.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, fmt.Errorf("CreatePolicy: %v: %w", err, types.NewErrInvalidAccAddrErr(err, msg.Creator))
	}

	acc := k.accountKeeper.GetAccount(ctx, addr)
	if acc == nil {
		return nil, fmt.Errorf("CreatePolicy: %w", types.NewAccNotFoundErr(msg.Creator))
	}

	actorID, err := did.IssueDID(acc)
	if err != nil {
		return nil, errors.Wrap("DirectPolicyCmd: could not issue did to creator",
			errors.ErrorType_BAD_INPUT, errors.Pair("address", msg.Creator))
	}

	metadata, err := types.BuildACPSuppliedMetadata(ctx, actorID, msg.Creator)
	if err != nil {
		return nil, err
	}

	ctx, err = utils.InjectPrincipal(ctx, actorID)
	if err != nil {
		return nil, err
	}

	coreResult, err := engine.CreatePolicy(goCtx, &coretypes.CreatePolicyRequest{
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
