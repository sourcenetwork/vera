package jubjub

import "errors"

const (
	// ElementSize is the size in bytes of a canonically-encoded Jubjub
	// prime-order-subgroup point.
	ElementSize = 32
	// ScalarSize is the size in bytes of a canonically-encoded Jubjub
	// scalar.
	ScalarSize = 32
)

// IdentityBytes is Jubjub's identity point's canonical encoding: 0x01
// followed by 31 zero bytes (the little-endian encoding of v=1, u=0, sign
// bit 0). This is a fixed, publicly-known constant pinned by
// orbis-rs's own test (crates/crypto/src/jubjub/types.rs,
// point_canonical_identity_and_arithmetic) — unlike decaf377, Jubjub's
// identity does NOT encode as all-zero bytes.
var IdentityBytes = [ElementSize]byte{0x01}

var (
	ErrInvalidEncoding = errors.New("jubjub: invalid encoding")
	ErrInvalidScalar   = errors.New("jubjub: invalid scalar")
	ErrWasmTrap        = errors.New("jubjub: wasm module trapped")
)
