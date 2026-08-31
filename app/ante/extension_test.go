package ante

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/client"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	antetypes "github.com/sourcenetwork/vera/app/ante/types"
	appparams "github.com/sourcenetwork/vera/app/params"
	test "github.com/sourcenetwork/vera/testutil"
	acptypes "github.com/sourcenetwork/vera/x/acp/types"
	coremoduletypes "github.com/sourcenetwork/vera/x/core/types"
)

type extensionCoreKeeper struct {
	chainConfig coremoduletypes.ChainConfig
	params      coremoduletypes.Params
	storeErr    error
}

func (k extensionCoreKeeper) GetChainConfig(context.Context) coremoduletypes.ChainConfig {
	return k.chainConfig
}

func (k extensionCoreKeeper) GetParams(context.Context) coremoduletypes.Params {
	return k.params
}

func (k extensionCoreKeeper) StoreOrUpdateJWSToken(
	context.Context,
	string,
	string,
	string,
	time.Time,
	time.Time,
) error {
	return k.storeErr
}

func buildExtensionTestTx(
	t *testing.T,
	s *AnteTestSuite,
	signer TestAccount,
	creator string,
	feeGranter sdk.AccAddress,
	authorizedAccount string,
	providerToken *antetypes.ProviderToken,
) (sdk.Tx, string) {
	t.Helper()
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()
	require.NoError(t, s.txBuilder.SetMsgs(&acptypes.MsgCreatePolicy{
		Creator:     creator,
		Policy:      "name: test policy",
		MarshalType: coretypes.PolicyMarshalingType_YAML,
	}))
	if feeGranter != nil {
		s.txBuilder.SetFeeGranter(feeGranter)
	}

	bearerToken, userDID := test.GenerateSignedJWSWithProvider(t, authorizedAccount, providerToken)
	jwsOpt, err := codectypes.NewAnyWithValue(&antetypes.JWSExtensionOption{BearerToken: bearerToken})
	require.NoError(t, err)
	extendedBuilder, ok := s.txBuilder.(client.ExtendedTxBuilder)
	require.True(t, ok)
	extendedBuilder.SetExtensionOptions(jwsOpt)

	tx, err := s.CreateTestTx(
		s.ctx,
		[]cryptotypes.PrivKey{signer.priv},
		[]uint64{0},
		[]uint64{0},
		s.ctx.ChainID(),
		signing.SignMode_SIGN_MODE_DIRECT,
	)
	require.NoError(t, err)
	return tx, userDID
}

func relayCoreKeeper(trusted sdk.AccAddress) extensionCoreKeeper {
	return extensionCoreKeeper{
		chainConfig: coremoduletypes.ChainConfig{IgnoreBearerAuth: true},
		params:      coremoduletypes.Params{TrustedRelayFeeGranters: []string{trusted.String()}},
	}
}

func anyRelayCoreKeeper() extensionCoreKeeper {
	return extensionCoreKeeper{
		chainConfig: coremoduletypes.ChainConfig{
			IgnoreBearerAuth: true,
			AllowAnyRelay:    true,
		},
	}
}

func nextAnte(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
	return ctx, nil
}

func TestExtensionOptionsDecorator_ValidJWSExtension(t *testing.T) {
	s := SetupTestSuite(t, true)
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()

	decorator := NewExtensionOptionsDecorator(nil)
	antehandler := sdk.ChainAnteDecorators(decorator)

	accs := s.CreateTestAccounts(1)

	msg := &acptypes.MsgCreatePolicy{
		Creator:     accs[0].acc.GetAddress().String(),
		Policy:      "name: test policy",
		MarshalType: coretypes.PolicyMarshalingType_YAML,
	}
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
	extractedDID := getExtractedDIDFromContext(newCtx)
	require.Equal(t, userDID, extractedDID, "DID should be extracted and stored in context")
}

