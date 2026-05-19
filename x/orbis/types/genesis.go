package types

// DefaultGenesis returns the default genesis state.
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:         DefaultParams(),
		Rings:          []Ring{},
		Documents:      []Document{},
		KeyDerivations: []KeyDerivation{},
	}
}

// Validate performs basic genesis state validation.
func (gs GenesisState) Validate() error {
	return gs.Params.Validate()
}
