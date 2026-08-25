package bearer_token

import (
	"context"
	"crypto"
	"testing"
	"time"

	acpdid "github.com/sourcenetwork/vera/x/acp/did"
	"github.com/sourcenetwork/vera/x/acp/types"
	"github.com/stretchr/testify/require"
)

const validVeraAddr = "source19djduggm345yf2dn0y0jqqgkr5q0pt234dkyvd"
const msgCreator = validVeraAddr

// bearerValidationTestVector models a test case which generates a JWS from the BearerToken definition
// and verifies whether the Token is valid or not
type bearerValidationTestVector struct {
	Description   string
	Token         BearerToken
	ExpectedError error
	Signer        crypto.Signer
	ServerTime    time.Time
}

func Test_AuthorizeMsg_Errors(t *testing.T) {
	validDID, signer, err := acpdid.ProduceDID()
	require.NoError(t, err)

	jwsTestVectors := []bearerValidationTestVector{
		{
			Description: "payload without issuer rejected",
			Token: BearerToken{
				IssuerID:          "",
				ExpirationTime:    mustUnixTime("2024-06-17 14:20:00"),
				IssuedTime:        mustUnixTime("2024-06-17 14:00:00"),
				AuthorizedAccount: validVeraAddr,
			},
			ExpectedError: ErrMissingClaim,
			Signer:        signer,
			ServerTime:    mustTime("2024-06-17 14:10:00"),
		},
		{
			Description: "payload without authorization_account rejected",
			Token: BearerToken{
				IssuerID:          validDID,
				ExpirationTime:    mustUnixTime("2024-06-17 14:20:00"),
				IssuedTime:        mustUnixTime("2024-06-17 14:00:00"),
				AuthorizedAccount: "",
			},
			ExpectedError: ErrMissingClaim,
			Signer:        signer,
			ServerTime:    mustTime("2024-06-17 14:10:00"),
		},
		{
			Description: "payload without isa rejected",
			Token: BearerToken{
				IssuerID:          validDID,
				ExpirationTime:    mustUnixTime("2024-06-17 14:20:00"),
				IssuedTime:        0,
				AuthorizedAccount: validVeraAddr,
			},
			ExpectedError: ErrMissingClaim,
			Signer:        signer,
			ServerTime:    mustTime("2024-06-17 14:10:00"),
		},
		{
			Description: "payload without exp rejected",
			Token: BearerToken{
				IssuerID:          validDID,
				ExpirationTime:    0,
				IssuedTime:        mustUnixTime("2024-06-17 14:00:00"),
				AuthorizedAccount: validVeraAddr,
			},
			ExpectedError: ErrMissingClaim,
			Signer:        signer,
			ServerTime:    mustTime("2024-06-17 14:10:00"),
		},
		{
			Description: "iss as invalid did rejected",
			Token: BearerToken{
				IssuerID:          "did:invalid",
				ExpirationTime:    mustUnixTime("2024-06-17 14:20:00"),
				IssuedTime:        mustUnixTime("2024-06-17 14:00:00"),
				AuthorizedAccount: validVeraAddr,
			},
			ExpectedError: ErrInvalidIssuer,
			Signer:        signer,
			ServerTime:    mustTime("2024-06-17 14:10:00"),
		},
		{
			Description: "authorization_account invalid Vera addr rejected",
			Token: BearerToken{
				IssuerID:          validDID,
				ExpirationTime:    mustUnixTime("2024-06-17 14:20:00"),
				IssuedTime:        mustUnixTime("2024-06-17 14:00:00"),
				AuthorizedAccount: "notsource123456z",
			},
			ExpectedError: ErrInvalidAuhtorizedAccount,
			Signer:        signer,
			ServerTime:    mustTime("2024-06-17 14:10:00"),
		},
		{
			Description: "expired token rejected",
			Token: BearerToken{
				IssuerID:          validDID,
				IssuedTime:        mustUnixTime("2024-06-17 14:00:00"),
				ExpirationTime:    mustUnixTime("2024-06-17 14:20:00"),
				AuthorizedAccount: validVeraAddr,
			},
			ExpectedError: ErrTokenExpired,
			Signer:        signer,
			ServerTime:    mustTime("2024-06-17 14:30:00"),
		},
		{
			Description: "tx signer different from authorized_account",
			Token: BearerToken{
				IssuerID:          validDID,
				IssuedTime:        mustUnixTime("2024-06-17 14:00:00"),
				ExpirationTime:    mustUnixTime("2024-06-17 14:20:00"),
				AuthorizedAccount: "source1dsc8ah9rytxrzhq0suj994anwuhvq7yjeh67dp",
			},
			ExpectedError: ErrMsgUnauthorized,
			Signer:        signer,
			ServerTime:    mustTime("2024-06-17 14:10:00"),
		},
	}

	for _, test := range jwsTestVectors {
		t.Run(test.Description, func(t *testing.T) {
			jws, err := test.Token.ToJWS(test.Signer)
			require.NoError(t, err)
			testAuthorizeMsg(t, jws, test.ServerTime, test.ExpectedError)
		})
	}
}

