package ante

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/TBD54566975/ssi-sdk/crypto"
	"github.com/TBD54566975/ssi-sdk/did/key"
	sdk "github.com/cosmos/cosmos-sdk/types"
	secp "github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/go-jose/go-jose/v3"
	"github.com/lestrrat-go/jwx/v2/jwa"
	jwxjws "github.com/lestrrat-go/jwx/v2/jws"

	antetypes "github.com/sourcenetwork/vera/app/ante/types"
	appparams "github.com/sourcenetwork/vera/app/params"
	"github.com/sourcenetwork/vera/types"
	"github.com/sourcenetwork/vera/x/acp/did"
)

// parseValidateJWS processes a JWS Bearer token by unmarshaling it and verifying its signature.
func parseValidateJWS(ctx context.Context, resolver did.Resolver, bearerJWS string, trustedRelay bool) (antetypes.BearerToken, error) {
	bearerJWS = strings.TrimLeft(bearerJWS, " \n\t\r")
	if strings.HasPrefix(bearerJWS, "{") {
		return antetypes.BearerToken{}, fmt.Errorf("JSON serialization is not supported for security reasons")
	}

	jws, err := jose.ParseSigned(bearerJWS)
	if err != nil {
		return antetypes.BearerToken{}, fmt.Errorf("failed parsing jws: %v", err)
	}

	payloadBytes := jws.UnsafePayloadWithoutVerification()
	bearer, err := unmarshalJWSPayload(payloadBytes)
	if err != nil {
		return antetypes.BearerToken{}, err
	}

	err = validateBearerTokenValues(&bearer, trustedRelay)
	if err != nil {
		return antetypes.BearerToken{}, err
	}

	// Verify signature against the issuer DID
	did := bearer.IssuerID
	didKey := key.DIDKey(did)
	pubBytes, _, keytype, err := didKey.Decode()
	if err != nil {
		return antetypes.BearerToken{}, fmt.Errorf("failed to resolve actor did: %v", err)
	}

	pubKey, err := crypto.BytesToPubKey(pubBytes, keytype)
	if err != nil {
		return antetypes.BearerToken{}, fmt.Errorf("failed to retrieve pub key: %v", err)
	}

	var algs []jwa.SignatureAlgorithm
	if secpKey, ok := pubKey.(secp.PublicKey); ok {
		// https://www.rfc-editor.org/rfc/rfc8812
		algs = []jwa.SignatureAlgorithm{jwa.ES256K}
		pubKey = secpKey.ToECDSA()
	} else {
		algs, err = jwxjws.AlgorithmsForKey(pubKey)
		if err != nil {
			return antetypes.BearerToken{}, fmt.Errorf("failed to retrieve algs for pub key: %v", err)
		}
	}

	_, err = jwxjws.Verify([]byte(bearerJWS), jwxjws.WithKey(algs[0], pubKey))
	if err != nil {
		return antetypes.BearerToken{}, fmt.Errorf("could not verify actor signature for jwk: %v", err)
	}

	return bearer, nil
}

// unmarshalJWSPayload unmarshals the JWS bytes into a BearerToken.
func unmarshalJWSPayload(payload []byte) (antetypes.BearerToken, error) {
	obj := make(map[string]any)
	err := json.Unmarshal(payload, &obj)
	if err != nil {
		return antetypes.BearerToken{}, err
	}

	for _, claim := range antetypes.RequiredClaims() {
		_, ok := obj[claim]
		if !ok {
			return antetypes.BearerToken{}, fmt.Errorf("missing required claim: %s", claim)
		}
	}

	token := antetypes.BearerToken{}
	err = json.Unmarshal(payload, &token)
	if err != nil {
		return antetypes.BearerToken{}, fmt.Errorf("could not unmarshal payload: %v", err)
	}
	return token, nil
}

// validateProviderToken validates the provider token fields.
func validateProviderToken(token *antetypes.ProviderToken) error {
	if token.ProviderName == "" {
		return fmt.Errorf("provider token missing provider_name")
	}
	if token.UserID == "" {
		return fmt.Errorf("provider token missing user_id")
	}
	if token.ActorDID == "" {
		return fmt.Errorf("provider token missing actor_did")
	}
	return nil
}

// validateBearerTokenValues validates the bearer token values.
func validateBearerTokenValues(token *antetypes.BearerToken, trustedRelay bool) error {
	if strings.HasPrefix(token.IssuerID, "did:") {
		if err := did.IsValidDID(token.IssuerID); err != nil {
			return fmt.Errorf("invalid issuer DID: %v", err)
		}
	} else {
		var providerToken antetypes.ProviderToken
		if err := json.Unmarshal([]byte(token.IssuerID), &providerToken); err != nil {
			return fmt.Errorf("issuer is neither a valid DID nor a valid provider token: %v", err)
		}
	}

	if token.ProviderToken != "" && !trustedRelay {
		return fmt.Errorf("provider token requires a trusted relay")
	}

	// Trusted relays bind the worker account through the transaction fee grant.
	if !trustedRelay {
		if err := types.IsValidVeraAddr(token.AuthorizedAccount); err != nil {
			return fmt.Errorf("invalid authorized account: %v", err)
		}
	}

	if token.ExpirationTime < token.IssuedTime {
		return fmt.Errorf("issue time cannot be after expiration time")
	}

	return nil
}

// validateBearerToken validates the bearer token including timing
func validateBearerToken(token *antetypes.BearerToken, currentTime *time.Time, trustedRelay bool) error {
	err := validateBearerTokenValues(token, trustedRelay)
	if err != nil {
		return err
	}

	now := currentTime.Unix()

	if now > token.ExpirationTime {
		return fmt.Errorf("token expired: current time %d > expiration time %d", now, token.ExpirationTime)
	}

	return nil
}

// validateJWSExtension parses and validates JWS extension option bearer token.
// Returns the actor DID (from provider token if present, otherwise issuer DID) and the authorized account.
// Provider identities are accepted only from a trusted relay.
func validateJWSExtension(ctx context.Context, bearerToken string, currentTime time.Time, trustedRelay bool) (string, string, error) {
	resolver := &did.KeyResolver{}

	token, err := parseValidateJWS(ctx, resolver, bearerToken, trustedRelay)
	if err != nil {
		return "", "", err
	}

	err = validateBearerToken(&token, &currentTime, trustedRelay)
	if err != nil {
		return "", "", err
	}

	actorDID := token.IssuerID

	if token.ProviderToken != "" {
		var providerToken antetypes.ProviderToken
		if err := json.Unmarshal([]byte(token.ProviderToken), &providerToken); err != nil {
			return "", "", fmt.Errorf("failed to unmarshal provider token: %v", err)
		}

		if err := validateProviderToken(&providerToken); err != nil {
			return "", "", err
		}

		actorDID = providerToken.ActorDID
	}

	return actorDID, token.AuthorizedAccount, nil
}

// getExtractedDIDFromContext retrieves the extracted DID from context.
func getExtractedDIDFromContext(ctx sdk.Context) string {
	if did, ok := ctx.Value(appparams.ExtractedDIDContextKey).(string); ok {
		return did
	}
	return ""
}

func getTrustedRelayFeeGranterFromContext(ctx sdk.Context) string {
	if address, ok := ctx.Value(appparams.TrustedRelayFeeGranterContextKey).(string); ok {
		return address
	}
	return ""
}
