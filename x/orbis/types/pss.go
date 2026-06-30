package types

import errorsmod "cosmossdk.io/errors"

const MinPSSIntervalSeconds uint64 = 86400

func ValidatePSSInterval(pssInterval uint64) error {
	if pssInterval < MinPSSIntervalSeconds {
		return errorsmod.Wrapf(ErrInvalidRing, "pss_interval must be at least %d seconds", MinPSSIntervalSeconds)
	}
	return nil
}
