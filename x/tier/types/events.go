package types

const (
	EventTypeCancelUnlocking   = "cancel_unlocking"
	EventTypeCompleteUnlocking = "complete_unlocking"
	EventTypeLock              = "lock"
	EventTypeRedelegate        = "redelegate"
	EventTypeUnlock            = "unlock"

	AttributeKeyCompletionTime       = "completion_time"
	AttributeKeyCreationHeight       = "creation_height"
	AttributeKeyDestinationValidator = "destination_validator"
	AttributeKeySourceValidator      = "source_validator"
	AttributeKeyUnlockTime           = "unlock_time"
)
