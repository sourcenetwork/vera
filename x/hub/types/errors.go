package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/hub module sentinel errors.
var (
	ErrInvalidSigner            = sdkerrors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrSample                   = sdkerrors.Register(ModuleName, 1101, "sample error")
	ErrInvalidInput             = sdkerrors.Register(ModuleName, 1102, "invalid input")
	ErrJWSTokenNotFound         = sdkerrors.Register(ModuleName, 1103, "JWS token not found")
	ErrJWSTokenAlreadyInvalid   = sdkerrors.Register(ModuleName, 1104, "JWS token is already invalid")
	ErrUnauthorizedInvalidation = sdkerrors.Register(ModuleName, 1105, "unauthorized to invalidate JWS token")
	ErrFailedToInvalidateToken  = sdkerrors.Register(ModuleName, 1106, "failed to invalidate JWS token")
	ErrJWSTokenExpired          = sdkerrors.Register(ModuleName, 1107, "JWS token has expired")
	ErrJWSTokenInvalid          = sdkerrors.Register(ModuleName, 1108, "JWS token is invalid")
	ErrConfigSet                = sdkerrors.Register(ModuleName, 1109, "ChainConfig already initialized: config is immutable and can only be set at genesis")
)
