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

	"github.com/sourcenetwork/sourcehub/x/acp/did"
)

// JWT claim constants
const (
	IssuedAtClaim          = "iat"
	ExpiresClaim           = "exp"
	IssuerClaim            = "iss"
	AuthorizedAccountClaim = "authorized_account"
)

// DefaultExpirationTime is the default expiration time for bearer tokens
const DefaultExpirationTime = time.Minute * 10

// Required claims for JWS payload validation
var requiredClaims = []string{
	IssuedAtClaim,
	IssuerClaim,
	AuthorizedAccountClaim,
	ExpiresClaim,
}

// BearerToken contains the structured fields included in the JWS Bearer Token
type BearerToken struct {
	// IssuerID is the Actor ID for the Token signer
	IssuerID string `json:"iss,omitempty"`
	// AuthorizedAccount is the SourceHub account address which is allowed to use this token
	AuthorizedAccount string `json:"authorized_account,omitempty"`
	// IssuedTime is the timestamp at which the token was generated
	IssuedTime int64 `json:"iat,omitempty"`
	// ExpirationTime is the timestamp at which the token will expire
	ExpirationTime int64 `json:"exp,omitempty"`
}

// parseValidateJWS processes a JWS Bearer token by unmarshaling it and verifying its signature.
func parseValidateJWS(ctx context.Context, resolver did.Resolver, bearerJWS string) (BearerToken, error) {
	bearerJWS = strings.TrimLeft(bearerJWS, " \n\t\r")
	if strings.HasPrefix(bearerJWS, "{") {
		return BearerToken{}, fmt.Errorf("JSON serialization is not supported for security reasons")
	}

	jws, err := jose.ParseSigned(bearerJWS)
	if err != nil {
		return BearerToken{}, fmt.Errorf("failed parsing jws: %v", err)
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

	// Verify signature against the issuer DID
	did := bearer.IssuerID
	didKey := key.DIDKey(did)
	pubBytes, _, keytype, err := didKey.Decode()
	if err != nil {
		return BearerToken{}, fmt.Errorf("failed to resolve actor did: %v", err)
	}

	pubKey, err := crypto.BytesToPubKey(pubBytes, keytype)
	if err != nil {
		return BearerToken{}, fmt.Errorf("failed to retrieve pub key: %v", err)
	}

	var algs []jwa.SignatureAlgorithm
	if secpKey, ok := pubKey.(secp.PublicKey); ok {
		// https://www.rfc-editor.org/rfc/rfc8812
		algs = []jwa.SignatureAlgorithm{jwa.ES256K}
		pubKey = secpKey.ToECDSA()
	} else {
		algs, err = jwxjws.AlgorithmsForKey(pubKey)
		if err != nil {
			return BearerToken{}, fmt.Errorf("failed to retrieve algs for pub key: %v", err)
		}
	}

	_, err = jwxjws.Verify([]byte(bearerJWS), jwxjws.WithKey(algs[0], pubKey))
	if err != nil {
		return BearerToken{}, fmt.Errorf("could not verify actor signature for jwk: %v", err)
	}

	return bearer, nil
}

// unmarshalJWSPayload unmarshals the JWS bytes into a BearerToken.
func unmarshalJWSPayload(payload []byte) (BearerToken, error) {
	obj := make(map[string]any)
	err := json.Unmarshal(payload, &obj)
	if err != nil {
		return BearerToken{}, err
	}

	for _, claim := range requiredClaims {
		_, ok := obj[claim]
		if !ok {
			return BearerToken{}, fmt.Errorf("missing required claim: %s", claim)
		}
	}

	token := BearerToken{}
	err = json.Unmarshal(payload, &token)
	if err != nil {
		return BearerToken{}, fmt.Errorf("could not unmarshal payload: %v", err)
	}
	return token, nil
}

// validateBearerTokenValues validates the bearer token values
func validateBearerTokenValues(token *BearerToken) error {
	if err := did.IsValidDID(token.IssuerID); err != nil {
		return fmt.Errorf("invalid issuer DID: %v", err)
	}

	if err := isValidSourceHubAddr(token.AuthorizedAccount); err != nil {
		return fmt.Errorf("invalid authorized account: %v", err)
	}

	if token.ExpirationTime < token.IssuedTime {
		return fmt.Errorf("issue time cannot be after expiration time")
	}

	return nil
}

// validateBearerToken validates the bearer token including timing
func validateBearerToken(token *BearerToken, currentTime *time.Time) error {
	err := validateBearerTokenValues(token)
	if err != nil {
		return err
	}

	now := currentTime.Unix()

	if now > token.ExpirationTime {
		return fmt.Errorf("token expired: current time %d > expiration time %d", token.ExpirationTime, now)
	}

	return nil
}

// isValidSourceHubAddr validates a SourceHub address format
func isValidSourceHubAddr(addr string) error {
	if len(addr) == 0 {
		return fmt.Errorf("address cannot be empty")
	}
	// Accept both "source" and "cosmos" prefixes for compatibility with different environments
	if !strings.HasPrefix(addr, "source") && !strings.HasPrefix(addr, "cosmos") {
		return fmt.Errorf("invalid SourceHub address format: %s", addr)
	}
	return nil
}

// validateJWSExtension parses and validates JWS extension option bearer token.
// Returns both the issuer DID and the authorized account.
func validateJWSExtension(ctx context.Context, bearerToken string, currentTime time.Time) (string, string, error) {
	resolver := &did.KeyResolver{}

	token, err := parseValidateJWS(ctx, resolver, bearerToken)
	if err != nil {
		return "", "", err
	}

	err = validateBearerToken(&token, &currentTime)
	if err != nil {
		return "", "", err
	}

	return token.IssuerID, token.AuthorizedAccount, nil
}

// GetExtractedDIDFromContext retrieves the extracted DID from context.
func GetExtractedDIDFromContext(ctx sdk.Context) string {
	if did, ok := ctx.Value(ExtractedDIDContextKey).(string); ok {
		return did
	}
	return ""
}

// NewBearerTokenNow creates a new BearerToken with current time and default expiration
func NewBearerTokenNow(actorID string, authorizedAccount string) BearerToken {
	now := time.Now()
	expires := now.Add(DefaultExpirationTime)

	return BearerToken{
		IssuerID:          actorID,
		AuthorizedAccount: authorizedAccount,
		IssuedTime:        now.Unix(),
		ExpirationTime:    expires.Unix(),
	}
}

// CreateJWSHeader creates the standard JWS header for EdDSA JWT tokens
func CreateJWSHeader() map[string]any {
	return map[string]any{
		"alg": "EdDSA",
		"typ": "JWT",
	}
}
