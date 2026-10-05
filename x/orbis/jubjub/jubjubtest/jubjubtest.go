// Package jubjubtest provides test-fixture helpers (derive a public key
// from a secret scalar, produce a deterministic-nonce signature) backed by
// the same real Jubjub implementation jubjubfrost.Verify checks against —
// so a test fixture can never diverge from what production verification
// expects. Never imported by production code.
package jubjubtest

import (
	"context"

	jubjub "github.com/sourcenetwork/vera/x/orbis/jubjub"
)

// DerivePublicKey computes secret*G, where secret is a 32-byte
// little-endian canonical scalar.
func DerivePublicKey(secret []byte) ([]byte, error) {
	return jubjub.CallDerivePublicKey(context.Background(), secret)
}

// ReduceScalarWide reduces an arbitrary 64-byte seed into a valid,
// canonical secret scalar (the same wide-reduction operation used
// throughout production code). Use this to turn an arbitrary test seed
// into a usable secret, rather than hand-rolling a big.Int modular
// reduction against a hardcoded modulus constant.
func ReduceScalarWide(seed64 []byte) ([]byte, error) {
	return jubjub.CallReduceScalarWide(context.Background(), seed64)
}

// Sign produces a deterministic-nonce Schnorr signature (R||z, 64 bytes)
// over msg, for the keypair (secret, pubKeyBytes). Deterministic because
// wasm32-unknown-unknown has no OS entropy source and test fixtures don't
// need one — see jubjub-wasm's test_sign_core doc comment.
func Sign(secret, pubKeyBytes, msg []byte) ([]byte, error) {
	return jubjub.CallTestSign(context.Background(), secret, pubKeyBytes, msg)
}
