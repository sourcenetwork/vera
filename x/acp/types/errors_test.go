package types

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewErrInvalidAccAddrErr(t *testing.T) {
	cause := fmt.Errorf("bad format")
	err := NewErrInvalidAccAddrErr(cause, "cosmos1invalid")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid account address")
}

func TestNewAccNotFoundErr(t *testing.T) {
	err := NewAccNotFoundErr("cosmos1missing")
	require.Error(t, err)
	require.Contains(t, err.Error(), "account not found")
}

func TestErrInvalidSigner(t *testing.T) {
	require.NotNil(t, ErrInvalidSigner)
	require.Contains(t, ErrInvalidSigner.Error(), "expected gov account")
}

func TestErrorTypeAliases(t *testing.T) {
	// verify the type aliases are properly wired
	require.NotEqual(t, ErrorType_UNKNOWN, ErrorType_INTERNAL)
	require.NotEqual(t, ErrorType_UNAUTHENTICATED, ErrorType_UNAUTHORIZED)
	require.NotEqual(t, ErrorType_BAD_INPUT, ErrorType_OPERATION_FORBIDDEN)
	require.NotEqual(t, ErrorType_NOT_FOUND, ErrorType_UNKNOWN)
}

func TestErrorConstructorAliases(t *testing.T) {
	err := New("test error", ErrorType_INTERNAL)
	require.Error(t, err)

	wrapped := Wrap("wrapped", err)
	require.Error(t, wrapped)

	withCause := NewWithCause("msg", fmt.Errorf("cause"), ErrorType_BAD_INPUT)
	require.Error(t, withCause)
}
