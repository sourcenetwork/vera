package main

import (
	"context"
	"log"

	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocdc "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"

	"github.com/sourcenetwork/sourcehub/sdk"
	"github.com/sourcenetwork/sourcehub/x/acp/bearer_token"
	"github.com/sourcenetwork/sourcehub/x/acp/did"
	acptypes "github.com/sourcenetwork/sourcehub/x/acp/types"
)

func main() {
	ctx := context.Background()

	client, err := sdk.NewClient()
	if err != nil {
		log.Fatal(err)
	}

	txBuilder, err := sdk.NewTxBuilder(
		sdk.WithSDKClient(client),
		sdk.WithChainID("sourcehub-dev"),
	)
	if err != nil {
		log.Fatal(err)
	}

	reg := cdctypes.NewInterfaceRegistry()
	cryptocdc.RegisterInterfaces(reg)
	cdc := codec.NewProtoCodec(reg)
	keyring, err := keyring.New("sourcehub", keyring.BackendTest, "~/.sourcehub", nil, cdc)
	if err != nil {
		log.Fatalf("could not load keyring: %v", err)
	}

	txSigner, err := sdk.NewTxSignerFromKeyringKey(keyring, "validator")
	if err != nil {
		log.Fatalf("could not load keyring: %v", err)
	}

	policy := `
name: test
resources:
  resource:
    relations:
      owner:
        types:
          - actor
`

	msgSet := sdk.MsgSet{}
	policyMapper := msgSet.WithCreatePolicy(acptypes.NewMsgCreatePolicy(txSigner.GetAccAddress(), policy, coretypes.PolicyMarshalingType_SHORT_YAML))
	tx, err := txBuilder.Build(ctx, txSigner, &msgSet)
	if err != nil {
		log.Fatal(err)
	}

	resp, err := client.BroadcastTx(ctx, tx)
	if err != nil {
		log.Fatal(err)
	}

	result, err := client.AwaitTx(ctx, resp.TxHash)
	if err != nil {
		log.Fatal(err)
	}
	if result.Error() != nil {
		log.Fatalf("Tx failed: %v", result.Error())
	}

	policyResponse, err := policyMapper.Map(result.TxPayload())
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("policy created: %v", policyResponse.Record.Policy.Id)

	alice, signer, err := did.ProduceDID()
	if err != nil {
		log.Fatalf("could not generate alice: %v", err)
	}
	log.Printf("alice's DID: %v", alice)

	token := bearer_token.NewBearerTokenNow(alice, txSigner.GetAccAddress())
	jws, err := token.ToJWS(signer)
	if err != nil {
		log.Fatalf("could not produce Bearer for alice: %v", err)
	}

	log.Printf("alice's raw token: %v", token)
	log.Printf("alice's JWS: %v", jws)

	bearerCmd := acptypes.MsgBearerPolicyCmd{
		Creator:     txSigner.GetAccAddress(),
		BearerToken: jws,
		PolicyId:    policyResponse.Record.Policy.Id,
		Cmd:         acptypes.NewRegisterObjectCmd(coretypes.NewObject("resource", "foo")),
	}

	log.Printf("Bearer Cmd: %v", bearerCmd)
	msgSet = sdk.MsgSet{}
	msgSet.WithBearerPolicyCmd(&bearerCmd)
	tx, err = txBuilder.Build(ctx, txSigner, &msgSet)
	if err != nil {
		log.Fatal(err)
	}
	resp, err = client.BroadcastTx(ctx, tx)
	if err != nil {
		log.Fatal(err)
	}

	result, err = client.AwaitTx(ctx, resp.TxHash)
	if err != nil {
		log.Fatal(err)
	}
	if result.Error() != nil {
		log.Fatalf("Tx failed: %v", result.Error())
	}

	log.Printf("object registered")
}
