package ante

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/client"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	antetypes "github.com/sourcenetwork/sourcehub/app/ante/types"
	appparams "github.com/sourcenetwork/sourcehub/app/params"
	test "github.com/sourcenetwork/sourcehub/testutil"
)

func TestExtensionOptionsDecorator_ValidJWSExtension(t *testing.T) {
	s := SetupTestSuite(t, true)
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()

	decorator := NewExtensionOptionsDecorator()
	antehandler := sdk.ChainAnteDecorators(decorator)

	accs := s.CreateTestAccounts(1)

	msg := testdata.NewTestMsg(accs[0].acc.GetAddress())
	require.NoError(t, s.txBuilder.SetMsgs(msg))

	// Create a valid JWS extension option with properly signed JWS using the test account address
	authorizedAccount := accs[0].acc.GetAddress().String()
	bearerToken, userDID := test.GenerateSignedJWSWithMatchingDID(t, authorizedAccount)
	jwsOpt := &antetypes.JWSExtensionOption{
		BearerToken: bearerToken,
	}

	// Pack the extension option
	any, err := codectypes.NewAnyWithValue(jwsOpt)
	require.NoError(t, err)

	// Add extension option to the transaction
	if extBuilder, ok := s.txBuilder.(client.ExtendedTxBuilder); ok {
		extBuilder.SetExtensionOptions(any)
	} else {
		t.Fatal("TxBuilder does not implement ExtendedTxBuilder")
	}

	privs, accNums, accSeqs := []cryptotypes.PrivKey{accs[0].priv}, []uint64{0}, []uint64{0}
	tx, err := s.CreateTestTx(s.ctx, privs, accNums, accSeqs, s.ctx.ChainID(), signing.SignMode_SIGN_MODE_DIRECT)
	require.NoError(t, err)

	newCtx, err := antehandler(s.ctx, tx, false)
	require.NoError(t, err, "ExtensionOptionsDecorator should accept valid JWS extension option")

	// Verify DID was extracted and stored in context
	extractedDID := GetExtractedDIDFromContext(newCtx)
	require.Equal(t, userDID, extractedDID, "DID should be extracted and stored in context")
}

func TestExtensionOptionsDecorator_InvalidJWSExtension(t *testing.T) {
	s := SetupTestSuite(t, true)
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()

	decorator := NewExtensionOptionsDecorator()
	antehandler := sdk.ChainAnteDecorators(decorator)

	accs := s.CreateTestAccounts(1)

	msg := testdata.NewTestMsg(accs[0].acc.GetAddress())
	require.NoError(t, s.txBuilder.SetMsgs(msg))

	// Create an invalid JWS extension option
	jwsOpt := &antetypes.JWSExtensionOption{
		BearerToken: "",
	}

	// Pack the extension option
	any, err := codectypes.NewAnyWithValue(jwsOpt)
	require.NoError(t, err)

	// Add extension option to the transaction
	if extBuilder, ok := s.txBuilder.(client.ExtendedTxBuilder); ok {
		extBuilder.SetExtensionOptions(any)
	} else {
		t.Fatal("TxBuilder does not implement ExtendedTxBuilder")
	}

	privs, accNums, accSeqs := []cryptotypes.PrivKey{accs[0].priv}, []uint64{0}, []uint64{0}
	tx, err := s.CreateTestTx(s.ctx, privs, accNums, accSeqs, s.ctx.ChainID(), signing.SignMode_SIGN_MODE_DIRECT)
	require.NoError(t, err)

	_, err = antehandler(s.ctx, tx, false)
	require.Error(t, err, "ExtensionOptionsDecorator should reject invalid JWS extension option")
	require.Contains(t, err.Error(), "failed parsing jws")
}

