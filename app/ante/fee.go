package ante

import (
	"bytes"
	"context"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	"github.com/cosmos/cosmos-sdk/x/auth/types"

	appparams "github.com/sourcenetwork/vera/app/params"
)

// TxFeeChecker validates provided fee and returns the effective fee and tx priority.
type TxFeeChecker func(ctx sdk.Context, tx sdk.Tx) (sdk.Coins, int64, error)

// CustomDeductFeeDecorator deducts fees from the fee payer.
type CustomDeductFeeDecorator struct {
	accountKeeper  ante.AccountKeeper
	bankKeeper     types.BankKeeper
	feegrantKeeper ante.FeegrantKeeper
	txFeeChecker   TxFeeChecker
	coreKeeper     CoreKeeper
}

// NewCustomDeductFeeDecorator initializes custom deduct fee decorator with a fee checker.
func NewCustomDeductFeeDecorator(
	ak ante.AccountKeeper,
	bk types.BankKeeper,
	fk ante.FeegrantKeeper,
	hk CoreKeeper,
	tfc TxFeeChecker,
) CustomDeductFeeDecorator {

	if tfc == nil {
		tfc = func(ctx sdk.Context, tx sdk.Tx) (sdk.Coins, int64, error) {
			return checkTxFeeWithMinGasPrices(ctx, tx, hk)
		}
	}

	return CustomDeductFeeDecorator{
		accountKeeper:  ak,
		bankKeeper:     bk,
		feegrantKeeper: fk,
		coreKeeper:     hk,
		txFeeChecker:   tfc,
	}
}

// AnteHandle performs fee validation and deduction for transactions. Transactions at genesis bypass fee validation.
func (cdfd CustomDeductFeeDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (
	sdk.Context, error) {

	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return ctx, errorsmod.Wrap(sdkerrors.ErrTxDecode, "tx must be a FeeTx")
	}

	// Skip fee validation and deduction for transactions at genesis
	if ctx.BlockHeight() == 0 {
		return next(ctx, tx, simulate)
	}

	if !simulate && ctx.BlockHeight() > 0 && feeTx.GetGas() == 0 {
		return ctx, errorsmod.Wrap(sdkerrors.ErrInvalidGasLimit, "must provide positive gas")
	}

	var (
		priority int64
		err      error
	)

	fees := feeTx.GetFee()

	if !simulate {
		// Check tx fees with min gas prices
		fees, priority, err = cdfd.txFeeChecker(ctx, tx)
		if err != nil {
			return ctx, err
		}
	}

	if err := cdfd.checkDeductFee(ctx, tx, fees); err != nil {
		return ctx, err
	}

	newCtx := ctx.WithPriority(priority)

	return next(newCtx, tx, simulate)
}

// checkTxFeeWithMinGasPrices checks if the tx fee with denom fee multiplier >= min gas price of the validator.
// Enforces the DefaultMinGasPrice to prevent spam if minimum gas price was set to 0 by the validator.
// NOTE: Always returns 0 for transaction priority because we handle TxPriority in priority_lane.go.
func checkTxFeeWithMinGasPrices(ctx sdk.Context, tx sdk.Tx, coreKeeper CoreKeeper) (sdk.Coins, int64, error) {
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return nil, 0, errorsmod.Wrap(sdkerrors.ErrTxDecode, "tx must be a FeeTx")
	}

	fees := feeTx.GetFee()
	gas := feeTx.GetGas()

	// Allow zero-fee transactions if allowed by app config and the "--fees" flag is omitted
	if fees.Empty() {
		if coreKeeper != nil && coreKeeper.GetChainConfig(ctx).AllowZeroFeeTxs {
			return fees, 0, nil
		}
		return nil, 0, sdkerrors.ErrInsufficientFee.Wrap("zero fees are not allowed")
	}

	if fees.Len() > 1 {
		return nil, 0, errorsmod.Wrapf(sdkerrors.ErrInvalidCoins,
			"only one fee denomination is allowed, got: %s", fees.String())
	}

	// Validate provided fees if this is a CheckTx
	if ctx.IsCheckTx() {
		fee := fees[0]
		minGasPrice := ctx.MinGasPrices().AmountOf(fee.Denom)

		// Denoms missing from MinGasPrices() are not supported
		if minGasPrice.IsNil() {
			return nil, 0, errorsmod.Wrapf(
				sdkerrors.ErrInvalidCoins,
				"invalid fee denom: %s is not supported, available fee denoms: %s",
				fee.Denom, ctx.MinGasPrices(),
			)
		}

		// Enforce default min gas price to prevent spam if it was set to 0 by the validator
		if minGasPrice.IsZero() {
			minGasPrice = math.LegacyMustNewDecFromStr(appparams.DefaultMinGasPrice)
		}

		// Calculate required fee by multiplying minimum gas price by gas limit and denom multiplier
		denomFeeMultiplier := math.LegacyOneDec()
		if fee.Denom == appparams.MicroCreditDenom {
			denomFeeMultiplier = math.LegacyNewDec(appparams.CreditFeeMultiplier)
		}
		requiredAmount := minGasPrice.Mul(math.LegacyNewDec(int64(gas))).Mul(denomFeeMultiplier).Ceil().RoundInt()

		// Make sure that provided fee is at least the required amount
		if fee.Amount.LT(requiredAmount) {
			return nil, 0, errorsmod.Wrapf(
				sdkerrors.ErrInsufficientFee,
				"insufficient fee; got: %s required: %s",
				fee, sdk.NewCoin(fee.Denom, requiredAmount),
			)
		}
	}

	return fees, 0, nil
}

