package types

import "time"

// JWT claim constants
const (
	IssuedAtClaim          = "iat"
	ExpiresClaim           = "exp"
	IssuerClaim            = "iss"
	AuthorizedAccountClaim = "authorized_account"
)

// DefaultExpirationTime is the default expiration time for bearer tokens
const DefaultExpirationTime = time.Minute * 10

// BearerToken contains the structured fields included in the JWS Bearer Token.
// This type is used for DID-based authorization via JWS extension options.
type BearerToken struct {
	// IssuerID is the Actor ID (DID) for the Token signer
	IssuerID string `json:"iss,omitempty"`
	// AuthorizedAccount is the SourceHub account address which is allowed to use this token
	AuthorizedAccount string `json:"authorized_account,omitempty"`
	// IssuedTime is the timestamp at which the token was generated
	IssuedTime int64 `json:"iat,omitempty"`
	// ExpirationTime is the timestamp at which the token will expire
	ExpirationTime int64 `json:"exp,omitempty"`
}

// RequiredClaims returns the list of required claims for JWS payload validation
func RequiredClaims() []string {
	return []string{
		IssuedAtClaim,
		IssuerClaim,
		AuthorizedAccountClaim,
		ExpiresClaim,
	}
}