func TestExtensionOptionsDecorator_InvalidJWSFormat(t *testing.T) {
	s := SetupTestSuite(t, true)
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()

	decorator := NewExtensionOptionsDecorator()
	antehandler := sdk.ChainAnteDecorators(decorator)

	accs := s.CreateTestAccounts(1)

	msg := testdata.NewTestMsg(accs[0].acc.GetAddress())
	require.NoError(t, s.txBuilder.SetMsgs(msg))

	// Create an invalid JWS extension option
	jwsOpt := &antetypes.JWSExtensionOption{
		BearerToken: "invalid.format",
	}

	// Pack the extension option
	any, err := codectypes.NewAnyWithValue(jwsOpt)
	require.NoError(t, err)

	// Add extension option to the transaction
	if extBuilder, ok := s.txBuilder.(client.ExtendedTxBuilder); ok {
		extBuilder.SetExtensionOptions(any)
	} else {
		t.Fatal("TxBuilder does not implement ExtendedTxBuilder")
	}

	privs, accNums, accSeqs := []cryptotypes.PrivKey{accs[0].priv}, []uint64{0}, []uint64{0}
	tx, err := s.CreateTestTx(s.ctx, privs, accNums, accSeqs, s.ctx.ChainID(), signing.SignMode_SIGN_MODE_DIRECT)
	require.NoError(t, err)

	_, err = antehandler(s.ctx, tx, false)
	require.Error(t, err, "ExtensionOptionsDecorator should reject JWS with invalid format")
	require.Contains(t, err.Error(), "failed parsing jws")
}

func TestExtensionOptionsDecorator_InvalidJWSSignature(t *testing.T) {
	s := SetupTestSuite(t, true)
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()

	decorator := NewExtensionOptionsDecorator()
	antehandler := sdk.ChainAnteDecorators(decorator)

	accs := s.CreateTestAccounts(1)

	msg := testdata.NewTestMsg(accs[0].acc.GetAddress())
	require.NoError(t, s.txBuilder.SetMsgs(msg))

	// Create an invalid JWS extension option
	jwsOpt := &antetypes.JWSExtensionOption{
		BearerToken: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
	}

	// Pack the extension option
	any, err := codectypes.NewAnyWithValue(jwsOpt)
	require.NoError(t, err)

	// Add extension option to the transaction
	if extBuilder, ok := s.txBuilder.(client.ExtendedTxBuilder); ok {
		extBuilder.SetExtensionOptions(any)
	} else {
		t.Fatal("TxBuilder does not implement ExtendedTxBuilder")
	}

	privs, accNums, accSeqs := []cryptotypes.PrivKey{accs[0].priv}, []uint64{0}, []uint64{0}
	tx, err := s.CreateTestTx(s.ctx, privs, accNums, accSeqs, s.ctx.ChainID(), signing.SignMode_SIGN_MODE_DIRECT)
	require.NoError(t, err)

	_, err = antehandler(s.ctx, tx, false)
	require.Error(t, err, "ExtensionOptionsDecorator should reject invalid JWS signature")
	require.Contains(t, err.Error(), "missing required claim")
}

func TestExtensionOptionsDecorator_UnknownExtensionOption(t *testing.T) {
	s := SetupTestSuite(t, true)
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()

	decorator := NewExtensionOptionsDecorator()
	antehandler := sdk.ChainAnteDecorators(decorator)

	accs := s.CreateTestAccounts(1)

	msg := testdata.NewTestMsg(accs[0].acc.GetAddress())
	require.NoError(t, s.txBuilder.SetMsgs(msg))

	// Create an unknown extension option
	unknownOpt := testdata.NewTestMsg(accs[0].acc.GetAddress())

	// Pack the extension option
	any, err := codectypes.NewAnyWithValue(unknownOpt)
	require.NoError(t, err)

	// Add extension option to the transaction
	if extBuilder, ok := s.txBuilder.(client.ExtendedTxBuilder); ok {
		extBuilder.SetExtensionOptions(any)
	} else {
		t.Fatal("TxBuilder does not implement ExtendedTxBuilder")
	}

	privs, accNums, accSeqs := []cryptotypes.PrivKey{accs[0].priv}, []uint64{0}, []uint64{0}
	tx, err := s.CreateTestTx(s.ctx, privs, accNums, accSeqs, s.ctx.ChainID(), signing.SignMode_SIGN_MODE_DIRECT)
	require.NoError(t, err)

	_, err = antehandler(s.ctx, tx, false)
	require.Error(t, err, "ExtensionOptionsDecorator should reject unknown extension option")
	require.Contains(t, err.Error(), "unsupported extension option type")
}

