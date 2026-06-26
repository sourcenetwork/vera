package types

import (
	errorsmod "cosmossdk.io/errors"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

const (
	DefaultNodeOfflineDemerits       = uint64(1)
	DefaultDemeritResetIntervalSecs = uint64(86400)
)

var _ paramtypes.ParamSet = (*Params)(nil)

// ParamKeyTable returns the module param key table.
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance.
func NewParams(defaultDemeritConfig DemeritConfig) Params {
	return Params{
		DefaultDemeritConfig: defaultDemeritConfig,
	}
}

// DefaultParams returns a default set of parameters.
func DefaultParams() Params {
	return NewParams(DefaultDemeritConfig())
}

// DefaultDemeritConfig returns the module's default report demerit policy.
func DefaultDemeritConfig() DemeritConfig {
	return DemeritConfig{
		NodeOfflineDemerits:   DefaultNodeOfflineDemerits,
		ResetIntervalSeconds: DefaultDemeritResetIntervalSecs,
	}
}

// ParamSetPairs returns the params.ParamSet.
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{}
}

// Validate validates the set of params.
func (p Params) Validate() error {
	return ValidateDemeritConfig(p.DefaultDemeritConfig)
}

// ValidateDemeritConfig validates per-report-type demerit point values.
func ValidateDemeritConfig(config DemeritConfig) error {
	if config.NodeOfflineDemerits == 0 {
		return errorsmod.Wrap(ErrInvalidRing, "node_offline_demerits must be at least 1")
	}
	if config.ResetIntervalSeconds == 0 {
		return errorsmod.Wrap(ErrInvalidRing, "reset_interval_seconds must be at least 1")
	}
	return nil
}
