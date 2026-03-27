package bearer_token

import (
	"crypto"
	"encoding/json"
	"time"

	"github.com/go-jose/go-jose/v3"
	"github.com/go-jose/go-jose/v3/cryptosigner"
)

const (
	IssuedAtClaim = "iat"
	ExpiresClaim  = "exp"
	IssuerClaim   = "iss"
	// AuthorizedAccountClaim is the name of the expected field in the JWS
	// which authorizes a SourceHub account to produce Txs on behalf of the
	// token issuer
	AuthorizedAccountClaim = "authorized_account"
)

const DefaultExpirationTime = time.Minute * 15

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

// ToJWS serializes the Token payload, signs it and marshals the result
// to a JWS using the compact serialization mode
func (t *BearerToken) ToJWS(signer crypto.Signer) (string, error) {
	bytes, err := json.Marshal(t)
	if err != nil {
		return "", err
	}
	payload := string(bytes)

	opaque := cryptosigner.Opaque(signer)
	key := jose.SigningKey{
		Algorithm: opaque.Algs()[0],
		Key:       opaque,
	}
	var opts *jose.SignerOptions
	joseSigner, err := jose.NewSigner(key, opts)
	if err != nil {
		return "", err
	}

	obj, err := joseSigner.Sign([]byte(payload))
	if err != nil {
		return "", err
	}

	return obj.CompactSerialize()
}

// NewBearerTokenNow issues a BearerToken using the current time and the default expiration delta
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

// NewBearerTokenFromTime constructs a BearerToken from timestamps
func NewBearerTokenFromTime(actorID string, authorizedAcc string, issuedAt time.Time, expires time.Time) BearerToken {
	return BearerToken{
		IssuerID:          actorID,
		AuthorizedAccount: authorizedAcc,
		IssuedTime:        issuedAt.Unix(),
		ExpirationTime:    expires.Unix(),
	}
}