func TestExtensionOptionsDecorator_NoExtensionOptions(t *testing.T) {
	s := SetupTestSuite(t, true)
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()

	decorator := NewExtensionOptionsDecorator()
	antehandler := sdk.ChainAnteDecorators(decorator)

	accs := s.CreateTestAccounts(1)

	msg := testdata.NewTestMsg(accs[0].acc.GetAddress())
	require.NoError(t, s.txBuilder.SetMsgs(msg))

	// No extension options set

	privs, accNums, accSeqs := []cryptotypes.PrivKey{accs[0].priv}, []uint64{0}, []uint64{0}
	tx, err := s.CreateTestTx(s.ctx, privs, accNums, accSeqs, s.ctx.ChainID(), signing.SignMode_SIGN_MODE_DIRECT)
	require.NoError(t, err)

	newCtx, err := antehandler(s.ctx, tx, false)
	require.NoError(t, err, "ExtensionOptionsDecorator should allow transactions with no extension options")

	// Verify no DID was extracted
	extractedDID := GetExtractedDIDFromContext(newCtx)
	require.Empty(t, extractedDID, "No DID should be extracted when no extension options are present")
}

func TestExtensionOptionsDecorator_JWSOptionWithInvalidSignature(t *testing.T) {
	s := SetupTestSuite(t, true)
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()

	decorator := NewExtensionOptionsDecorator()
	antehandler := sdk.ChainAnteDecorators(decorator)

	accs := s.CreateTestAccounts(1)

	msg := testdata.NewTestMsg(accs[0].acc.GetAddress())
	require.NoError(t, s.txBuilder.SetMsgs(msg))

	// Create a JWS extension option without DID in payload
	jwsOpt := &antetypes.JWSExtensionOption{
		BearerToken: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
	}

	// Pack the extension option
	any, err := codectypes.NewAnyWithValue(jwsOpt)
	require.NoError(t, err)

	// Add extension option to the transaction
	if extBuilder, ok := s.txBuilder.(client.ExtendedTxBuilder); ok {
		extBuilder.SetExtensionOptions(any)
	} else {
		t.Fatal("TxBuilder does not implement ExtendedTxBuilder")
	}

	privs, accNums, accSeqs := []cryptotypes.PrivKey{accs[0].priv}, []uint64{0}, []uint64{0}
	tx, err := s.CreateTestTx(s.ctx, privs, accNums, accSeqs, s.ctx.ChainID(), signing.SignMode_SIGN_MODE_DIRECT)
	require.NoError(t, err)

	_, err = antehandler(s.ctx, tx, false)
	require.Error(t, err, "ExtensionOptionsDecorator should reject JWS without valid DID in payload")
	require.Contains(t, err.Error(), "missing required claim")
}

