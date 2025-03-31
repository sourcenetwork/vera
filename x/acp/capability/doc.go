// package capability provides types that manage capabilities tied to acp policies.
//
// Example usage:
// Calling modules may create a new Policy using acp keeper's `CreateModulePolicy` method,
// which returns a capability.
// The returned capability represents an object which, if presented to the keeper,
// grants the caller unrestricted access to the policy bound to the capability.
//
// Callers which receive a capability are required to Claim this capability,
// using either cosmos-sdk capability keeper directly or the CapabilityManager abstraction.
// Claiming a capability allows it to be subsequently recovered at a later instant of time.
//
// In order to operate over a Policy registered by the module, it must present the Claimed
// capability, which can be fetch with the `Fetch` method of the manager.
package capability
