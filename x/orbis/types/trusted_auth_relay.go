package types

import (
	"strings"

	errorsmod "cosmossdk.io/errors"
	"github.com/TBD54566975/ssi-sdk/crypto"
	"github.com/TBD54566975/ssi-sdk/did/key"
)

// ValidateTrustedAuthRelayConfig validates a ring's immutable relay settings.
func ValidateTrustedAuthRelayConfig(allow bool, relayDIDs []string) error {
	if !allow && len(relayDIDs) != 0 {
		return errorsmod.Wrap(ErrInvalidRing, "trusted_auth_relay_dids require allow_trusted_auth_relays")
	}
	return ValidateTrustedAuthRelayDIDs(relayDIDs)
}

// ValidateTrustedAuthRelayDIDs validates a unique set of Ed25519 did:key issuers.
func ValidateTrustedAuthRelayDIDs(relayDIDs []string) error {
	seen := make(map[string]struct{}, len(relayDIDs))
	for _, relayDID := range relayDIDs {
		if relayDID == "" || strings.TrimSpace(relayDID) != relayDID {
			return errorsmod.Wrap(ErrInvalidRing, "trusted auth relay DID must be non-empty without surrounding whitespace")
		}
		_, _, keyType, err := key.DIDKey(relayDID).Decode()
		if err != nil {
			return errorsmod.Wrapf(ErrInvalidRing, "invalid trusted auth relay DID %q: %v", relayDID, err)
		}
		if keyType != crypto.Ed25519 {
			return errorsmod.Wrapf(ErrInvalidRing, "trusted auth relay DID %q must use Ed25519", relayDID)
		}
		if _, exists := seen[relayDID]; exists {
			return errorsmod.Wrapf(ErrInvalidRing, "duplicate trusted auth relay DID %q", relayDID)
		}
		seen[relayDID] = struct{}{}
	}
	return nil
}
