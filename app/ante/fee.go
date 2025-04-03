package ante

import (
	"bytes"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	"github.com/cosmos/cosmos-sdk/x/auth/types"

	appparams "github.com/sourcenetwork/sourcehub/app/params"
)

// TxFeeChecker validates provided fee and returns the effective fee and tx priority.
type TxFeeChecker func(ctx sdk.Context, tx sdk.Tx) (sdk.Coins, int64, error)

// CustomDeductFeeDecorator deducts fees from the fee payer.
type CustomDeductFeeDecorator struct {
	accountKeeper  ante.AccountKeeper
	bankKeeper     types.BankKeeper
	feegrantKeeper ante.FeegrantKeeper
	txFeeChecker   TxFeeChecker
}

// NewCustomDeductFeeDecorator initializes custom deduct fee decorator with a fee checker.
func NewCustomDeductFeeDecorator(
	ak ante.AccountKeeper,
	bk types.BankKeeper,
	fk ante.FeegrantKeeper,
	tfc TxFeeChecker,
) CustomDeductFeeDecorator {

	if tfc == nil {
		tfc = checkTxFeeWithMinGasPrices
	}

	return CustomDeductFeeDecorator{
		accountKeeper:  ak,
		bankKeeper:     bk,
		feegrantKeeper: fk,
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
// Enforces the DefaultMinGasPrice to prefent spam if minimum gas price was set to 0 by the validator.
// NOTE: Always returns 0 for transaction priority because we handle TxPriority in priority_lane.go.
func checkTxFeeWithMinGasPrices(ctx sdk.Context, tx sdk.Tx) (sdk.Coins, int64, error) {
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return nil, 0, errorsmod.Wrap(sdkerrors.ErrTxDecode, "tx must be a FeeTx")
	}

	fees := feeTx.GetFee()
	gas := feeTx.GetGas()

	// Ensure exactly one fee denom is provided
	if fees.Len() == 0 {
		return nil, 0, errorsmod.Wrap(sdkerrors.ErrInvalidCoins, "transaction must include a fee")
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
	deductFeesFrom := feePayer

	// If fee granter is used, deduct from feeGranterAddr
	if feeGranter != nil {
		feeGranterAddr := sdk.AccAddress(feeGranter)

		if cdfd.feegrantKeeper == nil {
			return sdkerrors.ErrInvalidRequest.Wrap("fee grants are not enabled")
		} else if !bytes.Equal(feeGranterAddr, feePayer) {
			err := cdfd.feegrantKeeper.UseGrantedFees(ctx, feeGranterAddr, feePayer, fees, sdkTx.GetMsgs())
			if err != nil {
				return errorsmod.Wrapf(err, "%s does not allow to pay fees for %s", feeGranter, feePayer)
			}
		}

		deductFeesFrom = feeGranterAddr
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

// deductFees deducts fees from the given account.
func deductFees(bankKeeper types.BankKeeper, ctx sdk.Context, acc sdk.AccountI, fees sdk.Coins) error {
	if !fees.IsValid() {
		return errorsmod.Wrapf(sdkerrors.ErrInsufficientFee, "invalid fee amount: %s", fees)
	}

	err := bankKeeper.SendCoinsFromAccountToModule(ctx, acc.GetAddress(), types.FeeCollectorName, fees)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInsufficientFunds, err.Error())
	}

	return nil
}
