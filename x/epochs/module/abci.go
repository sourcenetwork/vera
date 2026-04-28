package epochs

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/sourcenetwork/sourcehub/x/epochs/keeper"
)

// BeginBlocker delegates to the keeper's BeginBlocker.
func BeginBlocker(ctx sdk.Context, k *keeper.Keeper) error {
	return k.BeginBlocker(ctx)
}
