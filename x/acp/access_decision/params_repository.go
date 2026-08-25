package access_decision

import (
	"context"

	"github.com/sourcenetwork/vera/x/acp/types"
)

var _ ParamsRepository = (*StaticParamsRepository)(nil)

// defaultExpirationDelta sets the number of blocks a Decision is valid for
const defaultExpirationDelta uint64 = 100
const defaultProofExpirationDelta uint64 = 50

type StaticParamsRepository struct{}

func (r *StaticParamsRepository) GetDefaults(ctx context.Context) (*types.DecisionParams, error) {
	return &types.DecisionParams{
		DecisionExpirationDelta: defaultExpirationDelta,
		TicketExpirationDelta:   defaultExpirationDelta,
		ProofExpirationDelta:    defaultProofExpirationDelta,
	}, nil
}
