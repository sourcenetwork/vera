package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k *Keeper) EndBlock(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	now := uint64(sdkCtx.BlockTime().Unix())
	k.DeleteExpiredAcceptedReportPairs(ctx, now)
	return nil
}
