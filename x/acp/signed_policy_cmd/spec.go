package signed_policy_cmd

import (
	"context"
	"fmt"

	"github.com/sourcenetwork/acp_core/pkg/errors"
	"github.com/sourcenetwork/sourcehub/x/acp/did"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

// payloadSpec executes validation against a PolicyCmdPayload to ensure it should be accepted
func payloadSpec(params types.Params, currentHeight uint64, payload *types.SignedPolicyCmdPayload) error {
	if payload.ExpirationDelta > params.PolicyCommandMaxExpirationDelta {
		return fmt.Errorf("%w: max %v, given %v", ErrExpirationDeltaTooLarge, params.PolicyCommandMaxExpirationDelta, payload.ExpirationDelta)
	}

	maxHeight := payload.IssuedHeight + payload.ExpirationDelta
	if currentHeight > maxHeight {
		return fmt.Errorf("%v: current %v limit %v", ErrCommandExpired, currentHeight, maxHeight)
	}

	return nil
}

// ValidateAndExtractCmd validates a MsgPolicyCmd and return the Cmd payload
func ValidateAndExtractCmd(ctx context.Context, params types.Params, resolver did.Resolver, payload string, contentType types.MsgSignedPolicyCmd_ContentType, currentHeight uint64) (*types.SignedPolicyCmdPayload, error) {
	var cmd *types.SignedPolicyCmdPayload
	var err error

	switch contentType {
	case types.MsgSignedPolicyCmd_JWS:
		verifier := newJWSVerifier(resolver)
		cmd, err = verifier.Verify(ctx, payload)
	default:
		err = fmt.Errorf("invalid signed command: cmd %v: %w", payload, errors.ErrUnknownVariant)
	}

	if err != nil {
		return nil, fmt.Errorf("invalid signed command: %w", err)
	}

	err = payloadSpec(params, currentHeight, cmd)
	if err != nil {
		return nil, fmt.Errorf("invalid payload: %v", err)
	}

	return cmd, nil
}
