package types

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
	ErrInvalidPostUpdater        = sdkerrors.Register(ModuleName, 1110, "expected authorized account as a post updater")
	ErrCollaboratorAlreadyExists = sdkerrors.Register(ModuleName, 1111, "collaborator already exists")
	ErrCollaboratorNotFound      = sdkerrors.Register(ModuleName, 1112, "collaborator not found")
	ErrCouldNotEnsurePolicy      = sdkerrors.Register(ModuleName, 1113, "could not ensure policy")
	ErrInvalidThresholdSignature = sdkerrors.Register(ModuleName, 1114, "invalid threshold signature")
	ErrInvalidPostId             = sdkerrors.Register(ModuleName, 1115, "invalid post id")
	ErrInvalidSignatureScheme      = sdkerrors.Register(ModuleName, 1116, "invalid signature scheme")
	ErrInvalidSignaturePayload     = sdkerrors.Register(ModuleName, 1117, "invalid signature payload")
	ErrRingPayloadMissingPolicyId  = sdkerrors.Register(ModuleName, 1118, "ring payload missing policy_id")
	ErrReshareInProgress           = sdkerrors.Register(ModuleName, 1119, "reshare already in progress: finalize before changing reshare parameters")
)
