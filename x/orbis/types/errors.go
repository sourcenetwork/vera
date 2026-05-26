package types

import sdkerrors "cosmossdk.io/errors"

// x/orbis module sentinel errors.
var (
	ErrInvalidSigner              = sdkerrors.Register(ModuleName, 1200, "expected gov account as only signer for proposal message")
	ErrInvalidRingId              = sdkerrors.Register(ModuleName, 1202, "invalid ring id")
	ErrInvalidDocumentId          = sdkerrors.Register(ModuleName, 1203, "invalid document id")
	ErrInvalidKeyDerivationId     = sdkerrors.Register(ModuleName, 1204, "invalid key derivation id")
	ErrInvalidRing                = sdkerrors.Register(ModuleName, 1205, "invalid ring")
	ErrRingAlreadyExists          = sdkerrors.Register(ModuleName, 1206, "ring already exists")
	ErrRingNotFound               = sdkerrors.Register(ModuleName, 1207, "ring not found")
	ErrDocumentAlreadyExists      = sdkerrors.Register(ModuleName, 1208, "document already exists")
	ErrDocumentNotFound           = sdkerrors.Register(ModuleName, 1209, "document not found")
	ErrKeyDerivationAlreadyExists = sdkerrors.Register(ModuleName, 1210, "key derivation already exists")
	ErrKeyDerivationNotFound      = sdkerrors.Register(ModuleName, 1211, "key derivation not found")
	ErrInvalidDocument            = sdkerrors.Register(ModuleName, 1212, "invalid document")
	ErrInvalidKeyDerivation       = sdkerrors.Register(ModuleName, 1213, "invalid key derivation")
	ErrReshareInProgress          = sdkerrors.Register(ModuleName, 1216, "reshare already in progress: finalize before changing reshare parameters")
	ErrInvalidThresholdSignature  = sdkerrors.Register(ModuleName, 1217, "invalid threshold signature")
	ErrInvalidSignatureScheme     = sdkerrors.Register(ModuleName, 1218, "invalid signature scheme")
	ErrInvalidSignaturePayload    = sdkerrors.Register(ModuleName, 1219, "invalid signature payload")
	ErrNodeInfoAlreadyExists      = sdkerrors.Register(ModuleName, 1220, "node info already exists")
	ErrNodeInfoNotFound           = sdkerrors.Register(ModuleName, 1221, "node info not found")
	ErrInvalidNodeInfo            = sdkerrors.Register(ModuleName, 1222, "invalid node info")
	ErrUnauthorizedNodeInfoUpdate = sdkerrors.Register(ModuleName, 1223, "tx signer does not match controller key")
	ErrRingAlreadyFinalized       = sdkerrors.Register(ModuleName, 1225, "ring already has a public key set")
	ErrInvalidRingFinalizer       = sdkerrors.Register(ModuleName, 1226, "tx signer is not a member of the ring's peer set")
	ErrDuplicateConfirmation      = sdkerrors.Register(ModuleName, 1227, "node has already submitted a finalize confirmation")
	ErrRingPkConflict             = sdkerrors.Register(ModuleName, 1228, "submitted ring_pk conflicts with existing confirmation; ring deleted")
	ErrRingNotFinalized           = sdkerrors.Register(ModuleName, 1229, "ring is not finalized")
)
