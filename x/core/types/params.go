package types

import (
	"fmt"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	shtypes "github.com/sourcenetwork/vera/types"
)

var _ paramtypes.ParamSet = (*Params)(nil)

// ParamKeyTable the param key table for launch module.
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance.
func NewParams() Params {
	return Params{}
}

// DefaultParams returns a default set of parameters.
func DefaultParams() Params {
	return NewParams()
}

// ParamSetPairs get the params.ParamSet.
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{}
}

// Validate validates the set of params.
func (p Params) Validate() error {
	seen := make(map[string]struct{}, len(p.TrustedRelayFeeGranters))
	for _, address := range p.TrustedRelayFeeGranters {
		if err := shtypes.IsValidVeraAddr(address); err != nil {
			return fmt.Errorf("invalid trusted relay fee granter %q: %w", address, err)
		}
		if _, ok := seen[address]; ok {
			return fmt.Errorf("duplicate trusted relay fee granter %q", address)
		}
		seen[address] = struct{}{}
	}
	return nil
}

// IsTrustedRelayFeeGranter reports whether address can authorize relay workers.
func (p Params) IsTrustedRelayFeeGranter(address string) bool {
	for _, trusted := range p.TrustedRelayFeeGranters {
		if trusted == address {
			return true
		}
	}
	return false
}
