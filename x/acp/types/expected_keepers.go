package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	icatypes "github.com/sourcenetwork/sourcehub/x/ica/types"
)

// AccountKeeper defines the expected interface for the Account module.
type AccountKeeper interface {
	GetAccount(context.Context, sdk.AccAddress) sdk.AccountI
	// Methods imported from account should be defined here
}

// BankKeeper defines the expected interface for the Bank module.
type BankKeeper interface {
	SpendableCoins(context.Context, sdk.AccAddress) sdk.Coins
	// Methods imported from bank should be defined here
}

// ParamSubspace defines the expected Subspace interface for parameters.
type ParamSubspace interface {
	Get(context.Context, []byte, interface{})
	Set(context.Context, []byte, interface{})
}

// ICAKeeper defines the expected interface for the ICA module.
type ICAKeeper interface {
	GetICAConnection(ctx sdk.Context, icaAddress string) (icatypes.ICAConnection, bool)
	SetICAConnection(ctx sdk.Context, icaAddress, controllerAddress, controllerChainID, connectionID string) error
}
