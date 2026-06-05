package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type AccountKeeper interface {
	GetAccount(context.Context, sdk.AccAddress) sdk.AccountI
	GetModuleAccount(context.Context, string) sdk.ModuleAccountI
	SetAccount(context.Context, sdk.AccountI)
}