// checkDeductFee checks and deducts fees from the fee payer.
func (cdfd CustomDeductFeeDecorator) checkDeductFee(ctx sdk.Context, sdkTx sdk.Tx, fees sdk.Coins) error {
	feeTx, ok := sdkTx.(sdk.FeeTx)
	if !ok {
		return errorsmod.Wrap(sdkerrors.ErrTxDecode, "tx must be a FeeTx")
	}

	if addr := cdfd.accountKeeper.GetModuleAddress(types.FeeCollectorName); addr == nil {
		return fmt.Errorf("fee collector module account (%s) has not been set", types.FeeCollectorName)
	}

	feePayer := feeTx.FeePayer()
	feeGranter := feeTx.FeeGranter()

	// Determine who will pay the fees
	deductFeesFrom, err := cdfd.handleFeegrant(ctx, feeGranter, feePayer, fees, sdkTx.GetMsgs())
	if err != nil {
		return err
	}

	deductFeesFromAcc := cdfd.accountKeeper.GetAccount(ctx, deductFeesFrom)
	if deductFeesFromAcc == nil {
		return sdkerrors.ErrUnknownAddress.Wrapf("fee payer address: %s does not exist", deductFeesFrom)
	}

	if !fees.IsZero() {
		err := deductFees(cdfd.bankKeeper, ctx, deductFeesFromAcc, fees)
		if err != nil {
			return err
		}
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		sdk.EventTypeTx,
		sdk.NewAttribute(sdk.AttributeKeyFee, fees.String()),
		sdk.NewAttribute(sdk.AttributeKeyFeePayer, sdk.AccAddress(deductFeesFrom).String()),
	))

	return nil
}

// handleFeegrant determines who should pay for fees based on feegrant configuration.
// It handles both DID-based feegrants (when DID is extracted from context) and standard feegrants.
// Returns the address that should be charged for fees, or an error if feegrant validation fails.
func (cdfd CustomDeductFeeDecorator) handleFeegrant(
	ctx sdk.Context,
	feeGranter []byte,
	feePayer sdk.AccAddress,
	fees sdk.Coins,
	msgs []sdk.Msg,
) (sdk.AccAddress, error) {
	trustedRelayFeeGranter := getTrustedRelayFeeGranterFromContext(ctx)
	if trustedRelayFeeGranter != "" {
		if len(feeGranter) == 0 {
			return nil, sdkerrors.ErrUnauthorized.Wrap("trusted relay fee granter is missing")
		}
		feeGranterAddr := sdk.AccAddress(feeGranter)
		if feeGranterAddr.String() != trustedRelayFeeGranter {
			return nil, sdkerrors.ErrUnauthorized.Wrapf(
				"relay fee granter %s does not match trusted account %s",
				feeGranterAddr,
				trustedRelayFeeGranter,
			)
		}
		if bytes.Equal(feeGranterAddr, feePayer) {
			return nil, sdkerrors.ErrUnauthorized.Wrap("relay worker must have a fee grant")
		}
		if cdfd.feegrantKeeper == nil {
			return nil, sdkerrors.ErrInvalidRequest.Wrap("fee grants are not enabled")
		}
		if err := cdfd.feegrantKeeper.UseGrantedFees(ctx, feeGranterAddr, feePayer, fees, msgs); err != nil {
			return nil, errorsmod.Wrapf(err, "%s does not authorize relay worker %s", feeGranterAddr, feePayer)
		}
		return feeGranterAddr, nil
	}

	extractedDID := getExtractedDIDFromContext(ctx)

	// If DID was extracted, try to use the first available grant for that DID
	if extractedDID != "" && cdfd.feegrantKeeper != nil {
		if didKeeper, ok := cdfd.feegrantKeeper.(interface {
			UseFirstAvailableDIDGrant(ctx context.Context, granteeDID string, fee sdk.Coins, msgs []sdk.Msg) (sdk.AccAddress, error)
		}); ok {
			usedGranter, err := didKeeper.UseFirstAvailableDIDGrant(ctx, extractedDID, fees, msgs)
			if err == nil {
				return usedGranter, nil
			}
		}
	}

	// If no DID extracted, try to use fee granter from the tx
	if feeGranter != nil {
		feeGranterAddr := sdk.AccAddress(feeGranter)

		if cdfd.feegrantKeeper == nil {
			return nil, sdkerrors.ErrInvalidRequest.Wrap("fee grants are not enabled")
		}

		if !bytes.Equal(feeGranterAddr, feePayer) {
			err := cdfd.feegrantKeeper.UseGrantedFees(ctx, feeGranterAddr, feePayer, fees, msgs)
			if err != nil {
				return nil, errorsmod.Wrapf(err, "%s does not allow to pay fees for %s", feeGranter, feePayer)
			}
		}

		return feeGranterAddr, nil
	}

	// If there is no fee grant, we deduct from the fee payer
	return feePayer, nil
}

// deductFees deducts fees from the given account.
func deductFees(bankKeeper types.BankKeeper, ctx sdk.Context, acc sdk.AccountI, fees sdk.Coins) error {
	if !fees.IsValid() {
		return errorsmod.Wrapf(sdkerrors.ErrInsufficientFee, "invalid fee amount: %s", fees)
	}

	err := bankKeeper.SendCoinsFromAccountToModule(ctx, acc.GetAddress(), types.FeeCollectorName, fees)
	if err != nil {
		return errorsmod.Wrap(sdkerrors.ErrInsufficientFunds, err.Error())
	}

	return nil
}