func TestExtensionOptionsDecorator_ExtractDIDFromJWSPayload(t *testing.T) {
	s := SetupTestSuite(t, true)
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()

	decorator := NewExtensionOptionsDecorator()
	antehandler := sdk.ChainAnteDecorators(decorator)

	accs := s.CreateTestAccounts(1)

	msg := testdata.NewTestMsg(accs[0].acc.GetAddress())
	require.NoError(t, s.txBuilder.SetMsgs(msg))

	// Create a JWS with DID in payload using the test account address
	authorizedAccount := accs[0].acc.GetAddress().String()
	bearerToken, userDID := test.GenerateSignedJWSWithMatchingDID(t, authorizedAccount)
	jwsOpt := &antetypes.JWSExtensionOption{
		BearerToken: bearerToken,
	}

	// Pack the extension option
	any, err := codectypes.NewAnyWithValue(jwsOpt)
	require.NoError(t, err)

	// Add extension option to the transaction
	if extBuilder, ok := s.txBuilder.(client.ExtendedTxBuilder); ok {
		extBuilder.SetExtensionOptions(any)
	} else {
		t.Fatal("TxBuilder does not implement ExtendedTxBuilder")
	}

	privs, accNums, accSeqs := []cryptotypes.PrivKey{accs[0].priv}, []uint64{0}, []uint64{0}
	tx, err := s.CreateTestTx(s.ctx, privs, accNums, accSeqs, s.ctx.ChainID(), signing.SignMode_SIGN_MODE_DIRECT)
	require.NoError(t, err)

	newCtx, err := antehandler(s.ctx, tx, false)
	require.NoError(t, err, "ExtensionOptionsDecorator should accept JWS extension option with DID in payload")

	// Verify DID was extracted from payload
	extractedDID := GetExtractedDIDFromContext(newCtx)
	require.Equal(t, userDID, extractedDID, "DID should be extracted from JWS payload")
}

func TestExtensionOptionsDecorator_ValidJWSWithDIDInPayload(t *testing.T) {
	s := SetupTestSuite(t, true)
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()

	decorator := NewExtensionOptionsDecorator()
	antehandler := sdk.ChainAnteDecorators(decorator)

	accs := s.CreateTestAccounts(1)

	msg := testdata.NewTestMsg(accs[0].acc.GetAddress())
	require.NoError(t, s.txBuilder.SetMsgs(msg))

	// Create a JWS with DID in payload using the test account address
	authorizedAccount := accs[0].acc.GetAddress().String()
	bearerToken, userDID := test.GenerateSignedJWSWithMatchingDID(t, authorizedAccount)
	jwsOpt := &antetypes.JWSExtensionOption{
		BearerToken: bearerToken,
	}

	// Pack the extension option
	any, err := codectypes.NewAnyWithValue(jwsOpt)
	require.NoError(t, err)

	// Add extension option to the transaction
	if extBuilder, ok := s.txBuilder.(client.ExtendedTxBuilder); ok {
		extBuilder.SetExtensionOptions(any)
	} else {
		t.Fatal("TxBuilder does not implement ExtendedTxBuilder")
	}

	privs, accNums, accSeqs := []cryptotypes.PrivKey{accs[0].priv}, []uint64{0}, []uint64{0}
	tx, err := s.CreateTestTx(s.ctx, privs, accNums, accSeqs, s.ctx.ChainID(), signing.SignMode_SIGN_MODE_DIRECT)
	require.NoError(t, err)

	newCtx, err := antehandler(s.ctx, tx, false)
	require.NoError(t, err, "ExtensionOptionsDecorator should accept JWS extension option")

	// Verify that payload DID is extracted and used
	extractedDID := GetExtractedDIDFromContext(newCtx)
	require.Equal(t, userDID, extractedDID, "Payload DID should be extracted and used")
}

