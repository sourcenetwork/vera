package signed_policy_cmd

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/TBD54566975/ssi-sdk/crypto"
	"github.com/TBD54566975/ssi-sdk/did/key"
	"github.com/cosmos/gogoproto/jsonpb"
	secp "github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/go-jose/go-jose/v3"
	"github.com/lestrrat-go/jwx/v2/jwa"
	jwxjws "github.com/lestrrat-go/jwx/v2/jws"

	"github.com/sourcenetwork/sourcehub/x/acp/did"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

func newJWSVerifier(resolver did.Resolver) jwsVerifier {
	return jwsVerifier{
		resolver: resolver,
	}
}

// jwsVerifier verifies the Signature of a JWS which contains a PolicyCmd
type jwsVerifier struct {
	resolver did.Resolver
}

// Verify verifies the integrity of the JWS payload, returns the Payload if OK
//
// The verification extracts a VerificationMethod from the resolved Actor DID in the PolicyCmd.
// The JOSE header attributes are ignored and only the key derived from the Actor DID is accepted.
// This is done to assure no impersonation happens by thinkering the JOSE header in order to produce a valid
// JWS, signed by key different than that of the DID owner.
func (s *jwsVerifier) Verify(ctx context.Context, jwsStr string) (*types.SignedPolicyCmdPayload, error) {
	jws, err := jose.ParseSigned(jwsStr)
	if err != nil {
		return nil, fmt.Errorf("failed parsing jws: %v", err)
	}

	payloadBytes := jws.UnsafePayloadWithoutVerification()
	payload := &types.SignedPolicyCmdPayload{}
	err = jsonpb.UnmarshalString(string(payloadBytes), payload)
	if err != nil {
		return nil, fmt.Errorf("failed unmarshaling PolcyCmd payload: %v", err)
	}

	did := payload.Actor
	// currently the ssi-sdk key resolver does not support secp256k1
	// therefore we skip using the did pkg resolver and decode it directly,
	// as that does not error.
	didKey := key.DIDKey(did)
	pubBytes, _, keytype, err := didKey.Decode()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve actor did: %v", err)
	}

	pubKey, err := crypto.BytesToPubKey(pubBytes, keytype)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve pub key: %v", err)
	}

	var algs []jwa.SignatureAlgorithm
	if secpKey, ok := pubKey.(secp.PublicKey); ok {
		// https://www.rfc-editor.org/rfc/rfc8812
		algs = []jwa.SignatureAlgorithm{jwa.ES256K}
		pubKey = secpKey.ToECDSA()
	} else {
		algs, err = jwxjws.AlgorithmsForKey(pubKey)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve algs for pub key: %v", err)
		}
	}

	_, err = jwxjws.Verify([]byte(jwsStr), jwxjws.WithKey(algs[0], pubKey))
	if err != nil {
		return nil, fmt.Errorf("could not verify actor signature for jwk: %v", err)
	}

	return payload, nil
}

// ComputePayloadID hashes a JWS payload to produce an ID used for replay protection.
func ComputePayloadID(jwsPayload string) []byte {
	hasher := sha256.New()
	hasher.Write([]byte(jwsPayload))
	return hasher.Sum(nil)
}
