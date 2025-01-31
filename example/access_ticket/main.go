package main

import (
	"context"
	"log"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocdc "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/davecgh/go-spew/spew"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/spf13/cobra"

	"github.com/sourcenetwork/sourcehub/sdk"
	"github.com/sourcenetwork/sourcehub/x/acp/access_ticket"
	acptypes "github.com/sourcenetwork/sourcehub/x/acp/types"
)

const nodeAddrDefault = "tcp://localhost:26657"

const chainIdFlag = "chain-id"
const nodeAddrFlag = "node-addr"

var policy string = `
name: access ticket example
resources:
  file:
    relations:
      owner:
        types:
          - actor
    permissions:
      read:
        expr: owner
`

func main() {
	cmd := cobra.Command{
		Use:   "access_ticket_demo [validator-key-name]",
		Short: "acces_ticket_demo executes a self-contained example of the access ticket flow. Receives name of the validator key to send Txs from",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]
			runDemo(
				cmd.Flag(chainIdFlag).Value.String(),
				cmd.Flag(nodeAddrFlag).Value.String(),
				name,
			)
		},
	}
	flags := cmd.Flags()
	flags.String(chainIdFlag, "sourcehub-dev", "sets the chain-id")
	flags.String(nodeAddrFlag, nodeAddrDefault, "sets the address of the node to communicate with")

	cmd.Execute()
}

func runDemo(chainId string, nodeAddr string, validatorKeyName string) {
	ctx := context.Background()

	client, err := sdk.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	txBuilder, err := sdk.NewTxBuilder(
		sdk.WithSDKClient(client),
		sdk.WithChainID(chainId),
	)
	if err != nil {
		log.Fatal(err)
	}

	txSigner := getSigner(validatorKeyName)

	log.Printf("Creating Policy: %v", policy)
	policy := createPolicy(ctx, client, &txBuilder, txSigner)

	log.Printf("Registering object: file:readme")
	record := registerObject(ctx, client, &txBuilder, txSigner, policy.Id)

	log.Printf("Evaluating Access Request to read file:readme")
	decision := checkAccess(ctx, client, &txBuilder, txSigner, policy.Id, record.Metadata.OwnerDid, []*coretypes.Operation{
		{
			Object:     coretypes.NewObject("file", "readme"),
			Permission: "read",
		},
	})

	log.Printf("Issueing ticket for Access Decision %v", decision.Id)
	abciService, err := access_ticket.NewABCIService(nodeAddr)
	if err != nil {
		log.Fatalf("could not create abci service: %v", err)
	}
	issuer := access_ticket.NewTicketIssuer(&abciService)
	ticket, err := issuer.Issue(ctx, decision.Id, txSigner.GetPrivateKey())
	if err != nil {
		log.Fatalf("could not issue ticket: %v", err)
	}
	log.Printf("Access Ticket issued: %v", ticket)

	log.Printf("Waiting for next block")
	time.Sleep(time.Second * 5)

	log.Printf("Verifying Access Ticket")
	recoveredTicket, err := access_ticket.UnmarshalAndVerify(ctx, nodeAddr, ticket)
	if err != nil {
		log.Fatalf("could not verify ticket: %v", err)
	}

	// remove some fields for cleaner print
	recoveredTicket.Signature = nil
	recoveredTicket.DecisionProof = nil
	log.Printf("Acces Ticket verified: %v", spew.Sdump(recoveredTicket))
}

func getSigner(accAddr string) sdk.TxSigner {
	reg := cdctypes.NewInterfaceRegistry()
	cryptocdc.RegisterInterfaces(reg)
	cdc := codec.NewProtoCodec(reg)
	keyring, err := keyring.New("sourcehub", keyring.BackendTest, "~/.sourcehub", nil, cdc)
	if err != nil {
		log.Fatalf("could not load keyring: %v", err)
	}

	txSigner, err := sdk.NewTxSignerFromKeyringKey(keyring, accAddr)
	if err != nil {
		log.Fatalf("could not load keyring: %v", err)
	}
	return txSigner
}

func createPolicy(ctx context.Context, client *sdk.Client, txBuilder *sdk.TxBuilder, txSigner sdk.TxSigner) *coretypes.Policy {
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
	return policyResponse.Record.Policy
}

func registerObject(ctx context.Context, client *sdk.Client, txBuilder *sdk.TxBuilder, txSigner sdk.TxSigner, policyId string) *acptypes.RelationshipRecord {
	msgSet := sdk.MsgSet{}
	mapper := msgSet.WithDirectPolicyCmd(
		acptypes.NewMsgDirectPolicyCmd(
			txSigner.GetAccAddress(),
			policyId,
			acptypes.NewRegisterObjectCmd(coretypes.NewObject("file", "readme")),
		),
	)
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

	response, err := mapper.Map(result.TxPayload())
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("object registered: %v", response.Result.GetRegisterObjectResult())
	return response.Result.GetRegisterObjectResult().Record
}

func checkAccess(ctx context.Context, client *sdk.Client, txBuilder *sdk.TxBuilder, txSigner sdk.TxSigner, policyId string, actorId string, operations []*coretypes.Operation) *acptypes.AccessDecision {
	msgSet := sdk.MsgSet{}
	mapper := msgSet.WithCheckAccess(
		acptypes.NewMsgCheckAccess(txSigner.GetAccAddress(), policyId, &coretypes.AccessRequest{
			Operations: operations,
			Actor:      &coretypes.Actor{Id: actorId},
		}),
	)
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

	response, err := mapper.Map(result.TxPayload())
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("acces request evaluated: %v", response.Decision)
	return response.Decision
}