func TestExtensionOptionsDecorator_SecurityTamperedJWS(t *testing.T) {
	s := SetupTestSuite(t, true)
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()

	decorator := NewExtensionOptionsDecorator()
	antehandler := sdk.ChainAnteDecorators(decorator)

	accs := s.CreateTestAccounts(1)

	msg := testdata.NewTestMsg(accs[0].acc.GetAddress())
	require.NoError(t, s.txBuilder.SetMsgs(msg))

	// Create JWS with valid signature using the test account address
	authorizedAccount := accs[0].acc.GetAddress().String()
	bearerToken, _ := test.GenerateSignedJWSWithMatchingDID(t, authorizedAccount)
	// Tamper with the JWS signature to make it invalid
	tamperedSignature := bearerToken[:len(bearerToken)-10] + "tampered123"
	jwsOpt := &antetypes.JWSExtensionOption{
		BearerToken: tamperedSignature,
	}

	// Pack the extension option
	any, err := codectypes.NewAnyWithValue(jwsOpt)
	require.NoError(t, err)

	// Add extension option to the transaction
	if extBuilder, ok := s.txBuilder.(client.ExtendedTxBuilder); ok {
		extBuilder.SetExtensionOptions(any)
	} else {
		t.Fatal("TxBuilder does not implement ExtendedTxBuilder")
	}

	privs, accNums, accSeqs := []cryptotypes.PrivKey{accs[0].priv}, []uint64{0}, []uint64{0}
	tx, err := s.CreateTestTx(s.ctx, privs, accNums, accSeqs, s.ctx.ChainID(), signing.SignMode_SIGN_MODE_DIRECT)
	require.NoError(t, err)

	_, err = antehandler(s.ctx, tx, false)
	require.Error(t, err, "ExtensionOptionsDecorator should reject tampered JWS")
	require.Contains(t, err.Error(), "could not verify actor signature")
}

func TestExtensionOptionsDecorator_SecurityNoDIDInPayload(t *testing.T) {
	s := SetupTestSuite(t, true)
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()

	decorator := NewExtensionOptionsDecorator()
	antehandler := sdk.ChainAnteDecorators(decorator)

	accs := s.CreateTestAccounts(1)

	msg := testdata.NewTestMsg(accs[0].acc.GetAddress())
	require.NoError(t, s.txBuilder.SetMsgs(msg))

	// Create JWS with no DID in payload (e.g. {"sub":"1234567890","name":"John Doe","iat":1516239022})
	jwsOpt := &antetypes.JWSExtensionOption{
		BearerToken: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
	}

	// Pack the extension option
	any, err := codectypes.NewAnyWithValue(jwsOpt)
	require.NoError(t, err)

	// Add extension option to the transaction
	if extBuilder, ok := s.txBuilder.(client.ExtendedTxBuilder); ok {
		extBuilder.SetExtensionOptions(any)
	} else {
		t.Fatal("TxBuilder does not implement ExtendedTxBuilder")
	}

	privs, accNums, accSeqs := []cryptotypes.PrivKey{accs[0].priv}, []uint64{0}, []uint64{0}
	tx, err := s.CreateTestTx(s.ctx, privs, accNums, accSeqs, s.ctx.ChainID(), signing.SignMode_SIGN_MODE_DIRECT)
	require.NoError(t, err)

	_, err = antehandler(s.ctx, tx, false)
	require.Error(t, err, "ExtensionOptionsDecorator should reject JWS without DID in payload")
	require.Contains(t, err.Error(), "missing required claim")
}

