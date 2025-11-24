package ante

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"testing"

	"cosmossdk.io/math"
	"github.com/TBD54566975/ssi-sdk/crypto"
	"github.com/TBD54566975/ssi-sdk/did/key"
	"github.com/cosmos/cosmos-sdk/client"
	clienttx "github.com/cosmos/cosmos-sdk/client/tx"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	jwstypes "github.com/sourcenetwork/sourcehub/app/ante/types"
	testutil "github.com/sourcenetwork/sourcehub/testutil"
	"github.com/sourcenetwork/sourcehub/testutil/network"
	acptypes "github.com/sourcenetwork/sourcehub/x/acp/types"
	"github.com/sourcenetwork/sourcehub/x/feegrant"
)

// TestJWSExtensionOptionWithDIDBasedFeegrant ensures that tx with JWS extension succeeds using the DID-based feegrant.
func TestJWSExtensionOptionWithDIDBasedFeegrant(t *testing.T) {
	net := network.NewWithOptions(t, network.NetworkOptions{EnableFaucet: true})

	val := net.Validators[0]
	clientCtx := val.ClientCtx

	_, err := net.WaitForHeight(1)
	require.NoError(t, err)

	// Import faucet key into validator keyring
	faucetMnemonic := "comic very pond victory suit tube ginger antique life then core warm loyal deliver iron fashion erupt husband weekend monster sunny artist empty uphold"
	_, err = clientCtx.Keyring.NewAccount("faucet", faucetMnemonic, "", "m/44'/118'/0'/0/0", hd.Secp256k1)
	require.NoError(t, err)

	// Generate DID keypair
	mnemonic := "near smoke great nasty alley food crush nurse rubber say danger search employ under gaze today alien eager risk letter drum relief sponsor current"
	seed, err := hd.Secp256k1.Derive()(mnemonic, "", "m/44'/118'/0'/0/0")
	require.NoError(t, err)

	if len(seed) > 32 {
		seed = seed[:32]
	}
	privKey := ed25519.NewKeyFromSeed(seed)
	pubKey := privKey.Public().(ed25519.PublicKey)
	didKey, err := key.CreateDIDKey(crypto.Ed25519, pubKey)
	require.NoError(t, err)
	did := didKey.String()

	// Add DID-based allowance from validator to DID
	valAddr := sdk.AccAddress(val.Address)
	faucetAddr := sdk.MustAccAddressFromBech32("source12d9hjf0639k995venpv675sju9ltsvf8u5c9jt")

	// Create basic allowance for the DID
	basicAllowance := &feegrant.BasicAllowance{
		SpendLimit: sdk.NewCoins(sdk.NewCoin("uopen", math.NewInt(1000000))),
	}

	msgGrantAllowance := &feegrant.MsgGrantDIDAllowance{
		Granter:    valAddr.String(),
		GranteeDid: did,
		Allowance:  &codectypes.Any{},
	}

	// Pack the allowance
	allowanceAny, err := codectypes.NewAnyWithValue(basicAllowance)
	require.NoError(t, err)
	msgGrantAllowance.Allowance = allowanceAny

	// Build and send grant transaction
	txBuilder := clientCtx.TxConfig.NewTxBuilder()
	err = txBuilder.SetMsgs(msgGrantAllowance)
	require.NoError(t, err)

	txBuilder.SetGasLimit(200000)
	txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin("uopen", math.NewInt(5000))))

	// Create gRPC connection
	grpcAddr := net.Validators[0].AppConfig.GRPC.Address
	conn, err := grpc.NewClient(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()

	// Query validator account to get proper account number and sequence
	authClient := authtypes.NewQueryClient(conn)
	accountResp, err := authClient.Account(context.Background(), &authtypes.QueryAccountRequest{
		Address: valAddr.String(),
	})
	require.NoError(t, err)

	var account sdk.AccountI
	err = clientCtx.InterfaceRegistry.UnpackAny(accountResp.Account, &account)
	require.NoError(t, err)

	accNum := account.GetAccountNumber()
	seq := account.GetSequence()

	// Create transaction factory with proper account info
	txf := clienttx.Factory{}.
		WithTxConfig(clientCtx.TxConfig).
		WithChainID(clientCtx.ChainID).
		WithKeybase(clientCtx.Keyring).
		WithSignMode(signing.SignMode_SIGN_MODE_DIRECT).
		WithAccountRetriever(clientCtx.AccountRetriever).
		WithAccountNumber(accNum).
		WithSequence(seq)

	err = clienttx.Sign(context.Background(), txf, val.Moniker, txBuilder, true)
	require.NoError(t, err)

	txBytes, err := clientCtx.TxConfig.TxEncoder()(txBuilder.GetTx())
	require.NoError(t, err)

	res, err := clientCtx.BroadcastTxSync(txBytes)
	require.NoError(t, err)
	require.Equal(t, uint32(0), res.Code, "grant allowance check tx failed: %s", res.RawLog)

	// Wait for transaction to be processed
	_, err = net.WaitForHeight(3)
	require.NoError(t, err)

	// Query the transaction to verify it was actually executed successfully
	txClient := txtypes.NewServiceClient(conn)
	txResp, err := txClient.GetTx(context.Background(), &txtypes.GetTxRequest{
		Hash: res.TxHash,
	})
	require.NoError(t, err, "should be able to query grant transaction")
	require.NotNil(t, txResp.TxResponse, "tx response should not be nil")
	require.Equal(t, uint32(0), txResp.TxResponse.Code, "grant transaction should succeed in block: %s", txResp.TxResponse.RawLog)

	// Verify the DID allowance was created successfully via gRPC query
	feegrantClient := feegrant.NewQueryClient(conn)
	queryResp, err := feegrantClient.DIDAllowance(context.Background(), &feegrant.QueryDIDAllowanceRequest{
		Granter:    valAddr.String(),
		GranteeDid: did,
	})
	require.NoError(t, err, "DID allowance should exist")
	require.NotNil(t, queryResp.Allowance, "DID allowance should not be nil")

	// Create policy message with JWS extension
	policyContent := `
description: Base policy that defines permissions for bulletin namespaces
name: Bulletin Policy
resources:
- name: namespace
  permissions:
  - expr: owner + collaborator
    name: create_post
  relations:
  - name: collaborator
    types:
    - actor
  - name: owner
    types:
    - actor
`

	msg := &acptypes.MsgCreatePolicy{
		Creator:     faucetAddr.String(),
		Policy:      policyContent,
		MarshalType: coretypes.PolicyMarshalingType_YAML,
	}

	// Create bearer token payload
	bearerToken := testutil.NewBearerTokenNow(did, faucetAddr.String())
	payloadBytes, err := json.Marshal(bearerToken)
	require.NoError(t, err)

	// Create and sign JWS
	header := testutil.CreateJWSHeader()
	headerBytes, err := json.Marshal(header)
	require.NoError(t, err)

	headerEncoded := base64.RawURLEncoding.EncodeToString(headerBytes)
	payloadEncoded := base64.RawURLEncoding.EncodeToString(payloadBytes)
	signingInput := headerEncoded + "." + payloadEncoded
	signature := ed25519.Sign(privKey, []byte(signingInput))
	signatureEncoded := base64.RawURLEncoding.EncodeToString(signature)
	jwsString := signingInput + "." + signatureEncoded

	// Create extension option
	ext := &jwstypes.JWSExtensionOption{
		BearerToken: jwsString,
	}

	extAny, err := codectypes.NewAnyWithValue(ext)
	require.NoError(t, err)

	// Build transaction with JWS extension
	txBuilder = clientCtx.TxConfig.NewTxBuilder()
	err = txBuilder.SetMsgs(msg)
	require.NoError(t, err)

	// Set extension options
	if extBuilder, ok := txBuilder.(client.ExtendedTxBuilder); ok {
		extBuilder.SetExtensionOptions(extAny)
	} else {
		t.Fatal("TxBuilder does not implement ExtendedTxBuilder")
	}

	txBuilder.SetGasLimit(200000)
	txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin("uopen", math.NewInt(5000))))
	txBuilder.SetFeeGranter(valAddr)

	faucetAccountResp, err := authClient.Account(context.Background(), &authtypes.QueryAccountRequest{
		Address: faucetAddr.String(),
	})
	require.NoError(t, err)

	err = clientCtx.InterfaceRegistry.UnpackAny(faucetAccountResp.Account, &account)
	require.NoError(t, err)

	faucetAccNum := account.GetAccountNumber()
	faucetSeq := account.GetSequence()

	// Sign the transaction
	txf2 := clienttx.Factory{}.
		WithTxConfig(clientCtx.TxConfig).
		WithChainID(clientCtx.ChainID).
		WithKeybase(clientCtx.Keyring).
		WithSignMode(signing.SignMode_SIGN_MODE_DIRECT).
		WithAccountRetriever(clientCtx.AccountRetriever).
		WithAccountNumber(faucetAccNum).
		WithSequence(faucetSeq)

	err = clienttx.Sign(context.Background(), txf2, "faucet", txBuilder, true)
	require.NoError(t, err)

	// Encode and broadcast
	txBytes, err = clientCtx.TxConfig.TxEncoder()(txBuilder.GetTx())
	require.NoError(t, err)

	res, err = clientCtx.BroadcastTxSync(txBytes)
	require.NoError(t, err)
	require.Equal(t, uint32(0), res.Code, "create policy transaction with JWS failed: %s", res.RawLog)
}
