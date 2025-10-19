package types

import (
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	tx "github.com/cosmos/cosmos-sdk/types/tx"
)

func (m *JWSExtensionOption) IsExtensionOption() {}

func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*tx.TxExtensionOptionI)(nil),
		&JWSExtensionOption{},
	)
}
