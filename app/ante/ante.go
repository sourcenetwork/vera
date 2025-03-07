package ante

import (
	"cosmossdk.io/x/tx/signing"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	ibcante "github.com/cosmos/ibc-go/v8/modules/core/ante"
	ibckeeper "github.com/cosmos/ibc-go/v8/modules/core/keeper"
)

// NewAnteHandler extends the default AnteHandler with custom decorators.
func NewAnteHandler(
	accountKeeper ante.AccountKeeper,
	bankKeeper bankkeeper.Keeper,
	feegrantKeeper ante.FeegrantKeeper,
	signModeHandler *signing.HandlerMap,
	sigGasConsumer ante.SignatureVerificationGasConsumer,
	channelKeeper *ibckeeper.Keeper,
	TxEncoder sdk.TxEncoder,
) sdk.AnteHandler {
	return sdk.ChainAnteDecorators(
		// Wraps panics with the string format of the transaction.
		NewHandlePanicDecorator(),
		// Initializes the context with the gas meter. Must run before any gas consumption.
		ante.NewSetUpContextDecorator(),
		// Ensures that the transaction has no extension options.
		ante.NewExtensionOptionsDecorator(nil),
		// Performs basic validation on the transaction.
		ante.NewValidateBasicDecorator(),
		// Ensures that the tx has not exceeded the height timeout.
		ante.NewTxTimeoutHeightDecorator(),
		// Ensures that the memo does not exceed the allowed max length.
		ante.NewValidateMemoDecorator(accountKeeper),
		// Ensures that the gas limit covers the cost for transaction size. Consumes gas from the gas meter.
		ante.NewConsumeGasForTxSizeDecorator(accountKeeper),
		// Ensures that the fee payer (fee granter or first signer) has enough funds to pay for the tx.
		// Deducts fees from the fee payer and sets the tx priority in context.
		ante.NewDeductFeeDecorator(accountKeeper, bankKeeper, feegrantKeeper, nil),
		// Sets public keys in the context for the fee payer and signers. Must happen before signature checks.
		ante.NewSetPubKeyDecorator(accountKeeper),
		// Ensures that the number of signatures does not exceed the tx's signature limit.
		ante.NewValidateSigCountDecorator(accountKeeper),
		// Ensures that the tx's gas limit is > the gas consumed based on signature verification. Consumes gas from the gas meter.
		ante.NewSigGasConsumeDecorator(accountKeeper, sigGasConsumer),
		// Validates signatures and ensure each signer's nonce matches its account sequence. No gas consumed from the gas meter.
		ante.NewSigVerificationDecorator(accountKeeper, signModeHandler),
		// Increments the sequence number (nonce) for all tx signers.
		ante.NewIncrementSequenceDecorator(accountKeeper),
		// Checks that the tx is not a duplicate IBC packet or update message.
		ibcante.NewRedundantRelayDecorator(channelKeeper),
	)
}

var DefaultSigVerificationGasConsumer = ante.DefaultSigVerificationGasConsumer
