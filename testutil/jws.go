package test

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/TBD54566975/ssi-sdk/crypto"
	"github.com/TBD54566975/ssi-sdk/did/key"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/stretchr/testify/require"

	antetypes "github.com/sourcenetwork/sourcehub/app/ante/types"
)

// NewBearerTokenNow creates a new BearerToken with current time and default expiration
func NewBearerTokenNow(actorID string, authorizedAccount string) antetypes.BearerToken {
	now := time.Now()
	expires := now.Add(antetypes.DefaultExpirationTime)

	return antetypes.BearerToken{
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

// GenerateSignedJWSWithMatchingDID creates a JWS bearer token with specified authorized account.
// Returns the JWS string and the DID.
func GenerateSignedJWSWithMatchingDID(t *testing.T, authorizedAccount string) (string, string) {
	// Derive seed from mnemonic
	mnemonic := "near smoke great nasty alley food crush nurse rubber say danger search employ under gaze today alien eager risk letter drum relief sponsor current"
	seed, err := hd.Secp256k1.Derive()(mnemonic, "", "m/44'/118'/0'/0/0")
	require.NoError(t, err)

	// Generate Ed25519 key pair from the derived seed
	if len(seed) > 32 {
		seed = seed[:32]
	}
	privKey := ed25519.NewKeyFromSeed(seed)
	pubKey := privKey.Public().(ed25519.PublicKey)

	// Create DID from public key
	didKey, err := key.CreateDIDKey(crypto.Ed25519, pubKey)
	require.NoError(t, err)
	userDID := didKey.String()

	// Create bearer token with the matching DID
	bearerToken := NewBearerTokenNow(userDID, authorizedAccount)
	payloadBytes, err := json.Marshal(bearerToken)
	require.NoError(t, err)

	// Create and sign the JWS
	header := CreateJWSHeader()
	headerBytes, err := json.Marshal(header)
	require.NoError(t, err)

	headerEncoded := base64.RawURLEncoding.EncodeToString(headerBytes)
	payloadEncoded := base64.RawURLEncoding.EncodeToString(payloadBytes)

	signingInput := headerEncoded + "." + payloadEncoded
	signature := ed25519.Sign(privKey, []byte(signingInput))
	signatureEncoded := base64.RawURLEncoding.EncodeToString(signature)

	jws := signingInput + "." + signatureEncoded
	return jws, userDID
}
