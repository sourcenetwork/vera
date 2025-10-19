package signed_policy_cmd

import (
	"errors"
)

var (
	ErrExpirationDeltaTooLarge = errors.New("expiration delta greater than threshold")
	ErrCommandExpired          = errors.New("PolicyCmdPayload expiration height is stale")
	ErrBuilderMissingArgument  = errors.New("missing argument")
	ErrSignerRequired          = errors.New("no signer set for builder")
	ErrPayloadAlreadyProcessed = errors.New("PolicyCmdPayload already processed")
)
