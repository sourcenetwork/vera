package types

import (
	"testing"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/stretchr/testify/require"
)

func TestRegisterInterfaces(t *testing.T) {
	registry := codectypes.NewInterfaceRegistry()
	require.NotPanics(t, func() {
		RegisterInterfaces(registry)
	})
}
