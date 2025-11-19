package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	prototypes "github.com/cosmos/gogoproto/types"
	"github.com/sourcenetwork/acp_core/pkg/runtime"
)

var _ runtime.TimeService = (*SourceHubTimeProvider)(nil)

// SourceHubTimeProvider implements acp_core's TimeService
// in order to syncrhonize the block time with acp_core's engine time.
type SourceHubTimeProvider struct{}

// GetNow implements TimeService
func (p *SourceHubTimeProvider) GetNow(goCtx context.Context) (*prototypes.Timestamp, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	time := ctx.BlockTime()
	ts, err := prototypes.TimestampProto(time)
	if err != nil {
		return nil, err
	}
	return ts, nil
}
