// package bearer_token defines an authentication method for Policy operations using a self issued JWS
package bearer_token

import (
	"context"
	"fmt"
	"time"

	"github.com/sourcenetwork/sourcehub/x/acp/did"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

// AuthorizeMsg verifies whether the given MsgBearerPolicyCmd should be authorized
// to execute.
// If the validation is successful, returns the authenticated Actor DID or an error
//
// Note: an Authorized Msg means that the msg's Bearer token is well formed and valid (actor authentication)
// and that the token is bound to the account that signed the Tx (msg authorization).
func AuthorizeMsg(ctx context.Context, resolver did.Resolver, msg *types.MsgBearerPolicyCmd, currentTime time.Time) (string, error) {
	token, err := parseValidateJWS(ctx, resolver, msg.BearerToken)
	if err != nil {
		return "", err
	}

	err = validateBearerToken(&token, &currentTime)
	if err != nil {
		return "", err
	}

	if msg.Creator != token.AuthorizedAccount {
		return "", fmt.Errorf("msg creator %v: expected creator %v: %w", msg.Creator, token.AuthorizedAccount, ErrMsgUnauthorized)
	}

	return token.IssuerID, nil
}
