package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	prototypes "github.com/cosmos/gogoproto/types"
	"github.com/sourcenetwork/acp_core/pkg/runtime"
)

var _ runtime.TimeService = (*SourcehubTimeProvider)(nil)

type SourcehubTimeProvider struct{}

func (p *SourcehubTimeProvider) GetNow(goCtx context.Context) (*prototypes.Timestamp, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	time := ctx.BlockTime()
	ts, err := prototypes.TimestampProto(time)
	if err != nil {
		return nil, err
	}
	return ts, nil
}
