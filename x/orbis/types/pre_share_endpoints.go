package types

import (
	"net/url"
	"strings"

	errorsmod "cosmossdk.io/errors"
)

const (
	// MaxPreShareEndpoints bounds how many Defra storage targets a ring can
	// configure, so the field can't be used to bloat chain state.
	MaxPreShareEndpoints = 8
	// MaxPreShareEndpointLen bounds a single endpoint's byte length.
	MaxPreShareEndpointLen = 256
)

// ValidatePreShareEndpoints validates a ring's optional Defra storage
// endpoints. Empty keeps the direct-response PRE path unchanged.
func ValidatePreShareEndpoints(endpoints []string) error {
	if len(endpoints) > MaxPreShareEndpoints {
		return errorsmod.Wrapf(ErrInvalidRing, "pre_share_endpoints exceeds maximum length %d", MaxPreShareEndpoints)
	}
	seen := make(map[string]struct{}, len(endpoints))
	for _, endpoint := range endpoints {
		if err := validatePreShareEndpoint(endpoint); err != nil {
			return err
		}
		if _, dup := seen[endpoint]; dup {
			return errorsmod.Wrapf(ErrInvalidRing, "duplicate pre_share_endpoint: %q", endpoint)
		}
		seen[endpoint] = struct{}{}
	}
	return nil
}

// validatePreShareEndpoint rejects anything but a bare http(s) base URL.
// Credentials never belong in ring metadata, which is public on-chain state
// — Defra auth stays in each node's local config instead.
func validatePreShareEndpoint(endpoint string) error {
	if endpoint == "" || strings.TrimSpace(endpoint) != endpoint {
		return errorsmod.Wrap(ErrInvalidRing, "pre_share_endpoint must be non-empty without surrounding whitespace")
	}
	if len(endpoint) > MaxPreShareEndpointLen {
		return errorsmod.Wrapf(ErrInvalidRing, "pre_share_endpoint exceeds maximum length %d", MaxPreShareEndpointLen)
	}
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return errorsmod.Wrapf(ErrInvalidRing, "invalid pre_share_endpoint %q: %v", endpoint, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errorsmod.Wrapf(ErrInvalidRing, "pre_share_endpoint %q must use http or https", endpoint)
	}
	if parsed.Host == "" {
		return errorsmod.Wrapf(ErrInvalidRing, "pre_share_endpoint %q is missing a host", endpoint)
	}
	if parsed.User != nil {
		return errorsmod.Wrapf(ErrInvalidRing, "pre_share_endpoint %q must not embed credentials", endpoint)
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return errorsmod.Wrapf(ErrInvalidRing, "pre_share_endpoint %q must not include a query or fragment", endpoint)
	}
	return nil
}
