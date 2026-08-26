package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	coretypes "github.com/sourcenetwork/vera/x/core/types"
)

// AccountKeeper defines the expected interface for the Account module.
type AccountKeeper interface {
	GetAccount(context.Context, sdk.AccAddress) sdk.AccountI
}

// BankKeeper defines the expected interface for the Bank module.
type BankKeeper interface {
	SpendableCoins(context.Context, sdk.AccAddress) sdk.Coins
}

// CoreKeeper defines the expected interface for the core module.
type CoreKeeper interface {
	GetICAConnection(ctx sdk.Context, icaAddress string) (coretypes.ICAConnection, bool)
	SetICAConnection(ctx sdk.Context, icaAddress, controllerAddress, controllerChainID, connectionID string) error
}
