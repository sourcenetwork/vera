package access_decision

import (
	"context"
	"fmt"

	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/sourcenetwork/acp_core/pkg/errors"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"

	"github.com/sourcenetwork/sourcehub/utils"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

// DefaultExpirationDelta sets the number of blocks a Decision is valid for
const DefaultExpirationDelta uint64 = 100

type EvaluateAccessRequestsCommand struct {
	Policy     *coretypes.Policy
	Operations []*coretypes.Operation
	Actor      string

	CreationTime *types.Timestamp

	// Creator is the same as the Tx signer
	Creator sdktypes.AccountI

	// Current block height
	CurrentHeight uint64

	params *types.DecisionParams
}

func (c *EvaluateAccessRequestsCommand) Execute(ctx context.Context, engine coretypes.ACPEngineServer, repository Repository, paramsRepo ParamsRepository) (*types.AccessDecision, error) {
	err := c.validate()
	if err != nil {
		return nil, fmt.Errorf("EvaluateAccessRequest: %w", err)
	}

	err = c.evaluateRequest(ctx, engine)
	if err != nil {
		return nil, fmt.Errorf("EvaluateAccessRequest: %w", err)
	}

	c.params, err = paramsRepo.GetDefaults(ctx)
	if err != nil {
		return nil, fmt.Errorf("EvaluateAccessRequest: %w", err)
	}

	decision := c.buildDecision()

	err = repository.Set(ctx, decision)
	if err != nil {
		return nil, fmt.Errorf("EvaluateAccessRequest: %w", err)
	}

	return decision, nil
}

func (c *EvaluateAccessRequestsCommand) validate() error {
	if c.Policy == nil {
		return errors.New("policy cannot be nil", errors.ErrorType_BAD_INPUT)
	}

	if c.Operations == nil {
		return errors.New("access request cannot be nil", errors.ErrorType_BAD_INPUT)
	}

	if c.CurrentHeight == 0 {
		return errors.New("invalid height: must be nonzero postive number", errors.ErrorType_BAD_INPUT)
	}

	return nil
}

func (c *EvaluateAccessRequestsCommand) evaluateRequest(ctx context.Context, engine coretypes.ACPEngineServer) error {
	operations := utils.MapSlice(c.Operations, func(op *coretypes.Operation) *coretypes.Operation {
		return &coretypes.Operation{
			Object:     op.Object,
			Permission: op.Permission,
		}
	})
	resp, err := engine.VerifyAccessRequest(ctx, &coretypes.VerifyAccessRequestRequest{
		PolicyId: c.Policy.Id,
		AccessRequest: &coretypes.AccessRequest{
			Operations: operations,
			Actor: &coretypes.Actor{
				Id: c.Actor,
			},
		},
	})
	if err != nil {
		return err
	}
	if !resp.Valid {
		return errors.ErrorType_UNAUTHORIZED
	}

	return nil
}

func (c *EvaluateAccessRequestsCommand) buildDecision() *types.AccessDecision {
	decision := &types.AccessDecision{
		PolicyId:           c.Policy.Id,
		Params:             c.params,
		CreationTime:       c.CreationTime,
		Operations:         c.Operations,
		IssuedHeight:       c.CurrentHeight,
		Actor:              c.Actor,
		Creator:            c.Creator.GetAddress().String(),
		CreatorAccSequence: c.Creator.GetSequence(),
	}
	decision.Id = decision.ProduceId()
	return decision
}
