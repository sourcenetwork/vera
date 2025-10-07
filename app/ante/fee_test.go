package ante

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"cosmossdk.io/math"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	appparams "github.com/sourcenetwork/sourcehub/app/params"
	test "github.com/sourcenetwork/sourcehub/testutil"
)

// createCustomDecoratorWithMockDIDKeeper creates a custom decorator with DID-based mock keeper
func createCustomDecoratorWithMockDIDKeeper(t *testing.T, s *AnteTestSuite) (*CustomDeductFeeDecorator, *test.MockDIDFeegrantKeeper) {
	ctrl := gomock.NewController(t)
	mockKeeper := test.NewMockDIDFeegrantKeeper(ctrl)
	decorator := NewCustomDeductFeeDecorator(s.accountKeeper, s.bankKeeper, mockKeeper, nil, s.authStoreKey)
	return &decorator, mockKeeper
}

func TestCustomDeductFeeDecorator_CheckTx_ZeroGas(t *testing.T) {
	s := SetupTestSuite(t, true)
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()

	customDecorator := NewCustomDeductFeeDecorator(s.accountKeeper, s.bankKeeper, s.feeGrantKeeper, nil, s.authStoreKey)
	antehandler := sdk.ChainAnteDecorators(customDecorator)

	accs := s.CreateTestAccounts(1)

	msg := testdata.NewTestMsg(accs[0].acc.GetAddress())
	require.NoError(t, s.txBuilder.SetMsgs(msg))

	// set zero gas
	s.txBuilder.SetGasLimit(0)

	privs, accNums, accSeqs := []cryptotypes.PrivKey{accs[0].priv}, []uint64{0}, []uint64{0}
	tx, err := s.CreateTestTx(s.ctx, privs, accNums, accSeqs, s.ctx.ChainID(), signing.SignMode_SIGN_MODE_DIRECT)
	require.NoError(t, err)

	// set IsCheckTx to true
	s.ctx = s.ctx.WithIsCheckTx(true)

	_, err = antehandler(s.ctx, tx, false)
	require.Error(t, err)

	// zero gas is accepted in simulation mode
	_, err = antehandler(s.ctx, tx, true)
	require.NoError(t, err)
}

func TestCustomDeductFeeDecorator_CheckTx_InsufficientFee(t *testing.T) {
	s := SetupTestSuite(t, true)
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()

	customDecorator := NewCustomDeductFeeDecorator(s.accountKeeper, s.bankKeeper, s.feeGrantKeeper, nil, s.authStoreKey)
	antehandler := sdk.ChainAnteDecorators(customDecorator)

	accs := s.CreateTestAccounts(1)

	msg := testdata.NewTestMsg(accs[0].acc.GetAddress())
	require.NoError(t, s.txBuilder.SetMsgs(msg))

	gasLimit := uint64(20000)
	s.txBuilder.SetGasLimit(gasLimit)

	// set insufficient fee
	insufficientFee := sdk.NewCoins(sdk.NewInt64Coin(appparams.MicroOpenDenom, 1))
	s.txBuilder.SetFeeAmount(insufficientFee)

	privs, accNums, accSeqs := []cryptotypes.PrivKey{accs[0].priv}, []uint64{0}, []uint64{0}
	tx, err := s.CreateTestTx(s.ctx, privs, accNums, accSeqs, s.ctx.ChainID(), signing.SignMode_SIGN_MODE_DIRECT)
	require.NoError(t, err)

	// set IsCheckTx to true
	s.ctx = s.ctx.WithIsCheckTx(true)

	_, err = antehandler(s.ctx, tx, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "insufficient fee")
}