func TestExtensionOptionsDecorator_InvalidJWSExtension(t *testing.T) {
	s := SetupTestSuite(t, true)
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()

	decorator := NewExtensionOptionsDecorator(nil)
	antehandler := sdk.ChainAnteDecorators(decorator)

	accs := s.CreateTestAccounts(1)

	msg := &acptypes.MsgCreatePolicy{
		Creator:     accs[0].acc.GetAddress().String(),
		Policy:      "name: test policy",
		MarshalType: coretypes.PolicyMarshalingType_YAML,
	}
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

	decorator := NewExtensionOptionsDecorator(nil)
	antehandler := sdk.ChainAnteDecorators(decorator)

	accs := s.CreateTestAccounts(1)

	msg := &acptypes.MsgCreatePolicy{
		Creator:     accs[0].acc.GetAddress().String(),
		Policy:      "name: test policy",
		MarshalType: coretypes.PolicyMarshalingType_YAML,
	}
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

	decorator := NewExtensionOptionsDecorator(nil)
	antehandler := sdk.ChainAnteDecorators(decorator)

	accs := s.CreateTestAccounts(1)

	msg := &acptypes.MsgCreatePolicy{
		Creator:     accs[0].acc.GetAddress().String(),
		Policy:      "name: test policy",
		MarshalType: coretypes.PolicyMarshalingType_YAML,
	}
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

	decorator := NewExtensionOptionsDecorator(nil)
	antehandler := sdk.ChainAnteDecorators(decorator)

	accs := s.CreateTestAccounts(1)

	msg := &acptypes.MsgCreatePolicy{
		Creator:     accs[0].acc.GetAddress().String(),
		Policy:      "name: test policy",
		MarshalType: coretypes.PolicyMarshalingType_YAML,
	}
	require.NoError(t, s.txBuilder.SetMsgs(msg))

	// Create an unknown extension option
	unknownOpt := &acptypes.MsgCreatePolicy{
		Creator:     accs[0].acc.GetAddress().String(),
		Policy:      "name: test policy",
		MarshalType: coretypes.PolicyMarshalingType_YAML,
	}

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

	decorator := NewExtensionOptionsDecorator(nil)
	antehandler := sdk.ChainAnteDecorators(decorator)

	accs := s.CreateTestAccounts(1)

	msg := &acptypes.MsgCreatePolicy{
		Creator:     accs[0].acc.GetAddress().String(),
		Policy:      "name: test policy",
		MarshalType: coretypes.PolicyMarshalingType_YAML,
	}
	require.NoError(t, s.txBuilder.SetMsgs(msg))

	// No extension options set

	privs, accNums, accSeqs := []cryptotypes.PrivKey{accs[0].priv}, []uint64{0}, []uint64{0}
	tx, err := s.CreateTestTx(s.ctx, privs, accNums, accSeqs, s.ctx.ChainID(), signing.SignMode_SIGN_MODE_DIRECT)
	require.NoError(t, err)

	newCtx, err := antehandler(s.ctx, tx, false)
	require.NoError(t, err, "ExtensionOptionsDecorator should allow transactions with no extension options")

	// Verify no DID was extracted
	extractedDID := getExtractedDIDFromContext(newCtx)
	require.Empty(t, extractedDID, "No DID should be extracted when no extension options are present")
}

