package sample

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
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

// CreateValidator creates a validator for testing purposes
func CreateValidator(
	t *testing.T,
	ctx sdk.Context,
	stakingKeeper *keeper.Keeper,
	operatorAddress sdk.ValAddress,
	pubKey cryptotypes.PubKey,
	bondAmount math.Int,
) stakingtypes.Validator {

	description := stakingtypes.NewDescription(
		"TestSourceValidator",
		"mysterious_identity",
		"unknown_website",
		"suspicious_security_contact",
		"missing_details",
	)

	commission := stakingtypes.NewCommission(
		math.LegacyMustNewDecFromStr("0.1"),  // commission rate
		math.LegacyMustNewDecFromStr("0.2"),  // max commission rate
		math.LegacyMustNewDecFromStr("0.01"), // max change rate
	)

	validator, err := stakingtypes.NewValidator(operatorAddress.String(), pubKey, description)
	require.NoError(t, err)

	validator.Commission = commission
	validator.Status = stakingtypes.Bonded
	validator.Tokens = bondAmount
	stakingKeeper.SetValidator(ctx, validator)

	return validator
}
