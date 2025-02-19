package test

import (
	"testing"

	"cosmossdk.io/math"
	txsigning "cosmossdk.io/x/tx/signing"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type EncodingConfig struct {
	InterfaceRegistry types.InterfaceRegistry
	Codec             codec.Codec
	TxConfig          client.TxConfig
	Amino             *codec.LegacyAmino
}

// CreateTestValidator creates a validator for testing purposes.
func CreateTestValidator(
	t *testing.T,
	ctx sdk.Context,
	stakingKeeper *stakingkeeper.Keeper,
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

// CreateTestEncodingConfig creates encoding configuration for testing purposes.
func CreateTestEncodingConfig() EncodingConfig {
	interfaceRegistry, err := types.NewInterfaceRegistryWithOptions(types.InterfaceRegistryOptions{
		ProtoFiles: proto.HybridResolver,
		SigningOptions: txsigning.Options{
			AddressCodec:          addresscodec.NewBech32Codec("source"),
			ValidatorAddressCodec: addresscodec.NewBech32Codec("sourcevaloper"),
		},
	})
	if err != nil {
		panic(err)
	}

	banktypes.RegisterInterfaces(interfaceRegistry)
	cryptocodec.RegisterInterfaces(interfaceRegistry)
	stakingtypes.RegisterInterfaces(interfaceRegistry)
	authtypes.RegisterInterfaces(interfaceRegistry)
	distrtypes.RegisterInterfaces(interfaceRegistry)

	protoCodec := codec.NewProtoCodec(interfaceRegistry)

	return EncodingConfig{
		InterfaceRegistry: interfaceRegistry,
		Codec:             protoCodec,
		TxConfig:          tx.NewTxConfig(protoCodec, tx.DefaultSignModes),
		Amino:             codec.NewLegacyAmino(),
	}
}