func TestCustomDeductFeeDecorator_CheckTx_ValidFee(t *testing.T) {
	s := SetupTestSuite(t, true)
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()

	customDecorator := NewCustomDeductFeeDecorator(s.accountKeeper, s.bankKeeper, s.feeGrantKeeper, nil, s.authStoreKey)
	antehandler := sdk.ChainAnteDecorators(customDecorator)

	accs := s.CreateTestAccounts(1)

	msg := testdata.NewTestMsg(accs[0].acc.GetAddress())
	require.NoError(t, s.txBuilder.SetMsgs(msg))

	gasLimit := uint64(20000)
	s.txBuilder.SetGasLimit(gasLimit)

	// set valid fee
	validFee := sdk.NewCoins(sdk.NewInt64Coin(appparams.MicroOpenDenom, 100))
	s.txBuilder.SetFeeAmount(validFee)

	s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(
		gomock.Any(),
		accs[0].acc.GetAddress(),
		authtypes.FeeCollectorName,
		validFee,
	).Return(nil).Times(1)

	privs, accNums, accSeqs := []cryptotypes.PrivKey{accs[0].priv}, []uint64{0}, []uint64{0}
	tx, err := s.CreateTestTx(s.ctx, privs, accNums, accSeqs, s.ctx.ChainID(), signing.SignMode_SIGN_MODE_DIRECT)
	require.NoError(t, err)

	// set IsCheckTx to true
	s.ctx = s.ctx.WithIsCheckTx(true)

	_, err = antehandler(s.ctx, tx, false)
	require.NoError(t, err)
}

func TestCustomDeductFeeDecorator_DeliverTx_FeeGranter(t *testing.T) {
	s := SetupTestSuite(t, true)
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()

	customDecorator := NewCustomDeductFeeDecorator(s.accountKeeper, s.bankKeeper, s.feeGrantKeeper, nil, s.authStoreKey)
	antehandler := sdk.ChainAnteDecorators(customDecorator)

	accs := s.CreateTestAccounts(2)
	feePayer := accs[0].acc.GetAddress()
	feeGranter := accs[1].acc.GetAddress()

	msg := testdata.NewTestMsg(feePayer)
	require.NoError(t, s.txBuilder.SetMsgs(msg))

	gasLimit := uint64(20000)
	s.txBuilder.SetGasLimit(gasLimit)

	// set valid fee and fee granter
	validFee := sdk.NewCoins(sdk.NewInt64Coin(appparams.MicroOpenDenom, 100))
	s.txBuilder.SetFeeGranter(feeGranter)
	s.txBuilder.SetFeeAmount(validFee)

	s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(
		gomock.Any(),
		feeGranter,
		authtypes.FeeCollectorName,
		validFee,
	).Return(nil).Times(1)

	privs, accNums, accSeqs := []cryptotypes.PrivKey{accs[0].priv}, []uint64{0}, []uint64{0}
	tx, err := s.CreateTestTx(s.ctx, privs, accNums, accSeqs, s.ctx.ChainID(), signing.SignMode_SIGN_MODE_DIRECT)
	require.NoError(t, err)

	s.feeGrantKeeper.EXPECT().UseGrantedFees(gomock.Any(), feeGranter, feePayer, gomock.Any(), gomock.Any()).Return(nil)

	_, err = antehandler(s.ctx, tx, false)
	require.NoError(t, err)
}

func TestCustomDeductFeeDecorator_DeliverTx(t *testing.T) {
	s := SetupTestSuite(t, false)
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()

	cdfd := NewCustomDeductFeeDecorator(s.accountKeeper, s.bankKeeper, nil, nil, s.authStoreKey)
	antehandler := sdk.ChainAnteDecorators(cdfd)

	accs := s.CreateTestAccounts(1)

	msg := testdata.NewTestMsg(accs[0].acc.GetAddress())
	require.NoError(t, s.txBuilder.SetMsgs(msg))

	gasLimit := testdata.NewTestGasLimit()
	s.txBuilder.SetGasLimit(gasLimit)

	// set valid fee
	validFee := sdk.NewCoins(sdk.NewInt64Coin(appparams.MicroOpenDenom, 100))
	s.txBuilder.SetFeeAmount(validFee)

	privs, accNums, accSeqs := []cryptotypes.PrivKey{accs[0].priv}, []uint64{0}, []uint64{0}
	tx, err := s.CreateTestTx(s.ctx, privs, accNums, accSeqs, s.ctx.ChainID(), signing.SignMode_SIGN_MODE_DIRECT)
	require.NoError(t, err)

	s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(
		gomock.Any(),
		accs[0].acc.GetAddress(),
		authtypes.FeeCollectorName,
		validFee,
	).Return(sdkerrors.ErrInsufficientFunds).Times(1)

	_, err = antehandler(s.ctx, tx, false)
	require.Error(t, err, "Tx did not error when fee payer had insufficient funds")

	s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(
		gomock.Any(),
		accs[0].acc.GetAddress(),
		authtypes.FeeCollectorName,
		validFee,
	).Return(nil).Times(1)

	_, err = antehandler(s.ctx, tx, false)
	require.NoError(t, err, "Tx errored after account has been set with sufficient funds")
}

