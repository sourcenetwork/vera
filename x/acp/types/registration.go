package types

// ObjectClaimEvents contain a subset of ObjectRegistrationEventType
// which represents an actor's claim over an object,
// meaning that the object stated ownership over the object.
var ObjectClaimEvents []ObjectRegistrationEventType = []ObjectRegistrationEventType{
	ObjectRegistrationEventType_AMENDMENT,
	ObjectRegistrationEventType_REGISTRATION,
	ObjectRegistrationEventType_REVEAL_REGISTRATION,
}
