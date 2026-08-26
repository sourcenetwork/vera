package bearer_token

import (
	"fmt"
	"time"

	veratypes "github.com/sourcenetwork/vera/types"
	"github.com/sourcenetwork/vera/x/acp/did"
)

func validateBearerTokenValues(token *BearerToken) error {
	if did.IsValidDID(token.IssuerID) != nil {
		return ErrInvalidIssuer
	}

	if err := veratypes.IsValidVeraAddr(token.AuthorizedAccount); err != nil {
		return fmt.Errorf("%v: %w", err, ErrInvalidAuhtorizedAccount)
	}

	if token.ExpirationTime < token.IssuedTime {
		return fmt.Errorf("issue time cannot be after expiration time: %w", ErrInvalidBearerToken)
	}

	return nil
}

func validateBearerToken(token *BearerToken, currentTime *time.Time) error {
	err := validateBearerTokenValues(token)
	if err != nil {
		return err
	}

	now := currentTime.Unix()

	if now > token.ExpirationTime {
		return newErrTokenExpired(token.ExpirationTime, now)
	}

	return nil
}
