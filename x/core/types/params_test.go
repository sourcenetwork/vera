package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const testRelayAddress = "source1r5v5srda7xfth3hn2s26txvrcrntldjuac798p"

func TestParamsValidateTrustedRelayFeeGranters(t *testing.T) {
	require.NoError(t, DefaultParams().Validate())
	require.NoError(t, (Params{TrustedRelayFeeGranters: []string{testRelayAddress}}).Validate())
	require.Error(t, (Params{TrustedRelayFeeGranters: []string{"invalid"}}).Validate())
	require.Error(t, (Params{TrustedRelayFeeGranters: []string{testRelayAddress, testRelayAddress}}).Validate())
}

func TestParamsIsTrustedRelayFeeGranter(t *testing.T) {
	params := Params{TrustedRelayFeeGranters: []string{testRelayAddress}}
	require.True(t, params.IsTrustedRelayFeeGranter(testRelayAddress))
	require.False(t, params.IsTrustedRelayFeeGranter("source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et"))
}