func Test_JWS_InvalidSignatureIsRejected(t *testing.T) {
	jws := "eyJhbGciOiJFZERTQSJ9.eyJpc3MiOiJkaWQ6a2V5Ono2TWt0VHdhdFZtQ3d6YzJuYlpiTUt5YmNpdGlMaXFVZ21Kb2FOWTdmdm9YQzVpRCIsImF1dGhvcml6ZWRfYWNjb3VudCI6InNvdXJjZTE5ZGpkdWdnbTM0NXlmMmRuMHkwanFxZ2tyNXEwcHQyMzRka3l2ZCIsImlhdCI6MTcxODYzMjgwMCwiZXhwIjoxNzE4NjM0MDAwfQ.cIis7b8ctEctoSUrxvk5_X2jUC9-nmNCey2D0d1NrbtWKaPSnahJrj54CaLLCEiogc_NkqTjazrvF_Kp1y1-BQ"

	testAuthorizeMsg(t, jws, mustTime("2024-06-17 14:10:00"), ErrInvalidBearerToken)
}

func Test_JWS_WithTamperedHeaderIsRejected(t *testing.T) {
	// Original JWS:
	// eyJhbGciOiJFZERTQSJ9.eyJpc3MiOiJkaWQ6a2V5Ono2TWtnM1lnM2I2TU5zeU10aXJyZlh2NUdaM2p1TTJtNkhiVTZWbnN4ZjFtcEVleSIsImF1dGhvcml6ZWRfYWNjb3VudCI6InNvdXJjZTE5ZGpkdWdnbTM0NXlmMmRuMHkwanFxZ2tyNXEwcHQyMzRka3l2ZCIsImlhdCI6MTcxODYzMjgwMCwiZXhwIjoxNzE4NjM0MDAwfQ.X2T5KpLjH1IGDCxYJ5Hp2CvScwWjLwqtleEHg0PZiYDCpIWh-tgxzfEFPHoHyYjnvcHS8FQk4arlJQaJuW3IBA

	jws := "eyJhbbciOiJFZERTQSJ9.eyJpc3MiOiJkaWQ6a2V5Ono2TWtnM1lnM2I2TU5zeU10aXJyZlh2NUdaM2p1TTJtNkhiVTZWbnN4ZjFtcEVleSIsImF1dGhvcml6ZWRfYWNjb3VudCI6InNvdXJjZTE5ZGpkdWdnbTM0NXlmMmRuMHkwanFxZ2tyNXEwcHQyMzRka3l2ZCIsImlhdCI6MTcxODYzMjgwMCwiZXhwIjoxNzE4NjM0MDAwfQ.X2T5KpLjH1IGDCxYJ5Hp2CvScwWjLwqtleEHg0PZiYDCpIWh-tgxzfEFPHoHyYjnvcHS8FQk4arlJQaJuW3IBA"

	testAuthorizeMsg(t, jws, mustTime("2024-06-17 14:10:00"), ErrInvalidBearerToken)
}

func Test_JWS_JSONSerializationIsRejected(t *testing.T) {
	jsonSerializedToken := `
	{
		"payload":"eyJpc3MiOiJkaWQ6a2V5Ono2TWtxUTNhY2J0NWdVaHV1QnFSZUtTanpEcWNIZlREa0hMb3dhWFY2YURIREw4RCIsImF1dGhvcml6ZWRfYWNjb3VudCI6InNvdXJjZTE5ZGpkdWdnbTM0NXlmMmRuMHkwanFxZ2tyNXEwcHQyMzRka3l2ZCIsImlhdCI6MTcxODcyNzk4NCwiZXhwIjoxNzE4NzI4ODg0fQ",
		"protected":"eyJhbGciOiJFZERTQSJ9",
		"signature":"DuwXmY3pRMSIejK7mK4lSEzrMCP4PhI7etLncuRlGI3QhRjrbcWaQnEC3fcziAsQZ1cLgtCiffgX9zCTSg8iBg"
	}`

	testAuthorizeMsg(t, jsonSerializedToken, mustTime("2024-06-17 14:10:00"), ErrJSONSerializationUnsupported)
}

func testAuthorizeMsg(t *testing.T, jws string, serverTime time.Time, expectedErr error) {
	ctx := context.TODO()
	resolver := acpdid.KeyResolver{}
	msg := types.MsgBearerPolicyCmd{
		Creator:     msgCreator,
		BearerToken: jws,
	}

	actorId, err := AuthorizeMsg(ctx, &resolver, &msg, serverTime)

	require.Empty(t, actorId)
	require.ErrorIs(t, err, expectedErr)
}

func mustUnixTime(ts string) int64 {
	return mustTime(ts).Unix()
}

func mustTime(ts string) time.Time {
	t, err := time.Parse(time.DateTime, ts)
	if err != nil {
		panic(err)
	}
	return t
}
