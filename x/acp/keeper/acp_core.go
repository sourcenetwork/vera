package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	prototypes "github.com/cosmos/gogoproto/types"
	"github.com/sourcenetwork/acp_core/pkg/runtime"
)

var _ runtime.TimeService = (*VeraTimeProvider)(nil)

// VeraTimeProvider implements acp_core's TimeService
// in order to synchronize the block time with acp_core's engine time.
type VeraTimeProvider struct{}

// GetNow implements TimeService
func (p *VeraTimeProvider) GetNow(goCtx context.Context) (*prototypes.Timestamp, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	time := ctx.BlockTime()
	ts, err := prototypes.TimestampProto(time)
	if err != nil {
		return nil, err
	}
	return ts, nil
}
