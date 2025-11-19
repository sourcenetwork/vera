package main

import (
	"crypto/ed25519"
	"fmt"
	"log"

	"github.com/TBD54566975/ssi-sdk/crypto"
	"github.com/TBD54566975/ssi-sdk/did/key"
	"github.com/spf13/cobra"

	"github.com/sourcenetwork/sourcehub/x/acp/bearer_token"
)

func main() {
	cmd.Execute()
}

var cmd = cobra.Command{
	Use:   "bearer-token authorized_account",
	Short: "bearer-token generates a oneshot did / bearer token for the given account address",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		addr := args[0]

		pub, priv, err := ed25519.GenerateKey(nil)
		if err != nil {
			log.Fatalf("failed to generate key pair: %v", err)
		}

		did, err := key.CreateDIDKey(crypto.Ed25519, pub)
		if err != nil {
			log.Fatalf("failed to generate did: %v", err)
		}

		token := bearer_token.NewBearerTokenNow(did.String(), addr)
		jws, err := token.ToJWS(priv)
		if err != nil {
			log.Fatalf("failed to issue token: %v", err)
		}

		log.Printf("token raw: %v", token)
		fmt.Printf("Bearer Token: %v\n", jws)
		fmt.Printf("DID: %v\n", did.String())
	},
}
