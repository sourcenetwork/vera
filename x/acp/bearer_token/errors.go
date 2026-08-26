package bearer_token

import (
	"fmt"
	"time"

	"github.com/sourcenetwork/acp_core/pkg/errors"
)

var ErrInvalidBearerToken = errors.Wrap("invalid bearer token", errors.ErrorType_BAD_INPUT)
var ErrMsgUnauthorized = errors.Wrap("bearer policy msg: authorized_account doesn't match", errors.ErrorType_UNAUTHORIZED)

var ErrInvalidIssuer = errors.Wrap("invalid issuer: expected did", ErrInvalidBearerToken)
var ErrInvalidAuhtorizedAccount = errors.Wrap("invalid authorized_account: expects Vera address", ErrInvalidBearerToken)
var ErrTokenExpired = errors.Wrap("token expired", ErrInvalidBearerToken)
var ErrMissingClaim = errors.Wrap("required claim not found", ErrInvalidBearerToken)
var ErrJSONSerializationUnsupported = errors.Wrap("JWS JSON Serialization not supported", ErrInvalidBearerToken)

func newErrTokenExpired(expiresUnix int64, nowUnix int64) error {
	expires := time.Unix(expiresUnix, 0).Format(time.DateTime)
	now := time.Unix(nowUnix, 0).Format(time.DateTime)

	return fmt.Errorf("expired %v: now %v: %w", expires, now, ErrTokenExpired)
}

func newErrMissingClaim(claim string) error {
	return fmt.Errorf("claim %v: %w", claim, ErrMissingClaim)
}
