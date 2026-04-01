package types

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:            DefaultParams(),
		Lockups:           []Lockup{},
		UnlockingLockups:  []UnlockingLockup{},
		InsuranceLockups:  []Lockup{},
		Developers:        []Developer{},
		UserSubscriptions: []UserSubscription{},
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	return gs.Params.Validate()
}
