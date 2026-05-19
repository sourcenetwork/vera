package types

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNamespaceScopedKeysAreLengthPrefixed(t *testing.T) {
	documentKey := DocumentKey("orbis/a", "b/c")
	collidingDocumentKey := DocumentKey("orbis/a/b", "c")
	require.False(t, bytes.Equal(documentKey, collidingDocumentKey))

	keyDerivationKey := KeyDerivationKey("orbis/a", "b/c")
	collidingKeyDerivationKey := KeyDerivationKey("orbis/a/b", "c")
	require.False(t, bytes.Equal(keyDerivationKey, collidingKeyDerivationKey))

	require.True(t, bytes.HasPrefix(documentKey, NamespacePrefix("orbis/a")))
	require.True(t, bytes.HasPrefix(keyDerivationKey, NamespacePrefix("orbis/a")))
}
