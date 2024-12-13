package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/tier module sentinel errors
var (
	ErrUnauthorized   = sdkerrors.Register(ModuleName, 1101, "unauthorized")
	ErrNotFound       = sdkerrors.Register(ModuleName, 1102, "not found")
	ErrInvalidRequest = sdkerrors.Register(ModuleName, 1103, "invalid request")
	ErrInvalidAddress = sdkerrors.Register(ModuleName, 1104, "invalid address")
	ErrInvalidDenom   = sdkerrors.Register(ModuleName, 1105, "invalid denom")
	ErrInvalidAmount  = sdkerrors.Register(ModuleName, 1106, "invalid amount")
)