func TestCustomDeductFeeDecorator_OpenDenomFees(t *testing.T) {
	s := SetupTestSuite(t, true)
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()

	cdfd := NewCustomDeductFeeDecorator(s.accountKeeper, s.bankKeeper, s.feeGrantKeeper, nil, s.authStoreKey)
	antehandler := sdk.ChainAnteDecorators(cdfd)

	accs := s.CreateTestAccounts(1)

	msg := testdata.NewTestMsg(accs[0].acc.GetAddress())
	require.NoError(t, s.txBuilder.SetMsgs(msg))

	gasLimit := uint64(10)
	s.txBuilder.SetGasLimit(gasLimit)

	// set valid uopen fee
	validFee := sdk.NewCoins(sdk.NewInt64Coin(appparams.MicroOpenDenom, 100))
	s.txBuilder.SetFeeAmount(validFee)

	s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(
		gomock.Any(),
		accs[0].acc.GetAddress(),
		authtypes.FeeCollectorName,
		validFee,
	).Return(nil).Times(3)

	privs, accNums, accSeqs := []cryptotypes.PrivKey{accs[0].priv}, []uint64{0}, []uint64{0}
	tx, err := s.CreateTestTx(s.ctx, privs, accNums, accSeqs, s.ctx.ChainID(), signing.SignMode_SIGN_MODE_DIRECT)
	require.NoError(t, err)

	// set 20 uopen as gas price so tx with 100 fee fails (200 required)
	uopenPrice := sdk.NewDecCoinFromDec(appparams.MicroOpenDenom, math.LegacyNewDec(20))
	highGasPrice := []sdk.DecCoin{uopenPrice}
	s.ctx = s.ctx.WithMinGasPrices(highGasPrice)

	// set IsCheckTx to true
	s.ctx = s.ctx.WithIsCheckTx(true)

	_, err = antehandler(s.ctx, tx, false)
	require.Error(t, err, "CustomDeductFeeDecorator should have errored on too low fee for local gasPrice")

	// antehandler should not error since we do not check minGasPrice in simulation mode
	cacheCtx, _ := s.ctx.CacheContext()
	_, err = antehandler(cacheCtx, tx, true)
	require.NoError(t, err, "CustomDeductFeeDecorator should not have errored in simulation mode")

	// set IsCheckTx to false
	s.ctx = s.ctx.WithIsCheckTx(false)

	// antehandler should not error since we do not check minGasPrice in DeliverTx
	_, err = antehandler(s.ctx, tx, false)
	require.NoError(t, err, "CustomDeductFeeDecorator returned error in DeliverTx")

	// set IsCheckTx back to true for testing sufficient mempool fee
	s.ctx = s.ctx.WithIsCheckTx(true)

	// set 1 uopen as gas price so tx with 100 fee succeeds
	uopenPrice = sdk.NewDecCoinFromDec(appparams.MicroOpenDenom, math.LegacyOneDec())
	lowGasPrice := []sdk.DecCoin{uopenPrice}
	s.ctx = s.ctx.WithMinGasPrices(lowGasPrice)

	newCtx, err := antehandler(s.ctx, tx, false)
	require.NoError(t, err, "CustomDeductFeeDecorator should not have errored on fee higher than local gasPrice")
	require.Equal(t, int64(0), newCtx.Priority())
}

