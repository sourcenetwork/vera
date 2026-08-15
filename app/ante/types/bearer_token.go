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
	// ProviderToken contains provider information (provider name, user id, etc.) as a JWT string
	ProviderToken string `json:"provider_token,omitempty"`
	// AuthorizedAccount is the Vera account address which is allowed to use this token
	AuthorizedAccount string `json:"authorized_account,omitempty"`
	// IssuedTime is the timestamp at which the token was generated
	IssuedTime int64 `json:"iat,omitempty"`
	// ExpirationTime is the timestamp at which the token will expire
	ExpirationTime int64 `json:"exp,omitempty"`
}

// ProviderToken represents an authentication provider token.
type ProviderToken struct {
	// ProviderName is the name of the authentication provider (e.g., "google", "github")
	ProviderName string `json:"provider_name,omitempty"`
	// UserID is the unique user identifier from the provider
	UserID string `json:"user_id,omitempty"`
	// ActorDID is the actor/user DID derived from userID hash
	ActorDID string `json:"actor_did,omitempty"`
}

// RequiredClaims returns the list of required claims for JWS payload validation
func RequiredClaims() []string {
	return []string{
		IssuedAtClaim,
		IssuerClaim,
		// AuthorizedAccountClaim, // Authorized account claim is no longer required
		ExpiresClaim,
	}
}
