package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sourcenetwork/sourcehub/x/acp/registration"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

// EndBlocker called at every block, update validator set
func (k *Keeper) EndBlocker(goCtx context.Context) ([]*types.RegistrationsCommitment, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	var repo registration.CommitmentRepository
	commitments, err := registration.FlagExpiredCommitments(ctx, repo)
	if err != nil {
		return nil, err
	}

	return commitments, nil
}
