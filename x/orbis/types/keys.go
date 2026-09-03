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

// idDocumentSecret / idDocumentProof mirror the orbis-rs crypto `Secret` /
// `EncryptionProof` — only the fields needed to derive a canonical,
// serialization-independent document id. Pointer fields so a missing field is
// rejected rather than silently treated as empty (matching the Rust side).
type idDocumentSecret struct {
	EncCmt        *idBytes `json:"enc_cmt"`
	EncryptedData *idBytes `json:"encrypted_data"`
	Nonce         *idBytes `json:"nonce"`
}

type idDocumentProof struct {
	Challenge *idBytes `json:"challenge"`
	Response  *idBytes `json:"response"`
}

// GenerateDocumentID returns the stable ID for an encrypted document.
//
// `document` and `proof` are the JSON blobs from MsgStoreDocument. They are
// parsed and re-encoded into a canonical length-prefixed form before hashing, so
// two byte-different JSON encodings of the *same* ciphertext produce the *same*
// id — a semantic ciphertext has exactly one authorization identity. Returns an
// error if either blob is not the expected shape.
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
	var secret idDocumentSecret
	if err := json.Unmarshal([]byte(document), &secret); err != nil {
		return "", fmt.Errorf("malformed document: %w", err)
	}
	if secret.EncCmt == nil || secret.EncryptedData == nil || secret.Nonce == nil {
		return "", fmt.Errorf("malformed document: missing enc_cmt, encrypted_data, or nonce")
	}
	var pf idDocumentProof
	if err := json.Unmarshal([]byte(proof), &pf); err != nil {
		return "", fmt.Errorf("malformed proof: %w", err)
	}
	if pf.Challenge == nil || pf.Response == nil {
		return "", fmt.Errorf("malformed proof: missing challenge or response")
	}

	h := newIDHasher("orbis/document/v2")
	h.writeString(ringID)
	h.writeBytes(*secret.EncCmt)
	h.writeBytes(*secret.EncryptedData)
	h.writeBytes(*secret.Nonce)
	h.writeBytes(*pf.Challenge)
	h.writeBytes(*pf.Response)
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
