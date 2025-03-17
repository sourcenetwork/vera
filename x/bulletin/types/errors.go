package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/bulletin module sentinel errors
var (
	ErrInvalidSigner          = sdkerrors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrInvalidPolicyId        = sdkerrors.Register(ModuleName, 1101, "invalid policy id")
	ErrNamespaceAlreadyExists = sdkerrors.Register(ModuleName, 1102, "namespace already exists")
	ErrNamespaceNotFound      = sdkerrors.Register(ModuleName, 1103, "namespace not found")
	ErrInvalidNamespaceOwner  = sdkerrors.Register(ModuleName, 1104, "expected authorized account as a namespace owner")
	ErrInvalidPostCreator     = sdkerrors.Register(ModuleName, 1105, "expected authorized account as a post creator")
	ErrPostAlreadyExists      = sdkerrors.Register(ModuleName, 1106, "post already exists")
	ErrPostNotFound           = sdkerrors.Register(ModuleName, 1107, "post not found")
)
