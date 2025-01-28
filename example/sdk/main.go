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

	cmdBuilder, err := sdk.NewCmdBuilder(ctx, client)
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

	signer, err := sdk.NewTxSignerFromKeyringKey(keyring, "validator")
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
	policyMapper := msgSet.WithCreatePolicy(acptypes.NewMsgCreatePolicy(signer.GetAccAddress(), policy, coretypes.PolicyMarshalingType_SHORT_YAML))
	tx, err := txBuilder.Build(ctx, signer, &msgSet)
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

	log.Printf("policy created: %v", policyResponse)

	// create cmd which registers an object for the given did
	aliceDid, aliceSigner, err := did.ProduceDID()
	if err != nil {
		log.Fatal(err)
	}
	cmdBuilder.PolicyCmd(acptypes.NewRegisterObjectCmd(coretypes.NewObject("resource", "readme.txt")))
	cmdBuilder.PolicyID(policyResponse.Record.Policy.Id)
	cmdBuilder.Actor(aliceDid)
	cmdBuilder.SetSigner(aliceSigner)
	jws, err := cmdBuilder.BuildJWS(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// wraps cmd in a tx and broadcasts it
	msgSet = sdk.MsgSet{}
	msgSet.WithSignedPolicyCmd(acptypes.NewMsgSignedPolicyCmdFromJWS(signer.GetAccAddress(), jws))
	tx, err = txBuilder.Build(ctx, signer, &msgSet)
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
