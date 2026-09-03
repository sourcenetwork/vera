package types

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/sourcenetwork/immutable"
)

const (
	// ModuleName defines the module name.
	ModuleName = "orbis"

	// StoreKey defines the primary module store key.
	StoreKey = ModuleName

	RingKeyPrefix                  = "ring/"
	DocumentKeyPrefix              = "document/"
	KeyDerivationKeyPrefix         = "key_derivation/"
	NodeInfoKeyPrefix              = "node_info/"
	AcceptedReportKeyPrefix        = "accepted_report/"
	AcceptedReportSessionKeyPrefix = "accepted_report_session/"
	AcceptedReportExpiryKeyPrefix  = "accepted_report_expiry/"
	NodeDemeritKeyPrefix           = "node_demerit/"
)

var (
	ParamsKey = []byte("p_orbis")
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}

// GenerateRingID returns the stable ID for a ring's creation parameters.
func GenerateRingID(
	peerNodeKeys []string,
	threshold uint32,
	pssInterval uint64,
	policyID string,
	nonce immutable.Option[string],
	currentVersion uint64,
	allowTrustedAuthRelays bool,
	trustedAuthRelayDIDs []string,
) string {
	sortedPeerNodeKeys := slices.Clone(peerNodeKeys)
	if !slices.IsSorted(sortedPeerNodeKeys) {
		slices.Sort(sortedPeerNodeKeys)
	}

	h := newIDHasher("orbis/ring")
	h.writeStringSlice(sortedPeerNodeKeys)
	h.writeUint32(threshold)
	h.writeUint64(pssInterval)
	h.writeString(policyID)
	h.writeOptionalString(nonce)
	h.writeUint64(currentVersion)
	h.writeBool(allowTrustedAuthRelays)
	sortedRelayDIDs := slices.Clone(trustedAuthRelayDIDs)
	slices.Sort(sortedRelayDIDs)
	h.writeStringSlice(sortedRelayDIDs)
	return h.sum()
}

func (h *idHasher) writeUint64(value uint64) {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], value)
	h.bytes = append(h.bytes, buf[:]...)
}

func (h *idHasher) writeBool(value bool) {
	if value {
		h.bytes = append(h.bytes, 1)
		return
	}
	h.bytes = append(h.bytes, 0)
}

// idBytes decodes a serde_json-encoded Rust Vec<u8>: a JSON array of integers in
// 0-255. (Go's encoding/json would natively expect base64 for []byte, which is
// why a custom decoder is needed to interoperate with orbis-rs.)
type idBytes []byte

func (b *idBytes) UnmarshalJSON(data []byte) error {
	var nums []uint16
	if err := json.Unmarshal(data, &nums); err != nil {
		return err
	}
	out := make([]byte, len(nums))
	for i, n := range nums {
		if n > 255 {
			return fmt.Errorf("byte value %d out of range 0-255", n)
		}
		out[i] = byte(n)
	}
	*b = out
	return nil
}

// decodeIDByteFields parses `raw` as a JSON object that must contain *exactly*
// the fields in `want` (same names, same case, nothing extra) and decodes each
// value as a serde-style byte array.
//
// The exact-key requirement matters: Go's encoding/json matches struct fields
// case-insensitively, so a struct decode of `{"enc_cmt":[1],"ENC_CMT":[9]}`
// would yield [9] here while Serde on the orbis-rs side yields [1] — one
// ciphertext, two object ids. Decoding through a map keyed on the exact JSON
// string, and rejecting any key not in `want`, keeps both sides in agreement.
func decodeIDByteFields(raw string, want []string) (map[string][]byte, error) {
	var m map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return nil, err
	}
	if len(m) != len(want) {
		return nil, fmt.Errorf("expected exactly fields %v, got %d fields", want, len(m))
	}
	out := make(map[string][]byte, len(want))
	for _, key := range want {
		v, ok := m[key]
		if !ok {
			return nil, fmt.Errorf("missing or misnamed field %q", key)
		}
		var b idBytes
		if err := b.UnmarshalJSON(v); err != nil {
			return nil, fmt.Errorf("field %q: %w", key, err)
		}
		out[key] = b
	}
	return out, nil
}

