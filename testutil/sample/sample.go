package sample

import (
	"github.com/cometbft/cometbft/crypto/secp256k1"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// AccAddress returns a sample account address
func AccAddress() string {
	pk := ed25519.GenPrivKey().PubKey()
	addr := pk.Address()
	return sdk.AccAddress(addr).String()
}

// RandomAccAddress returns a sample account address
func RandomAccAddress() sdk.AccAddress {
	pk := ed25519.GenPrivKey().PubKey()
	pkAddr := pk.Address()
	accAddr := sdk.AccAddress(pkAddr)
	return accAddr
}

// RandomValAddress generates a random ValidatorAddress for simulation
func RandomValAddress() sdk.ValAddress {
	valPub := secp256k1.GenPrivKey().PubKey()
	return sdk.ValAddress(valPub.Address())
}
