package jubjub

import "context"

// Buffer layout constants matching the Rust side's documented offsets
// (crates/jubjub-wasm/src/lib.rs). Each exported wasm function has its own
// fixed layout; these are the shared building blocks.
const (
	sigSize = ElementSize + ScalarSize // R || z = 64

	verifyPkOff  = 0
	verifySigOff = ElementSize           // 32
	verifyMsgOff = ElementSize + sigSize // 96 (pk 32 + sig 64)

	signSecretOff = 0
	signPkOff     = ScalarSize               // 32
	signMsgOff    = ScalarSize + ElementSize // 64
)

// CallVerify writes pk/sig/msg into the wasm module's shared buffer at the
// layout the Rust `verify` export expects and returns its raw accept/reject
// result. Callers (jubjubfrost.Verify) are expected to have already
// validated pk/sig lengths.
func CallVerify(ctx context.Context, pk, msg, sig []byte) (bool, error) {
	buf := make([]byte, verifyMsgOff+len(msg))
	copy(buf[verifyPkOff:], pk)
	copy(buf[verifySigOff:], sig)
	copy(buf[verifyMsgOff:], msg)

	result, err := callVerifyLikeI32(ctx, "verify", buf, uint64(len(msg)))
	if err != nil {
		return false, err
	}
	return result == 1, nil
}

// CallIsIdentityPubkey reports whether pk decodes to Jubjub's identity
// point. Returns false (not an error) for any input that fails to decode
// at all — mirrors the existing decaf377 semantics this replaces: this
// check only ever flags the specific identity-forgery case, never a general
// malformed-encoding case (that's left to whatever else validates the
// encoding, e.g. a subsequent Verify call).
func CallIsIdentityPubkey(ctx context.Context, pk []byte) (bool, error) {
	result, err := callVerifyLikeI32(ctx, "is_identity_pubkey", pk)
	if err != nil {
		return false, err
	}
	return result == 1, nil
}

// CallDerivePublicKey computes secret*G via the real Jubjub implementation.
// Test-fixture use only — never called from production code.
func CallDerivePublicKey(ctx context.Context, secret []byte) ([]byte, error) {
	if len(secret) != ScalarSize {
		return nil, ErrInvalidScalar
	}
	return callOutputBytes(ctx, "derive_public_key", secret, ElementSize)
}

// CallReduceScalarWide reduces an arbitrary 64-byte seed modulo the Jubjub
// scalar order (the same wide-reduction operation Fr::from_bytes_wide
// performs throughout production code), returning a canonical 32-byte
// scalar. Test-fixture use only: lets a test derive a valid secret scalar
// from an arbitrary seed without this package exposing the raw scalar-order
// constant at all.
func CallReduceScalarWide(ctx context.Context, seed64 []byte) ([]byte, error) {
	if len(seed64) != 64 {
		return nil, ErrInvalidScalar
	}
	return callOutputBytes(ctx, "reduce_scalar_wide", seed64, ScalarSize)
}

// CallTestSign computes a deterministic-nonce Schnorr signature over msg
// via the real Jubjub implementation. Test-fixture use only — never called
// from production code; see jubjub-wasm's test_sign_core doc comment for
// why a deterministic nonce is used (no OS entropy on wasm32-unknown-
// unknown, and determinism is preferable for reproducible tests anyway).
func CallTestSign(ctx context.Context, secret, pk, msg []byte) ([]byte, error) {
	if len(secret) != ScalarSize || len(pk) != ElementSize {
		return nil, ErrInvalidScalar
	}
	buf := make([]byte, signMsgOff+len(msg))
	copy(buf[signSecretOff:], secret)
	copy(buf[signPkOff:], pk)
	copy(buf[signMsgOff:], msg)

	return callOutputBytes(ctx, "test_sign", buf, ElementSize+ScalarSize, uint64(len(msg)))
}
