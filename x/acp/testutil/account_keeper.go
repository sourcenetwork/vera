package testutil

import (
	"context"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/sourcenetwork/vera/x/acp/types"
)

var _ types.AccountKeeper = (*AccountKeeperStub)(nil)

type AccountKeeperStub struct {
	Accounts map[string]sdk.AccountI
}

func (s *AccountKeeperStub) GetAccount(ctx context.Context, address sdk.AccAddress) sdk.AccountI {
	acc := s.Accounts[address.String()]
	return acc
}

func (s *AccountKeeperStub) GenAccount() sdk.AccountI {
	pubKey := secp256k1.GenPrivKey().PubKey()
	return s.NewAccount(pubKey)
}

func (s *AccountKeeperStub) FirstAcc() sdk.AccountI {
	for _, acc := range s.Accounts {
		return acc
	}
	return nil
}

func (s *AccountKeeperStub) NewAccount(key cryptotypes.PubKey) sdk.AccountI {
	if s.Accounts == nil {
		s.Accounts = make(map[string]sdk.AccountI)
	}

	addr := sdk.AccAddress(key.Address())
	acc := authtypes.NewBaseAccount(addr, key, 1, 1)
	s.Accounts[addr.String()] = acc
	return acc
}
