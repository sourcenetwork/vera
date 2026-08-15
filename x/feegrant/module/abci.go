package module

import (
	"context"
	"errors"

	"github.com/sourcenetwork/vera/x/feegrant/keeper"
)

func EndBlocker(ctx context.Context, k keeper.Keeper) error {
	// 200 is an arbitrary value, we can change it later if needed
	errAllowances := k.RemoveExpiredAllowances(ctx, 200)
	errDidAllowances := k.RemoveExpiredDIDAllowances(ctx, 200)
	return errors.Join(errAllowances, errDidAllowances)
}