func TestExtensionOptionsDecorator_MultipleExtensionOptions(t *testing.T) {
	s := SetupTestSuite(t, true)
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()

	decorator := NewExtensionOptionsDecorator(nil)
	antehandler := sdk.ChainAnteDecorators(decorator)

	accs := s.CreateTestAccounts(1)

	msg := &acptypes.MsgCreatePolicy{
		Creator:     accs[0].acc.GetAddress().String(),
		Policy:      "name: test policy",
		MarshalType: coretypes.PolicyMarshalingType_YAML,
	}
	require.NoError(t, s.txBuilder.SetMsgs(msg))

	// Create two JWS extension options
	authorizedAccount := accs[0].acc.GetAddress().String()
	bearerToken1, _ := test.GenerateSignedJWSWithMatchingDID(t, authorizedAccount)
	bearerToken2, _ := test.GenerateSignedJWSWithMatchingDID(t, authorizedAccount)

	jwsOpt1 := &antetypes.JWSExtensionOption{
		BearerToken: bearerToken1,
	}
	jwsOpt2 := &antetypes.JWSExtensionOption{
		BearerToken: bearerToken2,
	}

	// Pack both extension options
	any1, err := codectypes.NewAnyWithValue(jwsOpt1)
	require.NoError(t, err)
	any2, err := codectypes.NewAnyWithValue(jwsOpt2)
	require.NoError(t, err)

	// Add multiple extension options to the transaction
	if extBuilder, ok := s.txBuilder.(client.ExtendedTxBuilder); ok {
		extBuilder.SetExtensionOptions(any1, any2)
	} else {
		t.Fatal("TxBuilder does not implement ExtendedTxBuilder")
	}

	privs, accNums, accSeqs := []cryptotypes.PrivKey{accs[0].priv}, []uint64{0}, []uint64{0}
	tx, err := s.CreateTestTx(s.ctx, privs, accNums, accSeqs, s.ctx.ChainID(), signing.SignMode_SIGN_MODE_DIRECT)
	require.NoError(t, err)

	_, err = antehandler(s.ctx, tx, false)
	require.Error(t, err, "ExtensionOptionsDecorator should reject transactions with multiple extension options")
	require.Contains(t, err.Error(), "only one extension option is supported")
	require.Contains(t, err.Error(), "got 2")
}

func TestExtensionOptionsDecorator_JWSOptionWithInvalidSignature(t *testing.T) {
	s := SetupTestSuite(t, true)
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()

	decorator := NewExtensionOptionsDecorator(nil)
	antehandler := sdk.ChainAnteDecorators(decorator)

	accs := s.CreateTestAccounts(1)

	msg := &acptypes.MsgCreatePolicy{
		Creator:     accs[0].acc.GetAddress().String(),
		Policy:      "name: test policy",
		MarshalType: coretypes.PolicyMarshalingType_YAML,
	}
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

	decorator := NewExtensionOptionsDecorator(nil)
	antehandler := sdk.ChainAnteDecorators(decorator)

	accs := s.CreateTestAccounts(1)

	msg := &acptypes.MsgCreatePolicy{
		Creator:     accs[0].acc.GetAddress().String(),
		Policy:      "name: test policy",
		MarshalType: coretypes.PolicyMarshalingType_YAML,
	}
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
	extractedDID := getExtractedDIDFromContext(newCtx)
	require.Equal(t, userDID, extractedDID, "DID should be extracted from JWS payload")
}

func TestExtensionOptionsDecorator_ValidJWSWithDIDInPayload(t *testing.T) {
	s := SetupTestSuite(t, true)
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()

	decorator := NewExtensionOptionsDecorator(nil)
	antehandler := sdk.ChainAnteDecorators(decorator)

	accs := s.CreateTestAccounts(1)

	msg := &acptypes.MsgCreatePolicy{
		Creator:     accs[0].acc.GetAddress().String(),
		Policy:      "name: test policy",
		MarshalType: coretypes.PolicyMarshalingType_YAML,
	}
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
	extractedDID := getExtractedDIDFromContext(newCtx)
	require.Equal(t, userDID, extractedDID, "Payload DID should be extracted and used")
}