func TestExtensionAndFeeDecorators_WithDID(t *testing.T) {
	s := SetupTestSuite(t, true)
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()

	// Create mock feegrant keeper with DID support
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockFeegrantKeeper := test.NewMockDIDFeegrantKeeper(ctrl)

	// Create both decorators
	extensionDecorator := NewExtensionOptionsDecorator()
	feeDecorator := NewCustomDeductFeeDecorator(s.accountKeeper, s.bankKeeper, mockFeegrantKeeper, nil, s.authStoreKey)

	// Chain them in the same order as actual ante handler
	antehandler := sdk.ChainAnteDecorators(extensionDecorator, feeDecorator)

	accs := s.CreateTestAccounts(2)
	feePayer := accs[0].acc.GetAddress()
	feeGranter := accs[1].acc.GetAddress()

	msg := testdata.NewTestMsg(feePayer)
	require.NoError(t, s.txBuilder.SetMsgs(msg))

	// Create valid JWS extension option using the fee payer address
	authorizedAccount := feePayer.String()
	bearerToken, userDID := test.GenerateSignedJWSWithMatchingDID(t, authorizedAccount)
	jwsOpt := &antetypes.JWSExtensionOption{
		BearerToken: bearerToken,
	}

	// Pack and add extension option
	any, err := codectypes.NewAnyWithValue(jwsOpt)
	require.NoError(t, err)

	if extBuilder, ok := s.txBuilder.(client.ExtendedTxBuilder); ok {
		extBuilder.SetExtensionOptions(any)
	} else {
		t.Fatal("TxBuilder does not implement ExtendedTxBuilder")
	}

	// Set fee granter and amount
	s.txBuilder.SetFeeGranter(feeGranter)
	validFee := sdk.NewCoins(sdk.NewInt64Coin(appparams.MicroOpenDenom, 500))
	s.txBuilder.SetFeeAmount(validFee)
	s.txBuilder.SetGasLimit(200000)

	// Mock expectations for DID-based feegrant
	mockFeegrantKeeper.EXPECT().UseGrantedFeesByDID(gomock.Any(), feeGranter, userDID, validFee, gomock.Any()).Return(nil)
	s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), feeGranter, authtypes.FeeCollectorName, validFee).Return(nil)

	// Create transaction
	privs, accNums, accSeqs := []cryptotypes.PrivKey{accs[0].priv}, []uint64{0}, []uint64{0}
	tx, err := s.CreateTestTx(s.ctx, privs, accNums, accSeqs, s.ctx.ChainID(), signing.SignMode_SIGN_MODE_DIRECT)
	require.NoError(t, err)

	// Run the ante handler chain
	newCtx, err := antehandler(s.ctx, tx, false)
	require.NoError(t, err, "Ante handler chain should succeed with DID-based feegrant")

	// Verify DID was passed through context
	extractedDID := GetExtractedDIDFromContext(newCtx)
	require.Equal(t, userDID, extractedDID, "DID should be preserved in context after ante chain")
}

func TestExtensionAndFeeDecorators_NoDID(t *testing.T) {
	s := SetupTestSuite(t, true)
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()

	// Create both decorators
	extensionDecorator := NewExtensionOptionsDecorator()
	feeDecorator := NewCustomDeductFeeDecorator(s.accountKeeper, s.bankKeeper, s.feeGrantKeeper, nil, s.authStoreKey)

	// Chain them in the same order as actual ante handler
	antehandler := sdk.ChainAnteDecorators(extensionDecorator, feeDecorator)

	accs := s.CreateTestAccounts(2)
	feePayer := accs[0].acc.GetAddress()
	feeGranter := accs[1].acc.GetAddress()

	msg := testdata.NewTestMsg(feePayer)
	require.NoError(t, s.txBuilder.SetMsgs(msg))

	// Set fee granter and amount
	s.txBuilder.SetFeeGranter(feeGranter)
	validFee := sdk.NewCoins(sdk.NewInt64Coin(appparams.MicroOpenDenom, 500))
	s.txBuilder.SetFeeAmount(validFee)
	s.txBuilder.SetGasLimit(200000)

	// Mock expectations for standard feegrant (no DID)
	s.feeGrantKeeper.EXPECT().UseGrantedFees(gomock.Any(), feeGranter, feePayer, validFee, gomock.Any()).Return(nil)
	s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), feeGranter, authtypes.FeeCollectorName, validFee).Return(nil)

	// Create transaction
	privs, accNums, accSeqs := []cryptotypes.PrivKey{accs[0].priv}, []uint64{0}, []uint64{0}
	tx, err := s.CreateTestTx(s.ctx, privs, accNums, accSeqs, s.ctx.ChainID(), signing.SignMode_SIGN_MODE_DIRECT)
	require.NoError(t, err)

	// Run the ante handler chain
	newCtx, err := antehandler(s.ctx, tx, false)
	require.NoError(t, err, "Ante handler chain should succeed with standard feegrant")

	// Verify no DID was extracted
	extractedDID := GetExtractedDIDFromContext(newCtx)
	require.Empty(t, extractedDID, "No DID should be in context when no extension options provided")
}