func TestCustomDeductFeeDecorator_CreditDenomFees(t *testing.T) {
	s := SetupTestSuite(t, true)
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()

	cdfd := NewCustomDeductFeeDecorator(s.accountKeeper, s.bankKeeper, s.feeGrantKeeper, nil, s.authStoreKey)
	antehandler := sdk.ChainAnteDecorators(cdfd)

	accs := s.CreateTestAccounts(1)

	msg := testdata.NewTestMsg(accs[0].acc.GetAddress())
	require.NoError(t, s.txBuilder.SetMsgs(msg))

	gasLimit := uint64(10)
	s.txBuilder.SetGasLimit(gasLimit)

	// set valid fee in ucredit
	validFee := sdk.NewCoins(sdk.NewInt64Coin(appparams.MicroCreditDenom, 100))
	s.txBuilder.SetFeeAmount(validFee)

	s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(
		gomock.Any(),
		accs[0].acc.GetAddress(),
		authtypes.FeeCollectorName,
		validFee,
	).Return(nil).Times(3)

	privs, accNums, accSeqs := []cryptotypes.PrivKey{accs[0].priv}, []uint64{0}, []uint64{0}
	tx, err := s.CreateTestTx(s.ctx, privs, accNums, accSeqs, s.ctx.ChainID(), signing.SignMode_SIGN_MODE_DIRECT)
	require.NoError(t, err)

	// set 10 ucredit as gas price so tx with 100 fee fails (10 * 10 * 10 required due to CreditFeeMultiplier)
	uopenPrice := sdk.NewDecCoinFromDec(appparams.MicroCreditDenom, math.LegacyNewDec(10))
	highGasPrice := []sdk.DecCoin{uopenPrice}
	s.ctx = s.ctx.WithMinGasPrices(highGasPrice)

	// set IsCheckTx to true
	s.ctx = s.ctx.WithIsCheckTx(true)

	_, err = antehandler(s.ctx, tx, false)
	require.Error(t, err, "CustomDeductFeeDecorator should have errored on too low fee for local gasPrice")

	// antehandler should not error since we do not check minGasPrice in simulation mode
	cacheCtx, _ := s.ctx.CacheContext()
	_, err = antehandler(cacheCtx, tx, true)
	require.NoError(t, err, "CustomDeductFeeDecorator should not have errored in simulation mode")

	// set IsCheckTx to false
	s.ctx = s.ctx.WithIsCheckTx(false)

	// antehandler should not error since we do not check minGasPrice in DeliverTx
	_, err = antehandler(s.ctx, tx, false)
	require.NoError(t, err, "CustomDeductFeeDecorator returned error in DeliverTx")

	// set IsCheckTx back to true for testing sufficient mempool fee
	s.ctx = s.ctx.WithIsCheckTx(true)

	// set 1 ucredit as gas price so tx with 100 fee succeeds (1 * 10 * 10 required)
	uopenPrice = sdk.NewDecCoinFromDec(appparams.MicroCreditDenom, math.LegacyOneDec())
	lowGasPrice := []sdk.DecCoin{uopenPrice}
	s.ctx = s.ctx.WithMinGasPrices(lowGasPrice)

	newCtx, err := antehandler(s.ctx, tx, false)
	require.NoError(t, err, "CustomDeductFeeDecorator should not have errored on fee higher than local gasPrice")
	require.Equal(t, int64(0), newCtx.Priority())
}

func TestCustomDeductFeeDecorator_HandleFeegrant_WithDID(t *testing.T) {
	s := SetupTestSuite(t, true)

	// Create decorator with mock DID keeper
	customDecorator, mockFeegrantKeeper := createCustomDecoratorWithMockDIDKeeper(t, s)

	accs := s.CreateTestAccounts(2)
	granter := accs[0].acc.GetAddress()
	grantee := accs[1].acc.GetAddress()
	fees := sdk.NewCoins(sdk.NewInt64Coin(appparams.MicroOpenDenom, 100))
	msgs := []sdk.Msg{testdata.NewTestMsg(grantee)}
	testDID := "did:key:z6MkpTHR8VNsBxYAAWHut2Geadd9jSwuBV8xRoAnwWsdvktH"

	// Set DID in context to simulate extraction from JWS extension
	// Use the helper function to ensure proper context key usage
	ctx := s.ctx.WithValue(ExtractedDIDContextKey, testDID)

	// Verify DID was properly set in context
	extractedDID := getExtractedDIDFromContext(ctx)
	require.Equal(t, testDID, extractedDID, "DID should be properly set in context")

	// Expect DID-based feegrant to be called
	mockFeegrantKeeper.EXPECT().UseGrantedFeesByDID(ctx, granter, testDID, fees, msgs).Return(nil)

	err := customDecorator.handleFeegrant(ctx, granter, grantee, fees, msgs)
	require.NoError(t, err, "handleFeegrant should succeed with DID-based feegrant")
}