func TestExtensionOptionsDecorator_SecurityTamperedJWS(t *testing.T) {
	s := SetupTestSuite(t, true)
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()

	decorator := NewExtensionOptionsDecorator(nil)
	antehandler := sdk.ChainAnteDecorators(decorator)

	accs := s.CreateTestAccounts(1)

	msg := &acptypes.MsgCreatePolicy{
		Creator:     accs[0].acc.GetAddress().String(),
		Policy:      "name: test policy",
		MarshalType: coretypes.PolicyMarshalingType_YAML,
	}
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

	decorator := NewExtensionOptionsDecorator(nil)
	antehandler := sdk.ChainAnteDecorators(decorator)

	accs := s.CreateTestAccounts(1)

	msg := &acptypes.MsgCreatePolicy{
		Creator:     accs[0].acc.GetAddress().String(),
		Policy:      "name: test policy",
		MarshalType: coretypes.PolicyMarshalingType_YAML,
	}
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
	extensionDecorator := NewExtensionOptionsDecorator(nil)
	feeDecorator := NewCustomDeductFeeDecorator(s.accountKeeper, s.bankKeeper, mockFeegrantKeeper, nil, nil)

	// Chain them in the same order as actual ante handler
	antehandler := sdk.ChainAnteDecorators(extensionDecorator, feeDecorator)

	accs := s.CreateTestAccounts(2)
	feePayer := accs[0].acc.GetAddress()
	feeGranter := accs[1].acc.GetAddress()

	msg := &acptypes.MsgCreatePolicy{
		Creator:     accs[0].acc.GetAddress().String(),
		Policy:      "name: test policy",
		MarshalType: coretypes.PolicyMarshalingType_YAML,
	}
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
	mockFeegrantKeeper.EXPECT().UseFirstAvailableDIDGrant(gomock.Any(), userDID, validFee, gomock.Any()).Return(feeGranter, nil)
	// Fee deduction happens in the ante handler after feegrant validation
	s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), feeGranter, authtypes.FeeCollectorName, validFee).Return(nil)

	// Create transaction
	privs, accNums, accSeqs := []cryptotypes.PrivKey{accs[0].priv}, []uint64{0}, []uint64{0}
	tx, err := s.CreateTestTx(s.ctx, privs, accNums, accSeqs, s.ctx.ChainID(), signing.SignMode_SIGN_MODE_DIRECT)
	require.NoError(t, err)

	// Run the ante handler chain
	newCtx, err := antehandler(s.ctx, tx, false)
	require.NoError(t, err, "Ante handler chain should succeed with DID-based feegrant")

	// Verify DID was passed through context
	extractedDID := getExtractedDIDFromContext(newCtx)
	require.Equal(t, userDID, extractedDID, "DID should be preserved in context after ante chain")
}

func TestExtensionAndFeeDecorators_NoDID(t *testing.T) {
	s := SetupTestSuite(t, true)
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()

	// Create both decorators
	extensionDecorator := NewExtensionOptionsDecorator(nil)
	feeDecorator := NewCustomDeductFeeDecorator(s.accountKeeper, s.bankKeeper, s.feeGrantKeeper, nil, nil)

	// Chain them in the same order as actual ante handler
	antehandler := sdk.ChainAnteDecorators(extensionDecorator, feeDecorator)

	accs := s.CreateTestAccounts(2)
	feePayer := accs[0].acc.GetAddress()
	feeGranter := accs[1].acc.GetAddress()

	msg := &acptypes.MsgCreatePolicy{
		Creator:     accs[0].acc.GetAddress().String(),
		Policy:      "name: test policy",
		MarshalType: coretypes.PolicyMarshalingType_YAML,
	}
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
	extractedDID := getExtractedDIDFromContext(newCtx)
	require.Empty(t, extractedDID, "No DID should be in context when no extension options provided")
}

func TestExtensionOptionsDecorator_CorrectAuthorizedAccount(t *testing.T) {
	s := SetupTestSuite(t, true)
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()

	decorator := NewExtensionOptionsDecorator(nil)
	antehandler := sdk.ChainAnteDecorators(decorator)

	accs := s.CreateTestAccounts(1)

	msg := &acptypes.MsgCreatePolicy{
		Creator:     accs[0].acc.GetAddress().String(),
		Policy:      "name: test policy",
		MarshalType: coretypes.PolicyMarshalingType_YAML,
	}
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
	extractedDID := getExtractedDIDFromContext(newCtx)
	require.Equal(t, userDID, extractedDID, "DID should be extracted and stored in context")
}

