package utils

import (
	"crypto/sha256"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHashTx(t *testing.T) {
	tests := []struct {
		name    string
		txBytes []byte
	}{
		{"empty input", []byte{}},
		{"single byte", []byte{0x42}},
		{"typical tx bytes", []byte("some-tx-payload-bytes")},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := HashTx(tc.txBytes)
			require.Len(t, result, sha256.Size)

			expected := sha256.Sum256(tc.txBytes)
			require.Equal(t, expected[:], result)
		})
	}
}

func TestHashTxDeterministic(t *testing.T) {
	input := []byte("deterministic-test")
	a := HashTx(input)
	b := HashTx(input)
	require.Equal(t, a, b)
}

func TestHashTxDifferentInputs(t *testing.T) {
	a := HashTx([]byte("input-a"))
	b := HashTx([]byte("input-b"))
	require.NotEqual(t, a, b)
}

func TestWriteBytes(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"nil slice", nil},
		{"empty slice", []byte{}},
		{"non-empty data", []byte("hello")},
		{"single byte", []byte{0xFF}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := sha256.New()
			WriteBytes(h, tc.data)

			// manually build expected hash state
			expected := sha256.New()
			var lenBuf [8]byte
			binary.BigEndian.PutUint64(lenBuf[:], uint64(len(tc.data)))
			expected.Write(lenBuf[:])
			if len(tc.data) > 0 {
				expected.Write(tc.data)
			}
			require.Equal(t, expected.Sum(nil), h.Sum(nil))
		})
	}
}

func TestWriteBytesLengthPrefix(t *testing.T) {
	// different-length inputs with same content prefix should produce different hashes
	h1 := sha256.New()
	WriteBytes(h1, []byte("ab"))

	h2 := sha256.New()
	WriteBytes(h2, []byte("abc"))

	require.NotEqual(t, h1.Sum(nil), h2.Sum(nil))
}
