package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateTrustedAuthRelayDIDs(t *testing.T) {
	const relayDID = "did:key:z6MkpTHR8VNsBxYAAWHut2Geadd9jSwuBV8xRoAnwWsdvktH"

	require.NoError(t, ValidateTrustedAuthRelayDIDs(nil))
	require.NoError(t, ValidateTrustedAuthRelayDIDs([]string{relayDID}))
	require.Error(t, ValidateTrustedAuthRelayDIDs([]string{""}))
	require.Error(t, ValidateTrustedAuthRelayDIDs([]string{" " + relayDID}))
	require.Error(t, ValidateTrustedAuthRelayDIDs([]string{"did:example:relay"}))
	require.Error(t, ValidateTrustedAuthRelayDIDs([]string{
		"did:key:zQ3shokFTS3brHcDQrn82RUDfCZESWL1ZdCEJwekUDPQiYBme",
	}))
	require.Error(t, ValidateTrustedAuthRelayDIDs([]string{relayDID, relayDID}))
}

func TestGenesisValidateRejectsInvalidTrustedAuthRelay(t *testing.T) {
	genesis := DefaultGenesis()
	genesis.Rings = []Ring{{
		Id:                     "ring-1",
		AllowTrustedAuthRelays: true,
		TrustedAuthRelayDids:   []string{"did:example:relay"},
	}}

	require.ErrorContains(t, genesis.Validate(), "ring \"ring-1\" trusted auth relays")
}

func TestGenesisValidateRejectsTrustedAuthRelaysWithoutOptIn(t *testing.T) {
	genesis := DefaultGenesis()
	genesis.Rings = []Ring{{
		Id:                   "ring-1",
		TrustedAuthRelayDids: []string{"did:key:z6MkpTHR8VNsBxYAAWHut2Geadd9jSwuBV8xRoAnwWsdvktH"},
	}}

	require.ErrorContains(t, genesis.Validate(), "require allow_trusted_auth_relays")
}
