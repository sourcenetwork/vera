package main

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"cosmossdk.io/math"
	rpchttp "github.com/cometbft/cometbft/rpc/client/http"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	clienttx "github.com/cosmos/cosmos-sdk/client/tx"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/TBD54566975/ssi-sdk/crypto"
	"github.com/TBD54566975/ssi-sdk/did/key"
	testutilmod "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/cosmos/cosmos-sdk/x/auth"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	jwstypes "github.com/sourcenetwork/vera/app/ante/types"
	testutil "github.com/sourcenetwork/vera/testutil"
	acp "github.com/sourcenetwork/vera/x/acp/module"
	acptypes "github.com/sourcenetwork/vera/x/acp/types"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run main.go <policy-file>")
	}

	// Read the policy file content
	policyContent, err := os.ReadFile(os.Args[1])
	if err != nil {
		log.Fatal("Failed to read policy file:", err)
	}

	// Initialize SDK config
	sdkConfig := sdk.GetConfig()
	sdkConfig.SetBech32PrefixForAccount("vera", "verapub")
	sdkConfig.Seal()

	// Setup encoding config with auth and acp modules
	encodingConfig := testutilmod.MakeTestEncodingConfig(auth.AppModuleBasic{}, acp.AppModuleBasic{})

	// Setup RPC client
	rpcClient, err := rpchttp.New("tcp://localhost:26657", "/websocket")
	if err != nil {
		log.Fatal("Failed to create RPC client:", err)
	}

	// Setup gRPC connection
	grpcConn, err := grpc.NewClient("localhost:9090", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal("Failed to create gRPC connection:", err)
	}
	defer grpcConn.Close()

	// Setup client context
	clientCtx := client.Context{}.
		WithChainID("vera-dev").
		WithKeyringDir(".vera").
		WithTxConfig(encodingConfig.TxConfig).
		WithCodec(encodingConfig.Codec).
		WithClient(rpcClient).
		WithNodeURI("tcp://localhost:26657").
		WithAccountRetriever(authtypes.AccountRetriever{}).
		WithGRPCClient(grpcConn).
		WithBroadcastMode(flags.BroadcastSync).
		WithSkipConfirmation(true)

	// Load keyring from home directory
	homeDir := os.Getenv("HOME") + "/.vera"
	kr, err := keyring.New(sdk.KeyringServiceName(), "test", homeDir, nil, encodingConfig.Codec)
	if err != nil {
		log.Fatal("Failed to create keyring:", err)
	}

	keyName := "validator"
	info, err := kr.Key(keyName)
	if err != nil {
		log.Fatal("Key not found in keyring:", err)
	}

	addr, err := info.GetAddress()
	if err != nil {
		log.Fatal("Failed to get address from key:", err)
	}
	senderAddr := addr.String()

	clientCtx = clientCtx.
		WithFromAddress(addr).
		WithFromName(info.Name).
		WithKeyring(kr)

	// Create the message
	msg := &acptypes.MsgCreatePolicy{
		Creator:     senderAddr,
		Policy:      string(policyContent),
		MarshalType: coretypes.PolicyMarshalingType_YAML,
	}

	// Generate the key pair for signing
	var privKey ed25519.PrivateKey
	var pubKey ed25519.PublicKey

	// Derive seed from mnemonic
	mnemonic := "near smoke great nasty alley food crush nurse rubber say danger search employ under gaze today alien eager risk letter drum relief sponsor current"
	seed, err := hd.Secp256k1.Derive()(mnemonic, "", "m/44'/118'/0'/0/0")
	if err != nil {
		log.Fatal("Failed to derive seed from mnemonic:", err)
	}

	// Generate Ed25519 key pair from the derived seed
	if len(seed) > 32 {
		seed = seed[:32]
	}
	privKey = ed25519.NewKeyFromSeed(seed)
	pubKey = privKey.Public().(ed25519.PublicKey)
	didKey, err := key.CreateDIDKey(crypto.Ed25519, pubKey)
	if err != nil {
		log.Fatal("Failed to create DID key:", err)
	}

	// Create bearer token with the matching DID
	bearerToken := testutil.NewBearerTokenNow(didKey.String(), senderAddr)
	payloadBytes, err := json.Marshal(bearerToken)
	if err != nil {
		log.Fatal("Failed to marshal payload:", err)
	}

	// Create and sign the JWS
	header := testutil.CreateJWSHeader()
	headerBytes, err := json.Marshal(header)
	if err != nil {
		log.Fatal("Failed to marshal header:", err)
	}

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
	if err != nil {
		log.Fatal("Failed to marshal extension:", err)
	}

	// Build transaction
	txBuilder := clientCtx.TxConfig.NewTxBuilder()
	err = txBuilder.SetMsgs(msg)
	if err != nil {
		log.Fatal("Failed to set messages:", err)
	}

	// Set extension options
	if extBuilder, ok := txBuilder.(client.ExtendedTxBuilder); ok {
		extBuilder.SetExtensionOptions(extAny)
	} else {
		log.Fatal("TxBuilder does not implement ExtendedTxBuilder")
	}
	txBuilder.SetGasLimit(200000)
	txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin("uopen", math.NewInt(5000))))

	// Query account to get acc number and sequence
	authClient := authtypes.NewQueryClient(grpcConn)
	accountResp, err := authClient.Account(context.Background(), &authtypes.QueryAccountRequest{
		Address: addr.String(),
	})
	if err != nil {
		log.Fatal("Failed to query account:", err)
	}

	var account sdk.AccountI
	err = encodingConfig.InterfaceRegistry.UnpackAny(accountResp.Account, &account)
	if err != nil {
		log.Fatal("Failed to unpack account:", err)
	}

	accNum := account.GetAccountNumber()
	seq := account.GetSequence()

	// Create transaction factory
	txf := clienttx.Factory{}.
		WithTxConfig(clientCtx.TxConfig).
		WithChainID(clientCtx.ChainID).
		WithKeybase(kr).
		WithSignMode(signing.SignMode_SIGN_MODE_DIRECT).
		WithAccountRetriever(clientCtx.AccountRetriever).
		WithAccountNumber(accNum).
		WithSequence(seq)

	// Sign transaction
	err = clienttx.Sign(context.Background(), txf, info.Name, txBuilder, true)
	if err != nil {
		log.Fatal("Failed to sign transaction:", err)
	}

	// Encode transaction
	txBytes, err := clientCtx.TxConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		log.Fatal("Failed to encode transaction:", err)
	}

	// Broadcast transaction
	res, err := clientCtx.BroadcastTx(txBytes)
	if err != nil {
		log.Fatal("Failed to broadcast transaction:", err)
	}

	output, _ := json.MarshalIndent(res, "", "  ")
	fmt.Println("Transaction Result:")
	fmt.Println(string(output))
}
