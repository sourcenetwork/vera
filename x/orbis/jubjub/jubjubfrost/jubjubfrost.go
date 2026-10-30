// Package jubjubfrost verifies Jubjub FROST Schnorr signatures — vera's
// entry point into this module. The actual curve arithmetic and challenge
// transcript live in the Rust jubjub-wasm crate (orbis-rs/crates/jubjub-wasm);
// this package only marshals bytes across the Go<->WASM boundary.
package jubjubfrost

import (
	"context"

	jubjub "github.com/sourcenetwork/vera/x/orbis/jubjub"
)

// Verify checks a Jubjub FROST Schnorr signature (R||z, 64 bytes) against
// publicKeyBytes (32 bytes) and msg. Returns (false, nil) for an invalid
// signature, (false, err) if the input couldn't even be evaluated (wrong
// lengths, or an internal wasm trap) — callers must treat both as "reject",
// never "accept".
//
// Verify already rejects an identity public key on its own: it calls the
// same verify_core the real orbis-rs ThresholdJubjubSigner::verify uses,
// which has that check built in (unlike decaf377's bare orbisfrost.Verify,
// which didn't — this is a difference in the two backends' primitives, not
// a gap introduced by this port). vera's threshold_signature.go still also
// calls jubjub.CallIsIdentityPubkey separately before ring finalization,
// independent of any particular signature check — see its own comment for
// why that layer stays regardless.
func Verify(publicKeyBytes, msg, signatureBytes []byte) (bool, error) {
	if len(publicKeyBytes) != jubjub.ElementSize {
		return false, jubjub.ErrInvalidEncoding
	}
	if len(signatureBytes) != jubjub.ElementSize+jubjub.ScalarSize {
		return false, jubjub.ErrInvalidScalar
	}
	return jubjub.CallVerify(context.Background(), publicKeyBytes, msg, signatureBytes)
}
