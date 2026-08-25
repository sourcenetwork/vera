package bearer_token

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/TBD54566975/ssi-sdk/crypto"
	"github.com/TBD54566975/ssi-sdk/did/key"
	secp "github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/go-jose/go-jose/v3"
	"github.com/lestrrat-go/jwx/v2/jwa"
	jwxjws "github.com/lestrrat-go/jwx/v2/jws"

	"github.com/sourcenetwork/vera/x/acp/did"
)

var requiredClaims = []string{
	IssuedAtClaim,
	IssuerClaim,
	AuthorizedAccountClaim,
	ExpiresClaim,
}

// parseValidateJWS processes a JWS Bearer token by unmarshaling it and verifying its signature.
// Returns a BearerToken is the JWS is valid and the signature matches the IssuerID
//
// Note: the JWS must be compact serialized, JSON serialization will be rejected as a conservative
// security measure against unprotected header attacks.
func parseValidateJWS(ctx context.Context, resolver did.Resolver, bearerJWS string) (BearerToken, error) {
	bearerJWS = strings.TrimLeft(bearerJWS, " \n\t\r")
	if strings.HasPrefix(bearerJWS, "{") {
		return BearerToken{}, ErrJSONSerializationUnsupported
	}

	jws, err := jose.ParseSigned(bearerJWS)
	if err != nil {
		return BearerToken{}, fmt.Errorf("failed parsing jws: %v: %w", err, ErrInvalidBearerToken)
	}

	payloadBytes := jws.UnsafePayloadWithoutVerification()
	bearer, err := unmarshalJWSPayload(payloadBytes)
	if err != nil {
		return BearerToken{}, err
	}

	err = validateBearerTokenValues(&bearer)
	if err != nil {
		return BearerToken{}, err
	}

	// currently the ssi-sdk key resolver does not support secp256k1
	// therefore we skip using the did pkg resolver and decode it directly,
	// as that does not error.
	did := bearer.IssuerID
	didKey := key.DIDKey(did)
	pubBytes, _, keytype, err := didKey.Decode()
	if err != nil {
		return BearerToken{}, fmt.Errorf("failed to resolve actor did: %v: %w", err, ErrInvalidBearerToken)
	}

	pubKey, err := crypto.BytesToPubKey(pubBytes, keytype)
	if err != nil {
		return BearerToken{}, fmt.Errorf("failed to retrieve pub key: %v: %w", err, ErrInvalidBearerToken)
	}
	var algs []jwa.SignatureAlgorithm
	if secpKey, ok := pubKey.(secp.PublicKey); ok {
		// https://www.rfc-editor.org/rfc/rfc8812
		algs = []jwa.SignatureAlgorithm{jwa.ES256K}
		pubKey = secpKey.ToECDSA()
	} else {
		algs, err = jwxjws.AlgorithmsForKey(pubKey)
		if err != nil {
			return BearerToken{}, fmt.Errorf("failed to retrieve algs for pub key: %v: %w", err, ErrInvalidBearerToken)
		}
	}

	_, err = jwxjws.Verify([]byte(bearerJWS), jwxjws.WithKey(algs[0], pubKey))
	if err != nil {
		return BearerToken{}, fmt.Errorf("could not verify actor signature for jwk: %v: %w", err, ErrInvalidBearerToken)
	}

	return bearer, nil
}

// unmarshalJWSPayload unmarshals the JWS bytes into a BearerToken.
//
// The unmarshaling is strict, meaning that if the json object did not contain *all*
// required claims, it returns an error.
func unmarshalJWSPayload(payload []byte) (BearerToken, error) {
	obj := make(map[string]any)
	err := json.Unmarshal(payload, &obj)
	if err != nil {
		return BearerToken{}, err
	}

	for _, claim := range requiredClaims {
		_, ok := obj[claim]
		if !ok {
			return BearerToken{}, newErrMissingClaim(claim)
		}
	}

	token := BearerToken{}
	err = json.Unmarshal(payload, &token)
	if err != nil {
		return BearerToken{}, fmt.Errorf("could not unmarshal payload: %v: %w", err, ErrInvalidBearerToken)
	}
	return token, nil
}
