package keeper

import (
	"context"
	"fmt"
	"time"

	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gometrics "github.com/hashicorp/go-metrics"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/sourcenetwork/sourcehub/app/metrics"
	"github.com/sourcenetwork/sourcehub/x/acp/utils"

	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

func (k msgServer) CreatePolicy(goCtx context.Context, msg *types.MsgCreatePolicy) (res *types.MsgCreatePolicyResponse, err error) {
	start := time.Now()

	defer func() {
		metrics.ModuleMeasureWithCounter(
			types.ModuleName,
			metrics.CreatePolicy,
			start,
			err,
			[]gometrics.Label{
				metrics.NewLabel(metrics.Actor, msg.Creator),
			},
		)
	}()

	ctx := sdk.UnwrapSDKContext(goCtx)

	engine := k.getACPEngine(ctx)

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

	metrics.ModuleIncrCounterWithLabels(
		types.ModuleName,
		1,
		[]string{metrics.App, metrics.Msg, metrics.Count},
		[]gometrics.Label{
			telemetry.NewLabel(metrics.Msg, metrics.CreatePolicy),
			telemetry.NewLabel(metrics.Actor, msg.Creator),
		},
	)

	return &types.MsgCreatePolicyResponse{
		Record: rec,
	}, nil
}
