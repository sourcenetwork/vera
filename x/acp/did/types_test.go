package did

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"
)

func TestIsValidDID(t *testing.T) {
	tests := []struct {
		name    string
		did     string
		wantErr bool
	}{
		{"valid did:key", "did:key:z6MkhaXgBZDvotDkL5257faiztiGiC2QtKLGpbnnEGta2doK", false},
		{"invalid did", "not-a-did", true},
		{"empty string", "", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := IsValidDID(tc.did)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestIssueDID(t *testing.T) {
	priv := ed25519.GenPrivKey()
	pub := priv.PubKey()
	acc := authtypes.NewBaseAccount(pub.Address().Bytes(), pub, 0, 0)

	did, err := IssueDID(acc)
	require.NoError(t, err)
	require.Contains(t, did, "did:key:")
}

func TestDIDFromPubKeyNilPanics(t *testing.T) {
	require.Panics(t, func() {
		DIDFromPubKey(nil)
	})
}

func TestIssueModuleDID(t *testing.T) {
	did := IssueModuleDID("acp")
	require.Equal(t, "did:module:acp", did)
}

func TestIssueInterchainAccountDID(t *testing.T) {
	did := IssueInterchainAccountDID("cosmos1abc")
	require.Equal(t, "did:ica:cosmos1abc", did)
}
