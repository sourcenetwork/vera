package sdk

import (
	"context"
	"fmt"
	"math"

	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	xauthsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/sourcenetwork/sourcehub/app"
	antetypes "github.com/sourcenetwork/sourcehub/app/ante/types"
	"github.com/sourcenetwork/sourcehub/app/params"
	appparams "github.com/sourcenetwork/sourcehub/app/params"
)

type TxBuilder struct {
	chainID       string
	gasLimit      uint64
	feeGranter    sdk.AccAddress
	authClient    authtypes.QueryClient
	txClient      txtypes.ServiceClient
	txCfg         client.TxConfig
	feeTokenDenom string
	feeAmt        int64
	account       authtypes.BaseAccount
	gasAdjustment float64
	bearerToken   string // JWS bearer token for DID-based authorization
}

func NewTxBuilder(opts ...TxBuilderOpt) (TxBuilder, error) {
	registry := cdctypes.NewInterfaceRegistry()
	//acptypes.RegisterInterfaces(registry)
	cfg := authtx.NewTxConfig(
		codec.NewProtoCodec(registry),
		[]signing.SignMode{
			signing.SignMode_SIGN_MODE_DIRECT,
		},
	)

	builder := TxBuilder{
		txCfg:         cfg,
		chainID:       DefaultChainID,
		feeTokenDenom: appparams.DefaultBondDenom,
		feeAmt:        200,
		gasLimit:      200000,
		gasAdjustment: 1.2,
	}

	for _, opt := range opts {
		err := opt(&builder)
		if err != nil {
			return TxBuilder{}, err
		}
	}

	if builder.authClient == nil {
		return TxBuilder{}, fmt.Errorf("TxBuilder: Auth GRPC Client is required: use either WithAuthQueryClient or WithSDKClient to set it")
	}

	return builder, nil
}

// Build builds a SourceHub Tx containing msgs in MsgSet.
// The returned Tx can then be broadcast.
func (b *TxBuilder) Build(ctx context.Context, signer TxSigner, set *MsgSet) (xauthsigning.Tx, error) {
	return b.BuildFromMsgs(ctx, signer, set.GetMsgs()...)
}

// BuildFromMsgs builds a SourceHub Tx containing msgs.
// The returned Tx can then be broadcast.
func (b *TxBuilder) BuildFromMsgs(ctx context.Context, signer TxSigner, msgs ...sdk.Msg) (xauthsigning.Tx, error) {
	builder, err := b.initTx(ctx, signer, msgs...)
	if err != nil {
		return nil, err
	}

	if err := b.evaluateTx(ctx, builder); err != nil {
		return nil, err
	}

	tx, err := b.finalizeTx(ctx, signer, builder)
	if err != nil {
		return nil, err
	}

	return tx, nil
}

func (b *TxBuilder) initTx(ctx context.Context, signer TxSigner, msgs ...sdk.Msg) (client.TxBuilder, error) {
	txBuilder := b.txCfg.NewTxBuilder()
	err := txBuilder.SetMsgs(msgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to set msgs: %v", err)
	}

	// Set JWS bearer token extension option if provided
	if b.bearerToken != "" {
		jwsOpt := &antetypes.JWSExtensionOption{
			BearerToken: b.bearerToken,
		}
		any, err := cdctypes.NewAnyWithValue(jwsOpt)
		if err != nil {
			return nil, fmt.Errorf("failed to pack JWS extension option: %v", err)
		}
		if extBuilder, ok := txBuilder.(client.ExtendedTxBuilder); ok {
			extBuilder.SetExtensionOptions(any)
		} else {
			return nil, fmt.Errorf("txBuilder does not support extension options")
		}
	}

	txBuilder.SetGasLimit(b.gasLimit)
	feeAmt := sdk.NewCoins(sdk.NewInt64Coin(b.feeTokenDenom, b.feeAmt))
	txBuilder.SetFeeAmount(feeAmt)
	if b.feeGranter != nil {
		txBuilder.SetFeeGranter(b.feeGranter)
	}

	acc, err := b.getAccount(ctx, signer.GetAccAddress())
	if err != nil {
		return nil, err
	}
	b.account = acc

	// NOTE: The following snippet was based on the Cosmos-SDK documentation and codebase
	// See:
	// https://docs.cosmos.network/v0.50/user/run-node/txs#signing-a-transaction-1
	// https://github.com/cosmos/cosmos-sdk/blob/v0.50.6/client/tx/tx.go#L284
	sigV2 := signing.SignatureV2{
		PubKey: signer.GetPrivateKey().PubKey(),
		Data: &signing.SingleSignatureData{
			SignMode:  signing.SignMode_SIGN_MODE_DIRECT,
			Signature: nil,
		},
		Sequence: acc.GetSequence(),
	}

	err = txBuilder.SetSignatures(sigV2)
	if err != nil {
		return nil, fmt.Errorf("set signatures: %v", err)
	}

	return txBuilder, nil
}