func TestCustomDeductFeeDecorator_HandleFeegrant_WithoutDID(t *testing.T) {
	s := SetupTestSuite(t, true)

	customDecorator := NewCustomDeductFeeDecorator(s.accountKeeper, s.bankKeeper, s.feeGrantKeeper, nil, s.authStoreKey)

	accs := s.CreateTestAccounts(2)
	granter := accs[0].acc.GetAddress()
	grantee := accs[1].acc.GetAddress()
	fees := sdk.NewCoins(sdk.NewInt64Coin(appparams.MicroOpenDenom, 100))
	msgs := []sdk.Msg{testdata.NewTestMsg(grantee)}

	// No DID in context - should use standard feegrant
	ctx := s.ctx

	// Expect standard feegrant to be called
	s.feeGrantKeeper.EXPECT().UseGrantedFees(ctx, granter, grantee, fees, msgs).Return(nil)

	err := customDecorator.handleFeegrant(ctx, granter, grantee, fees, msgs)
	require.NoError(t, err, "handleFeegrant should succeed with standard feegrant")
}

func TestCustomDeductFeeDecorator_HandleFeegrant_DIDBasedError(t *testing.T) {
	s := SetupTestSuite(t, true)

	// Create decorator with mock DID keeper
	customDecorator, mockFeegrantKeeper := createCustomDecoratorWithMockDIDKeeper(t, s)

	accs := s.CreateTestAccounts(2)
	granter := accs[0].acc.GetAddress()
	grantee := accs[1].acc.GetAddress()
	fees := sdk.NewCoins(sdk.NewInt64Coin(appparams.MicroOpenDenom, 100))
	msgs := []sdk.Msg{testdata.NewTestMsg(grantee)}
	testDID := "did:key:z6MkpTHR8VNsBxYAAWHut2Geadd9jSwuBV8xRoAnwWsdvktH"

	// Set DID in context
	ctx := s.ctx.WithValue(contextKey("extracted_did"), testDID)

	// Expect DID-based feegrant to return error
	expectedErr := sdkerrors.ErrUnauthorized.Wrap("insufficient allowance")
	mockFeegrantKeeper.EXPECT().UseGrantedFeesByDID(ctx, granter, testDID, fees, msgs).Return(expectedErr)

	err := customDecorator.handleFeegrant(ctx, granter, grantee, fees, msgs)
	require.Error(t, err, "handleFeegrant should return error from DID-based feegrant")
	require.Equal(t, expectedErr, err)
}

func TestCustomDeductFeeDecorator_HandleFeegrant_StandardError(t *testing.T) {
	s := SetupTestSuite(t, true)

	customDecorator := NewCustomDeductFeeDecorator(s.accountKeeper, s.bankKeeper, s.feeGrantKeeper, nil, s.authStoreKey)

	accs := s.CreateTestAccounts(2)
	granter := accs[0].acc.GetAddress()
	grantee := accs[1].acc.GetAddress()
	fees := sdk.NewCoins(sdk.NewInt64Coin(appparams.MicroOpenDenom, 100))
	msgs := []sdk.Msg{testdata.NewTestMsg(grantee)}

	// No DID in context
	ctx := s.ctx

	// Expect standard feegrant to return error
	expectedErr := sdkerrors.ErrUnauthorized.Wrap("fee allowance not found")
	s.feeGrantKeeper.EXPECT().UseGrantedFees(ctx, granter, grantee, fees, msgs).Return(expectedErr)

	err := customDecorator.handleFeegrant(ctx, granter, grantee, fees, msgs)
	require.Error(t, err, "handleFeegrant should return error from standard feegrant")
	require.Equal(t, expectedErr, err)
}
