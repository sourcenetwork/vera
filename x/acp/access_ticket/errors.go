package access_ticket

import (
	"errors"
	"fmt"
)

var (
	// ErrExternal represents an error in an external system,
	// which means the validation procedure can be retried
	ErrExternal error = errors.New("external error")

	ErrInvalidInput     = errors.New("invalid input")
	ErrDecisionNotFound = fmt.Errorf("decision not found: %w", ErrInvalidInput)

	// ErrInvalidTicket flags that the provided AccessTicket is
	// invalid and will never be valid.
	ErrInvalidTicket   error = errors.New("invalid AccessTicket")
	ErrExpiredDecision       = fmt.Errorf("expired decision: %w", ErrInvalidTicket)
	ErrExpiredProof          = fmt.Errorf("expired proof: %w", ErrInvalidTicket)
	ErrExpiredTicket         = fmt.Errorf("expired ticket: %w", ErrInvalidTicket)

	ErrInvalidDecisionProof = fmt.Errorf("invalid DecisionProof bytes: %w", ErrInvalidTicket)
	ErrdDecisionProofDenied = fmt.Errorf("DecisionProof denied: %w", ErrInvalidTicket)
	ErrInvalidSignature     = fmt.Errorf("invalid signature for ticket: %w", ErrInvalidTicket)
	ErrDecisionTampered     = fmt.Errorf("AccessDecision fingerprint different from DecisionId: %w", ErrInvalidTicket)
)
