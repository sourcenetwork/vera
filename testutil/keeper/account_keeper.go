package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/sourcenetwork/vera/x/acp/types"
)

var _ types.AccountKeeper = (*AccountKeeperStub)(nil)

type AccountKeeperStub struct{}

func (s *AccountKeeperStub) GetAccount(ctx context.Context, address sdk.AccAddress) sdk.AccountI {
	return &authtypes.BaseAccount{
		Address:  string(address),
		Sequence: 1,
	}
}
