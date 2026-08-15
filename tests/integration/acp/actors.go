package test

import (
	stdcrypto "crypto"
	"crypto/ed25519"

	sdked25519 "github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	sdksecp256k1 "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdkcrypto "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	secp256k1 "github.com/decred/dcrd/dcrec/secp256k1/v4"

	"github.com/sourcenetwork/vera/app"
	"github.com/sourcenetwork/vera/x/acp/did"
)

// TestActor models a Vera actor in the test suite
// Each actor has a keypair and a DID
// The actor has a VeraAddr only if their keys are of type secp256k1
type TestActor struct {
	Name     string
	DID      string
	PubKey   sdkcrypto.PubKey
	PrivKey  sdkcrypto.PrivKey
	VeraAddr string
	Signer   stdcrypto.Signer
}

// MustNewED25519ActorFromName deterministically generates a Test Actor from a string name as seed
// The Actor carries a ed25519 key pair and has no Vera addr
func MustNewED25519ActorFromName(name string) *TestActor {
	privKey := sdked25519.GenPrivKeyFromSecret([]byte(name))

	didStr, err := did.DIDFromPubKey(privKey.PubKey())
	if err != nil {
		panic(err)
	}

	stdPriv := ed25519.PrivateKey(privKey.Bytes())

	return &TestActor{
		Name:     name,
		DID:      didStr,
		PubKey:   privKey.PubKey(),
		PrivKey:  privKey,
		VeraAddr: "",
		Signer:   stdPriv,
	}
}

// MustNewVeraActorFromName deterministically generates a Test Actor from a string name as seed
// The Actor carries a secp256k1 key pair and a Vera addr
func MustNewVeraActorFromName(name string) *TestActor {
	key := sdksecp256k1.GenPrivKeyFromSecret([]byte(name))
	addr, err := bech32.ConvertAndEncode(app.AccountAddressPrefix, key.PubKey().Address())
	if err != nil {
		panic(err)
	}
	didStr, err := did.DIDFromPubKey(key.PubKey())
	if err != nil {
		panic(err)
	}

	s256Priv := secp256k1.PrivKeyFromBytes(key.Key)
	stdPrivKey := s256Priv.ToECDSA()

	return &TestActor{
		Name:     name,
		DID:      didStr,
		PubKey:   key.PubKey(),
		PrivKey:  key,
		VeraAddr: addr,
		Signer:   stdPrivKey,
	}
}
