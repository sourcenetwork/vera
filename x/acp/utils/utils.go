package utils

import "crypto/sha256"

// HasTx produces a sha256 of a Tx bytes
func HashTx(txBytes []byte) []byte {
	hasher := sha256.New()
	hasher.Write(txBytes)
	return hasher.Sum(nil)
}
