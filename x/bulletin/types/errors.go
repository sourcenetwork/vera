package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/bulletin module sentinel errors
var (
	ErrInvalidSigner             = sdkerrors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrInvalidPolicyId           = sdkerrors.Register(ModuleName, 1101, "invalid policy id")
	ErrNamespaceAlreadyExists    = sdkerrors.Register(ModuleName, 1102, "namespace already exists")
	ErrNamespaceNotFound         = sdkerrors.Register(ModuleName, 1103, "namespace not found")
	ErrInvalidNamespaceId        = sdkerrors.Register(ModuleName, 1104, "invalid namespace id")
	ErrInvalidNamespaceOwner     = sdkerrors.Register(ModuleName, 1105, "expected authorized account as a namespace owner")
	ErrInvalidPostCreator        = sdkerrors.Register(ModuleName, 1106, "expected authorized account as a post creator")
	ErrPostAlreadyExists         = sdkerrors.Register(ModuleName, 1107, "post already exists")
	ErrPostNotFound              = sdkerrors.Register(ModuleName, 1108, "post not found")
	ErrInvalidPostPayload        = sdkerrors.Register(ModuleName, 1109, "invalid post payload")
	ErrInvalidPostProof          = sdkerrors.Register(ModuleName, 1110, "invalid post proof")
	ErrCollaboratorAlreadyExists = sdkerrors.Register(ModuleName, 1111, "collaborator already exists")
	ErrCollaboratorNotFound      = sdkerrors.Register(ModuleName, 1112, "collaborator not found")
)