// evaluateTx simulates the tx and adjusts gas and fee amounts accordingly.
func (b *TxBuilder) evaluateTx(ctx context.Context, txBuilder client.TxBuilder) error {
	if b.txClient == nil {
		return nil
	}

	encoder := b.txCfg.TxEncoder()
	txBytes, err := encoder(txBuilder.GetTx())
	if err != nil {
		return fmt.Errorf("encode tx for simulation: %w", err)
	}

	simRes, err := b.txClient.Simulate(ctx, &txtypes.SimulateRequest{TxBytes: txBytes})
	if err != nil {
		return fmt.Errorf("simulate tx: %w", err)
	}

	gasUsed := simRes.GetGasInfo().GetGasUsed()
	if gasUsed == 0 {
		return nil
	}

	adjusted := uint64(math.Ceil(float64(gasUsed) * b.gasAdjustment))
	txBuilder.SetGasLimit(adjusted)
	b.gasLimit = adjusted

	minGasPrice := sdkmath.LegacyMustNewDecFromStr(appparams.DefaultMinGasPrice)
	denomMultiplier := sdkmath.LegacyNewDec(1)
	if b.feeTokenDenom == params.MicroCreditDenom {
		denomMultiplier = sdkmath.LegacyNewDec(appparams.CreditFeeMultiplier)
	}
	gasInt := sdkmath.NewIntFromUint64(adjusted)
	requiredAmount := minGasPrice.Mul(sdkmath.LegacyNewDecFromInt(gasInt)).Mul(denomMultiplier).Ceil().RoundInt()
	txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(b.feeTokenDenom, requiredAmount)))
	if requiredAmount.IsInt64() {
		b.feeAmt = requiredAmount.Int64()
	}

	if b.feeGranter != nil {
		txBuilder.SetFeeGranter(b.feeGranter)
	}

	return nil
}

func (b *TxBuilder) finalizeTx(ctx context.Context, signer TxSigner, txBuilder client.TxBuilder) (xauthsigning.Tx, error) {
	signerData := xauthsigning.SignerData{
		ChainID:       b.chainID,
		AccountNumber: b.account.GetAccountNumber(),
		Sequence:      b.account.GetSequence(),
		PubKey:        signer.GetPrivateKey().PubKey(),
	}

	sigV2, err := tx.SignWithPrivKey(
		ctx,
		signing.SignMode_SIGN_MODE_DIRECT,
		signerData,
		txBuilder,
		signer.GetPrivateKey(),
		b.txCfg,
		b.account.GetSequence())
	if err != nil {
		return nil, fmt.Errorf("sign tx: %v", err)
	}

	err = txBuilder.SetSignatures(sigV2)
	if err != nil {
		return nil, fmt.Errorf("set signatures: %v", err)
	}

	return txBuilder.GetTx(), nil
}