// GenerateDocumentID returns the stable ID for an encrypted document.
//
// `document` and `proof` are the JSON blobs from MsgStoreDocument. They are
// parsed and re-encoded into a canonical length-prefixed form before hashing, so
// two byte-different JSON encodings of the *same* ciphertext produce the *same*
// id — a semantic ciphertext has exactly one authorization identity. Returns an
// error if either blob is not exactly the expected shape.
func GenerateDocumentID(
	ringID string,
	document string,
	proof string,
	policyID string,
	resource string,
	permission string,
	tier immutable.Option[string],
	timestamp immutable.Option[uint64],
) (string, error) {
	secret, err := decodeIDByteFields(document, []string{"enc_cmt", "encrypted_data", "nonce"})
	if err != nil {
		return "", fmt.Errorf("malformed document: %w", err)
	}
	pf, err := decodeIDByteFields(proof, []string{"challenge", "response"})
	if err != nil {
		return "", fmt.Errorf("malformed proof: %w", err)
	}

	h := newIDHasher("orbis/document/v2")
	h.writeString(ringID)
	h.writeBytes(secret["enc_cmt"])
	h.writeBytes(secret["encrypted_data"])
	h.writeBytes(secret["nonce"])
	h.writeBytes(pf["challenge"])
	h.writeBytes(pf["response"])
	h.writeString(policyID)
	h.writeString(resource)
	h.writeString(permission)
	h.writeOptionalString(tier)
	h.writeOptionalUint64(timestamp)
	return h.sum(), nil
}

// GenerateKeyDerivationID returns the stable ID for a key derivation.
func GenerateKeyDerivationID(ringID, derivation, policyID, resource, permission string) string {
	h := newIDHasher("orbis/key_derivation/v1")
	h.writeString(ringID)
	h.writeString(derivation)
	h.writeString(policyID)
	h.writeString(resource)
	h.writeString(permission)
	return h.sum()
}

type idHasher struct {
	bytes []byte
}

func newIDHasher(domain string) *idHasher {
	h := &idHasher{}
	h.writeString(domain)
	return h
}

func (h *idHasher) writeString(value string) {
	h.writeBytes([]byte(value))
}

func (h *idHasher) writeOptionalString(value immutable.Option[string]) {
	if !value.HasValue() {
		h.bytes = append(h.bytes, 0)
		return
	}
	h.bytes = append(h.bytes, 1)
	h.writeString(value.Value())
}

func (h *idHasher) writeStringSlice(values []string) {
	var buf [4]byte
	binary.BigEndian.PutUint32(buf[:], uint32(len(values)))
	h.bytes = append(h.bytes, buf[:]...)
	for _, value := range values {
		h.writeString(value)
	}
}

func (h *idHasher) writeUint32(value uint32) {
	var buf [4]byte
	binary.BigEndian.PutUint32(buf[:], value)
	h.bytes = append(h.bytes, buf[:]...)
}

func (h *idHasher) writeOptionalUint64(value immutable.Option[uint64]) {
	if !value.HasValue() {
		h.bytes = append(h.bytes, 0)
		return
	}
	h.bytes = append(h.bytes, 1)
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], value.Value())
	h.bytes = append(h.bytes, buf[:]...)
}

func (h *idHasher) writeBytes(value []byte) {
	var buf [4]byte
	binary.BigEndian.PutUint32(buf[:], uint32(len(value)))
	h.bytes = append(h.bytes, buf[:]...)
	h.bytes = append(h.bytes, value...)
}

func (h *idHasher) sum() string {
	hash := sha256.Sum256(h.bytes)
	return hex.EncodeToString(hash[:])
}
