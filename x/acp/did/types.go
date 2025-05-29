package did

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/TBD54566975/ssi-sdk/crypto"
	"github.com/TBD54566975/ssi-sdk/did"
	"github.com/TBD54566975/ssi-sdk/did/key"
	"github.com/TBD54566975/ssi-sdk/did/resolution"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256r1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
)

type DIDDocument struct {
	*did.Document
}

type Resolver interface {
	Resolve(ctx context.Context, did string) (DIDDocument, error)
}

// IsValidDID verifies the validity of the specified did string.
func IsValidDID(didStr string) error {
	_, err := resolution.GetMethodForDID(didStr)
	if err != nil {
		return err
	}
	return nil
}

// IssueDID produces a DID for a SourceHub account.
func IssueDID(acc sdk.AccountI) (string, error) {
	return DIDFromPubKey(acc.GetPubKey())
}

// DIDFromPubKey constructs and returns a DID from a public key.
func DIDFromPubKey(pk cryptotypes.PubKey) (string, error) {
	var keyType crypto.KeyType
	switch t := pk.(type) {
	case *secp256k1.PubKey:
		keyType = crypto.SECP256k1
	case *secp256r1.PubKey:
		keyType = crypto.P256
	case *ed25519.PubKey:
		keyType = crypto.Ed25519
	default:
		return "", fmt.Errorf(
			"failed to issue did for key %v: account key type must be secp256k1, secp256r1, or ed25519, got %v",
			pk.Bytes(), t,
		)
	}

	pkBytes := pk.Bytes()

	if keyType == crypto.P256 {
		sdkPubKey := pk.(*secp256r1.PubKey)
		ecdsaPubKey := &ecdsa.PublicKey{Curve: elliptic.P256(), X: sdkPubKey.Key.X, Y: sdkPubKey.Key.Y}
		ecdhPubKey, err := ecdsaPubKey.ECDH()
		if err != nil {
			return "", fmt.Errorf("failed to create ecdh public key: %w", err)
		}
		pkBytes = ecdhPubKey.Bytes()
	}

	did, err := key.CreateDIDKey(keyType, pkBytes)
	if err != nil {
		return "", fmt.Errorf("failed to generated did for %v: %v", pk.Bytes(), err)
	}

	return did.String(), nil
}

// IssueModuleDID produces a DID for a SourceHub module, based on its name.
//
// The issued did uses a pseudo-method named "module", which simply appends the module name.
func IssueModuleDID(moduleName string) string {
	return "did:module:" + moduleName
}