func TestExtensionOptionsDecorator_IncorrectAuthorizedAccount(t *testing.T) {
	s := SetupTestSuite(t, true)
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()

	decorator := NewExtensionOptionsDecorator(nil)
	antehandler := sdk.ChainAnteDecorators(decorator)

	accs := s.CreateTestAccounts(1)

	msg := &acptypes.MsgCreatePolicy{
		Creator:     accs[0].acc.GetAddress().String(),
		Policy:      "name: test policy",
		MarshalType: coretypes.PolicyMarshalingType_YAML,
	}
	require.NoError(t, s.txBuilder.SetMsgs(msg))

	// Create JWS token with authorized account NOT matching the transaction signer
	wrongAuthorizedAccount := "vera1wjj5v5rlf57kayyeskncpu4hwev25ty697gcev"
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
	require.Contains(t, err.Error(), "signer mismatch")
}

func TestExtensionOptionsDecorator_MultipleMessagesAllAuthorized(t *testing.T) {
	s := SetupTestSuite(t, true)
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()

	decorator := NewExtensionOptionsDecorator(nil)
	antehandler := sdk.ChainAnteDecorators(decorator)

	accs := s.CreateTestAccounts(1)

	// Multiple ACP messages from same creator
	msg1 := &acptypes.MsgCreatePolicy{
		Creator:     accs[0].acc.GetAddress().String(),
		Policy:      "policy 1",
		MarshalType: coretypes.PolicyMarshalingType_YAML,
	}
	msg2 := &acptypes.MsgEditPolicy{
		Creator:     accs[0].acc.GetAddress().String(),
		PolicyId:    "policy-id-1",
		Policy:      "updated policy",
		MarshalType: coretypes.PolicyMarshalingType_YAML,
	}
	msg3 := &acptypes.MsgCreatePolicy{
		Creator:     accs[0].acc.GetAddress().String(),
		Policy:      "policy 2",
		MarshalType: coretypes.PolicyMarshalingType_YAML,
	}
	require.NoError(t, s.txBuilder.SetMsgs(msg1, msg2, msg3))

	authorizedAccount := accs[0].acc.GetAddress().String()
	bearerToken, userDID := test.GenerateSignedJWSWithMatchingDID(t, authorizedAccount)
	jwsOpt := &antetypes.JWSExtensionOption{
		BearerToken: bearerToken,
	}

	any, err := codectypes.NewAnyWithValue(jwsOpt)
	require.NoError(t, err)

	if extBuilder, ok := s.txBuilder.(client.ExtendedTxBuilder); ok {
		extBuilder.SetExtensionOptions(any)
	} else {
		t.Fatal("TxBuilder does not implement ExtendedTxBuilder")
	}

	privs, accNums, accSeqs := []cryptotypes.PrivKey{accs[0].priv}, []uint64{0}, []uint64{0}
	tx, err := s.CreateTestTx(s.ctx, privs, accNums, accSeqs, s.ctx.ChainID(), signing.SignMode_SIGN_MODE_DIRECT)
	require.NoError(t, err)

	newCtx, err := antehandler(s.ctx, tx, false)
	require.NoError(t, err, "ExtensionOptionsDecorator should accept transaction with multiple ACP messages all from authorized account")

	extractedDID := getExtractedDIDFromContext(newCtx)
	require.Equal(t, userDID, extractedDID, "DID should be extracted and stored in context")
}

