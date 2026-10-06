package jubjub_test

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	jubjub "github.com/sourcenetwork/vera/x/orbis/jubjub"
	"github.com/sourcenetwork/vera/x/orbis/jubjub/jubjubfrost"
	"github.com/sourcenetwork/vera/x/orbis/jubjub/jubjubtest"
)

type frostVector struct {
	Scheme    string `json:"scheme"`
	Generator string `json:"generator"`
	PublicKey string `json:"public_key"`
	Message   string `json:"message"`
	Challenge string `json:"challenge"`
	Signature string `json:"signature"`
}

func loadVector(t *testing.T) frostVector {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "frost_vector.json"))
	if err != nil {
		t.Fatalf("reading frost_vector.json: %v", err)
	}
	var v frostVector
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatalf("parsing frost_vector.json: %v", err)
	}
	return v
}

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("decoding hex %q: %v", s, err)
	}
	return b
}

// TestOrbisRustFrostVector proves this module's wasm-backed Verify actually
// reproduces orbis-rs's real signing-side behavior end-to-end, not just
// internal self-consistency.
func TestOrbisRustFrostVector(t *testing.T) {
	v := loadVector(t)
	if v.Scheme != "jubjub_frost" {
		t.Fatalf("unexpected scheme %q", v.Scheme)
	}
	pk := mustHex(t, v.PublicKey)
	msg := mustHex(t, v.Message)
	sig := mustHex(t, v.Signature)

	ok, err := jubjubfrost.Verify(pk, msg, sig)
	if err != nil {
		t.Fatalf("Verify returned an error for the real vector: %v", err)
	}
	if !ok {
		t.Fatal("the independently-generated orbis-rs FROST vector must verify")
	}

	ok, err = jubjubfrost.Verify(pk, []byte("a different message"), sig)
	if err != nil {
		t.Fatalf("Verify on a mutated message returned an error (expected false, nil): %v", err)
	}
	if ok {
		t.Fatal("a mutated message must be rejected")
	}
}

// TestIdentityPubkeyRejectedByVerify exercises the primitive-level forgery:
// for Y = identity, (R, z) = (z*G, z) trivially satisfies the verify
// equation for any z, without the "signer" knowing anything. Unlike
// decaf377's bare orbisfrost.Verify, jubjubfrost.Verify's underlying
// verify_core mirrors orbis-rs's real ThresholdJubjubSigner::verify, which
// already rejects the identity itself — so this test demonstrates the
// safety property holds, not that a lower layer is forgeable the way the
// decaf377 regression test did. vera's own identity pre-check
// (jubjub.CallIsIdentityPubkey, exercised by threshold_signature_test.go)
// remains anyway, for ring-finalization-time reasons independent of any one
// signature check.
func TestIdentityPubkeyRejectedByVerify(t *testing.T) {
	z := []byte{42}
	zPadded := make([]byte, jubjub.ScalarSize)
	copy(zPadded, z)

	rPoint, err := jubjubtest.DerivePublicKey(zPadded) // R = z*G
	if err != nil {
		t.Fatalf("deriving R = z*G: %v", err)
	}
	sig := append(append([]byte{}, rPoint...), zPadded...)

	ok, err := jubjubfrost.Verify(jubjub.IdentityBytes[:], []byte("forged"), sig)
	if err != nil {
		t.Fatalf("Verify against the identity returned an error (expected false, nil): %v", err)
	}
	if ok {
		t.Fatal("a forged signature against the identity public key must be rejected")
	}
}

func TestIsIdentityPubkey(t *testing.T) {
	isIdentity, err := jubjub.CallIsIdentityPubkey(context.Background(), jubjub.IdentityBytes[:])
	if err != nil {
		t.Fatalf("CallIsIdentityPubkey: %v", err)
	}
	if !isIdentity {
		t.Fatal("IdentityBytes must be recognized as the identity point")
	}

	secret := make([]byte, jubjub.ScalarSize)
	secret[0] = 9
	pk, err := jubjubtest.DerivePublicKey(secret)
	if err != nil {
		t.Fatalf("DerivePublicKey: %v", err)
	}
	isIdentity, err = jubjub.CallIsIdentityPubkey(context.Background(), pk)
	if err != nil {
		t.Fatalf("CallIsIdentityPubkey: %v", err)
	}
	if isIdentity {
		t.Fatal("a non-identity public key must not be flagged as the identity")
	}
}

func TestMalformedInputsReturnErrorsNotPanics(t *testing.T) {
	cases := []struct {
		name string
		pk   []byte
		msg  []byte
		sig  []byte
	}{
		{"short pubkey", make([]byte, 31), []byte("m"), make([]byte, 64)},
		{"long pubkey", make([]byte, 33), []byte("m"), make([]byte, 64)},
		{"short signature", make([]byte, 32), []byte("m"), make([]byte, 63)},
		{"garbage everything", []byte{0xff, 0xff, 0xff}, []byte("m"), []byte{0xff}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ok, err := jubjubfrost.Verify(c.pk, c.msg, c.sig)
			if ok {
				t.Fatalf("malformed input must never verify as true (err=%v)", err)
			}
			// Either a clean (false, non-nil) rejection or (false, nil) is
			// acceptable here; what matters is this call returns at all
			// rather than panicking the Go process.
		})
	}
}

// TestDeriveAndSignRoundTrip exercises jubjubtest's fixture helpers against
// jubjubfrost.Verify end-to-end, independent of the fixed vector above —
// this is the pattern vera's own test helpers use to sign arbitrary,
// test-run-time-constructed messages.
func TestDeriveAndSignRoundTrip(t *testing.T) {
	secret := make([]byte, jubjub.ScalarSize)
	secret[0] = 123
	secret[1] = 45

	pk, err := jubjubtest.DerivePublicKey(secret)
	if err != nil {
		t.Fatalf("DerivePublicKey: %v", err)
	}
	msg := []byte("vera ring-reshare-finalize payload, constructed at test time")
	sig, err := jubjubtest.Sign(secret, pk, msg)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	ok, err := jubjubfrost.Verify(pk, msg, sig)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !ok {
		t.Fatal("a freshly-derived keypair's own signature must verify")
	}

	ok, err = jubjubfrost.Verify(pk, []byte("tampered"), sig)
	if err != nil {
		t.Fatalf("Verify on tampered message returned an error (expected false, nil): %v", err)
	}
	if ok {
		t.Fatal("a tampered message must be rejected")
	}
}