func TestExtensionOptionsDecorator_CorrectAuthorizedAccount(t *testing.T) {
	s := SetupTestSuite(t, true)
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()

	decorator := NewExtensionOptionsDecorator()
	antehandler := sdk.ChainAnteDecorators(decorator)

	accs := s.CreateTestAccounts(1)

	msg := testdata.NewTestMsg(accs[0].acc.GetAddress())
	require.NoError(t, s.txBuilder.SetMsgs(msg))

	// Create JWS token with authorized account matching the transaction signer
	authorizedAccount := accs[0].acc.GetAddress().String()
	bearerToken, userDID := test.GenerateSignedJWSWithMatchingDID(t, authorizedAccount)
	jwsOpt := &antetypes.JWSExtensionOption{
		BearerToken: bearerToken,
	}

	// Pack the extension option
	any, err := codectypes.NewAnyWithValue(jwsOpt)
	require.NoError(t, err)

	// Add extension option to the transaction
	if extBuilder, ok := s.txBuilder.(client.ExtendedTxBuilder); ok {
		extBuilder.SetExtensionOptions(any)
	} else {
		t.Fatal("TxBuilder does not implement ExtendedTxBuilder")
	}

	privs, accNums, accSeqs := []cryptotypes.PrivKey{accs[0].priv}, []uint64{0}, []uint64{0}
	tx, err := s.CreateTestTx(s.ctx, privs, accNums, accSeqs, s.ctx.ChainID(), signing.SignMode_SIGN_MODE_DIRECT)
	require.NoError(t, err)

	newCtx, err := antehandler(s.ctx, tx, false)
	require.NoError(t, err, "ExtensionOptionsDecorator should accept JWS with matching authorized account")

	// Verify DID was extracted and stored in context
	extractedDID := GetExtractedDIDFromContext(newCtx)
	require.Equal(t, userDID, extractedDID, "DID should be extracted and stored in context")
}

func TestExtensionOptionsDecorator_IncorrectAuthorizedAccount(t *testing.T) {
	s := SetupTestSuite(t, true)
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()

	decorator := NewExtensionOptionsDecorator()
	antehandler := sdk.ChainAnteDecorators(decorator)

	accs := s.CreateTestAccounts(1)

	msg := testdata.NewTestMsg(accs[0].acc.GetAddress())
	require.NoError(t, s.txBuilder.SetMsgs(msg))

	// Create JWS token with authorized account NOT matching the transaction signer
	wrongAuthorizedAccount := "source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et"
	bearerToken, _ := test.GenerateSignedJWSWithMatchingDID(t, wrongAuthorizedAccount)
	jwsOpt := &antetypes.JWSExtensionOption{
		BearerToken: bearerToken,
	}

	// Pack the extension option
	any, err := codectypes.NewAnyWithValue(jwsOpt)
	require.NoError(t, err)

	// Add extension option to the transaction
	if extBuilder, ok := s.txBuilder.(client.ExtendedTxBuilder); ok {
		extBuilder.SetExtensionOptions(any)
	} else {
		t.Fatal("TxBuilder does not implement ExtendedTxBuilder")
	}

	privs, accNums, accSeqs := []cryptotypes.PrivKey{accs[0].priv}, []uint64{0}, []uint64{0}
	tx, err := s.CreateTestTx(s.ctx, privs, accNums, accSeqs, s.ctx.ChainID(), signing.SignMode_SIGN_MODE_DIRECT)
	require.NoError(t, err)

	_, err = antehandler(s.ctx, tx, false)
	require.Error(t, err, "ExtensionOptionsDecorator should reject JWS with mismatched authorized account")
	require.Contains(t, err.Error(), "authorized account mismatch")
}