func TestExtensionOptionsDecorator_MultipleMessagesOneUnauthorized(t *testing.T) {
	s := SetupTestSuite(t, true)
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()

	decorator := NewExtensionOptionsDecorator(nil)
	antehandler := sdk.ChainAnteDecorators(decorator)

	accs := s.CreateTestAccounts(2)

	// Mix of ACP messages from different creators - should be rejected
	msg1 := &acptypes.MsgCreatePolicy{
		Creator:     accs[0].acc.GetAddress().String(),
		Policy:      "policy 1",
		MarshalType: coretypes.PolicyMarshalingType_YAML,
	}
	msg2 := &acptypes.MsgEditPolicy{
		Creator:     accs[1].acc.GetAddress().String(), // Different creator - unauthorized
		PolicyId:    "policy-id-1",
		Policy:      "malicious update",
		MarshalType: coretypes.PolicyMarshalingType_YAML,
	}
	msg3 := &acptypes.MsgCreatePolicy{
		Creator:     accs[0].acc.GetAddress().String(),
		Policy:      "policy 2",
		MarshalType: coretypes.PolicyMarshalingType_YAML,
	}
	require.NoError(t, s.txBuilder.SetMsgs(msg1, msg2, msg3))

	// JWS token authorized only for accs[0]
	authorizedAccount := accs[0].acc.GetAddress().String()
	bearerToken, _ := test.GenerateSignedJWSWithMatchingDID(t, authorizedAccount)
	jwsOpt := &antetypes.JWSExtensionOption{
		BearerToken: bearerToken,
	}

	any, err := codectypes.NewAnyWithValue(jwsOpt)
	require.NoError(t, err)

	if extBuilder, ok := s.txBuilder.(client.ExtendedTxBuilder); ok {
		extBuilder.SetExtensionOptions(any)
	} else {
		t.Fatal("TxBuilder does not implement ExtendedTxBuilder")
	}

	privs, accNums, accSeqs := []cryptotypes.PrivKey{accs[0].priv, accs[1].priv}, []uint64{0, 0}, []uint64{0, 0}
	tx, err := s.CreateTestTx(s.ctx, privs, accNums, accSeqs, s.ctx.ChainID(), signing.SignMode_SIGN_MODE_DIRECT)
	require.NoError(t, err)

	_, err = antehandler(s.ctx, tx, false)
	require.Error(t, err, "ExtensionOptionsDecorator should reject transaction with ACP message from unauthorized account")
	require.Contains(t, err.Error(), "message 1 signer mismatch", "Error should indicate which message failed")
}

func TestExtensionOptionsDecorator_TrustedRelay(t *testing.T) {
	s := SetupTestSuite(t, true)
	accs := s.CreateTestAccounts(2)
	worker := accs[0].acc.GetAddress()
	feeGranter := accs[1].acc.GetAddress()
	tx, userDID := buildExtensionTestTx(t, s, accs[0], worker.String(), feeGranter, "", nil)

	newCtx, err := NewExtensionOptionsDecorator(relayCoreKeeper(feeGranter)).
		AnteHandle(s.ctx, tx, false, nextAnte)
	require.NoError(t, err)
	require.Equal(t, userDID, getExtractedDIDFromContext(newCtx))
	require.Equal(t, feeGranter.String(), getTrustedRelayFeeGranterFromContext(newCtx))
}

func TestExtensionOptionsDecorator_RejectsProviderIdentityWithoutTrustedRelay(t *testing.T) {
	s := SetupTestSuite(t, true)
	account := s.CreateTestAccounts(1)[0]
	address := account.acc.GetAddress()
	tx, _ := buildExtensionTestTx(t, s, account, address.String(), nil, address.String(), &antetypes.ProviderToken{
		ProviderName: "google",
		UserID:       "victim@example.com",
		ActorDID:     "did:opk:victim",
	})

	_, err := NewExtensionOptionsDecorator(nil).AnteHandle(s.ctx, tx, false, nextAnte)
	require.ErrorContains(t, err, "provider token requires a trusted relay")
}

func TestExtensionOptionsDecorator_AcceptsProviderIdentityFromTrustedRelay(t *testing.T) {
	s := SetupTestSuite(t, true)
	accs := s.CreateTestAccounts(2)
	worker := accs[0].acc.GetAddress()
	feeGranter := accs[1].acc.GetAddress()
	actorDID := "did:opk:user"
	tx, _ := buildExtensionTestTx(t, s, accs[0], worker.String(), feeGranter, "", &antetypes.ProviderToken{
		ProviderName: "google",
		UserID:       "user@example.com",
		ActorDID:     actorDID,
	})

	newCtx, err := NewExtensionOptionsDecorator(relayCoreKeeper(feeGranter)).
		AnteHandle(s.ctx, tx, false, nextAnte)
	require.NoError(t, err)
	require.Equal(t, actorDID, getExtractedDIDFromContext(newCtx))
	require.Equal(t, feeGranter.String(), getTrustedRelayFeeGranterFromContext(newCtx))
}

