package types

import (
	errorsmod "cosmossdk.io/errors"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

const (
	DefaultNodeOfflineDemerits           = uint64(1)
	DefaultInvalidCryptoResponseDemerits = uint64(1)
	DefaultUnauthorizedRequestDemerits   = uint64(1)
	DefaultDemeritResetIntervalSecs      = uint64(86400)
	DefaultReportingKickThreshold        = uint64(3)
)

var _ paramtypes.ParamSet = (*Params)(nil)

// ParamKeyTable returns the module param key table.
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance.
func NewParams(defaultReporting ReportingDefaults) Params {
	return Params{
		DefaultReporting: defaultReporting,
	}
}

// DefaultParams returns a default set of parameters.
func DefaultParams() Params {
	return NewParams(DefaultReportingDefaults())
}

// DefaultDemeritConfig returns the module's default report demerit policy.
func DefaultDemeritConfig() DemeritConfig {
	return DemeritConfig{
		NodeOfflineDemerits:           DefaultNodeOfflineDemerits,
		InvalidCryptoResponseDemerits: DefaultInvalidCryptoResponseDemerits,
		UnauthorizedRequestDemerits:   DefaultUnauthorizedRequestDemerits,
		ResetIntervalSeconds:          DefaultDemeritResetIntervalSecs,
	}
}

// DefaultReportingDefaults returns the module's default reporting policy.
func DefaultReportingDefaults() ReportingDefaults {
	return ReportingDefaults{
		DemeritConfig: DefaultDemeritConfig(),
		KickThreshold: DefaultReportingKickThreshold,
	}
}

// ReportingConfigFromDefaults converts module defaults into a ring-scoped config.
func ReportingConfigFromDefaults(defaults ReportingDefaults) ReportingConfig {
	return ReportingConfig{
		DemeritConfig: defaults.DemeritConfig,
		KickThreshold: defaults.KickThreshold,
	}
}

// DefaultReportingConfig returns the default ring-scoped reporting policy.
func DefaultReportingConfig() ReportingConfig {
	return ReportingConfigFromDefaults(DefaultReportingDefaults())
}

// ParamSetPairs returns the params.ParamSet.
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{}
}

// Validate validates the set of params.
func (p Params) Validate() error {
	return ValidateReportingDefaults(p.DefaultReporting)
}

// ValidateDemeritConfig validates per-report-type demerit point values.
func ValidateDemeritConfig(config DemeritConfig) error {
	if config.NodeOfflineDemerits == 0 {
		return errorsmod.Wrap(ErrInvalidRing, "node_offline_demerits must be at least 1")
	}
	if config.InvalidCryptoResponseDemerits == 0 {
		return errorsmod.Wrap(ErrInvalidRing, "invalid_crypto_response_demerits must be at least 1")
	}
	if config.UnauthorizedRequestDemerits == 0 {
		return errorsmod.Wrap(ErrInvalidRing, "unauthorized_request_demerits must be at least 1")
	}
	if config.ResetIntervalSeconds == 0 {
		return errorsmod.Wrap(ErrInvalidRing, "reset_interval_seconds must be at least 1")
	}
	return nil
}

// ValidateReportingDefaults validates module-level reporting defaults.
func ValidateReportingDefaults(defaults ReportingDefaults) error {
	if err := ValidateDemeritConfig(defaults.DemeritConfig); err != nil {
		return err
	}
	if defaults.KickThreshold == 0 {
		return errorsmod.Wrap(ErrInvalidRing, "kick_threshold must be at least 1")
	}
	return nil
}

// ValidateReportingConfig validates ring-level reporting policy values.
func ValidateReportingConfig(config ReportingConfig) error {
	if err := ValidateDemeritConfig(config.DemeritConfig); err != nil {
		return err
	}
	if config.KickThreshold == 0 {
		return errorsmod.Wrap(ErrInvalidRing, "kick_threshold must be at least 1")
	}
	return nil
}
