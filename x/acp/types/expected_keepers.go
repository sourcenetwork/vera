package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	hubtypes "github.com/sourcenetwork/sourcehub/x/hub/types"
)

// AccountKeeper defines the expected interface for the Account module.
type AccountKeeper interface {
	GetAccount(context.Context, sdk.AccAddress) sdk.AccountI
}

// BankKeeper defines the expected interface for the Bank module.
type BankKeeper interface {
	SpendableCoins(context.Context, sdk.AccAddress) sdk.Coins
}

// HubKeeper defines the expected interface for the hub module.
type HubKeeper interface {
	GetICAConnection(ctx sdk.Context, icaAddress string) (hubtypes.ICAConnection, bool)
	SetICAConnection(ctx sdk.Context, icaAddress, controllerAddress, controllerChainID, connectionID string) error
}