func (c *TxBuilder) getAccount(ctx context.Context, addr string) (authtypes.BaseAccount, error) {
	msg := authtypes.QueryAccountRequest{
		Address: addr,
	}

	resp, err := c.authClient.Account(ctx, &msg)
	if err != nil {
		return authtypes.BaseAccount{}, fmt.Errorf("fetching account: %v", err)
	}

	acc := authtypes.BaseAccount{}
	err = acc.Unmarshal(resp.Account.Value)
	if err != nil {
		return acc, fmt.Errorf("unmarshaling account: %v", err)
	}

	return acc, nil
}

// TxBuilderOpt is a constructor option to initialize a TxBuilder
type TxBuilderOpt func(*TxBuilder) error

// WithChainID specifies the ChainID to which the Tx will be signed to.
// Defaults to the most recent SourceHub chain deployment
func WithChainID(id string) TxBuilderOpt {
	return func(b *TxBuilder) error {
		b.chainID = id
		return nil
	}
}

// WithMicroOpen configures TxBuilder to build Txs paid using open tokens
func WithMicroOpen() TxBuilderOpt {
	return func(b *TxBuilder) error {
		b.feeTokenDenom = params.MicroOpenDenom
		return nil
	}
}

// WithMicroCredit configures TxBuilder to build Txs paid using credits
func WithMicroCredit() TxBuilderOpt {
	return func(b *TxBuilder) error {
		b.feeTokenDenom = params.MicroCreditDenom
		return nil
	}
}

// WithMainnetChainID specifies the ChainID to be SourceHub's main net
//func WithMainnetChainID() Option {return nil }

// WithTestnetChainID specifies the ChainID to be SourceHub's latest test net
func WithTestnetChainID() TxBuilderOpt {
	return func(b *TxBuilder) error {
		b.chainID = TestnetLatest
		return nil
	}
}

// WithFeeToken specifies the fee value
func WithFeeAmount(fee int64) TxBuilderOpt {
	return func(b *TxBuilder) error {
		b.feeAmt = fee
		return nil
	}
}

// WithFeeToken specifies the token denominator to use for the fee
func WithFeeToken(denom string) TxBuilderOpt {
	return func(b *TxBuilder) error {
		b.feeTokenDenom = denom
		return nil
	}
}

// WithGasLimit configures the maxium
func WithGasLimit(limit uint64) TxBuilderOpt {
	return func(b *TxBuilder) error {
		b.gasLimit = limit
		return nil
	}
}

// WithFeeGranter sets the fee granter for the Tx.
// The fee granter pays for the executed Tx.
// Fee grants are configured by cosmos x/feegrant module.
// See: https://pkg.go.dev/github.com/cosmos/cosmos-sdk/x/feegrant#section-readme
func WithFeeGranter(acc string) TxBuilderOpt {
	return func(b *TxBuilder) error {
		addr, err := sdk.AccAddressFromBech32(acc)
		if err != nil {
			return fmt.Errorf("invalid feegrant account: %w", err)
		}
		b.feeGranter = addr

		return nil
	}
}

// WithGRPCConnection sets the GRPC Connection to be used in the Builder.
// The connection should point to a trusted  SourceHub node which will be used to perform
// Cosmos Query client queries.
// Note: Cosmos queries are not verifiable therefore only trusted RPC nodes should be used.
func WithAuthQueryClient(client authtypes.QueryClient) TxBuilderOpt {
	return func(b *TxBuilder) error {
		b.authClient = client
		return nil
	}
}

// WithClient uses an SDK Client to retrieve the required connections.
func WithSDKClient(client *Client) TxBuilderOpt {
	return func(b *TxBuilder) error {
		b.authClient = client.AuthQueryClient()
		b.txClient = client.txClient
		return nil
	}
}

// WithBearerToken sets a JWS bearer token for DID-based authorization.
// The bearer token will be added as an extension option to the transaction.
func WithBearerToken(token string) TxBuilderOpt {
	return func(b *TxBuilder) error {
		b.bearerToken = token
		return nil
	}
}

// FIXME what's a better way of doing this for a lib?
func init() {
	app.SetConfig(false)
}
