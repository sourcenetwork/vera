package types

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewParams(t *testing.T) {
	validity := NewDurationFromTimeDuration(5 * time.Minute)
	p := NewParams(1000, validity)
	require.Equal(t, uint64(1000), p.PolicyCommandMaxExpirationDelta)
	require.Equal(t, validity, p.RegistrationsCommitmentValidity)
}

func TestDefaultParams(t *testing.T) {
	p := DefaultParams()
	require.Equal(t, uint64(DefaultPolicyCommandMaxExpirationDelta), p.PolicyCommandMaxExpirationDelta)
	require.NotNil(t, p.RegistrationsCommitmentValidity)
}

func TestParamsValidate(t *testing.T) {
	p := DefaultParams()
	require.NoError(t, p.Validate())

	// zero values also pass since Validate is a no-op
	zero := Params{}
	require.NoError(t, zero.Validate())
}

func TestParamKeyTable(t *testing.T) {
	kt := ParamKeyTable()
	require.NotNil(t, kt)
}

func TestParamSetPairs(t *testing.T) {
	p := DefaultParams()
	pairs := p.ParamSetPairs()
	require.Empty(t, pairs)
}