func TestExtensionOptionsDecorator_RejectsUntrustedRelay(t *testing.T) {
	s := SetupTestSuite(t, true)
	accs := s.CreateTestAccounts(3)
	worker := accs[0].acc.GetAddress()
	feeGranter := accs[1].acc.GetAddress()
	trusted := accs[2].acc.GetAddress()
	tx, _ := buildExtensionTestTx(t, s, accs[0], worker.String(), feeGranter, "", nil)

	_, err := NewExtensionOptionsDecorator(relayCoreKeeper(trusted)).
		AnteHandle(s.ctx, tx, false, nextAnte)
	require.ErrorContains(t, err, "is not a trusted relay")
}

func TestExtensionOptionsDecorator_RejectsRelayWithoutFeeGranter(t *testing.T) {
	s := SetupTestSuite(t, true)
	accs := s.CreateTestAccounts(2)
	worker := accs[0].acc.GetAddress()
	trusted := accs[1].acc.GetAddress()
	tx, _ := buildExtensionTestTx(t, s, accs[0], worker.String(), nil, "", nil)

	_, err := NewExtensionOptionsDecorator(relayCoreKeeper(trusted)).
		AnteHandle(s.ctx, tx, false, nextAnte)
	require.ErrorContains(t, err, "requires a trusted fee granter")
}

func TestExtensionOptionsDecorator_RejectsMessageNotSignedByRelayWorker(t *testing.T) {
	s := SetupTestSuite(t, true)
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()
	accs := s.CreateTestAccounts(3)
	worker := accs[0].acc.GetAddress()
	creator := accs[1].acc.GetAddress()
	trusted := accs[2].acc.GetAddress()

	require.NoError(t, s.txBuilder.SetMsgs(&acptypes.MsgCreatePolicy{
		Creator:     creator.String(),
		Policy:      "name: test policy",
		MarshalType: coretypes.PolicyMarshalingType_YAML,
	}))
	s.txBuilder.SetFeePayer(worker)
	s.txBuilder.SetFeeGranter(trusted)

	keeper := extensionCoreKeeper{
		chainConfig: coremoduletypes.ChainConfig{IgnoreBearerAuth: true},
		params:      coremoduletypes.Params{TrustedRelayFeeGranters: []string{trusted.String()}},
	}
	_, err := NewExtensionOptionsDecorator(keeper).validateRelay(s.ctx, s.txBuilder.GetTx())
	require.ErrorContains(t, err, "does not match relay worker")
}

func TestExtensionOptionsDecorator_AllowsUnlistedRelayWhenAllowAnyRelay(t *testing.T) {
	s := SetupTestSuite(t, true)
	accs := s.CreateTestAccounts(2)
	worker := accs[0].acc.GetAddress()
	feeGranter := accs[1].acc.GetAddress()
	tx, userDID := buildExtensionTestTx(t, s, accs[0], worker.String(), feeGranter, "", nil)

	newCtx, err := NewExtensionOptionsDecorator(anyRelayCoreKeeper()).
		AnteHandle(s.ctx, tx, false, nextAnte)
	require.NoError(t, err)
	require.Equal(t, userDID, getExtractedDIDFromContext(newCtx))
	require.Equal(t, feeGranter.String(), getTrustedRelayFeeGranterFromContext(newCtx))
}

func TestExtensionOptionsDecorator_PropagatesTokenStoreFailure(t *testing.T) {
	s := SetupTestSuite(t, true)
	accs := s.CreateTestAccounts(1)
	creator := accs[0].acc.GetAddress().String()
	tx, _ := buildExtensionTestTx(t, s, accs[0], creator, nil, creator, nil)

	_, err := NewExtensionOptionsDecorator(extensionCoreKeeper{storeErr: errors.New("token invalidated")}).
		AnteHandle(s.ctx, tx, false, nextAnte)
	require.ErrorContains(t, err, "bearer token is not active")
}
