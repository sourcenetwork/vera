package types

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"sort"
)

const (
	// ModuleName defines the module name.
	ModuleName = "orbis"

	// StoreKey defines the primary module store key.
	StoreKey = ModuleName

	RingKeyPrefix          = "ring/"
	DocumentKeyPrefix      = "document/"
	KeyDerivationKeyPrefix = "key_derivation/"
	NodeInfoKeyPrefix      = "node_info/"
)

var (
	ParamsKey = []byte("p_orbis")
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}

// GenerateRingID returns the stable ID for a ring's creation parameters.
// peerNodeKeys are sorted before hashing so the ID is order-independent.
func GenerateRingID(peerNodeKeys []string, threshold uint32, pssInterval *uint64, policyID string, nonce *string) string {
	sorted := make([]string, len(peerNodeKeys))
	copy(sorted, peerNodeKeys)
	sort.Strings(sorted)

	h := newIDHasher("orbis/ring/v1")
	h.writeStringSlice(sorted)
	h.writeUint32(threshold)
	h.writeOptionalUint64(pssInterval)
	h.writeString(policyID)
	h.writeOptionalString(nonce)
	return h.sum()
}

// GenerateDocumentID returns the stable ID for an encrypted document.
func GenerateDocumentID(
	ringID string,
	document string,
	proof string,
	policyID string,
	resource string,
	permission string,
	tier *string,
	timestamp *uint64,
) string {
	h := newIDHasher("orbis/document/v1")
	h.writeString(ringID)
	h.writeString(document)
	h.writeString(proof)
	h.writeString(policyID)
	h.writeString(resource)
	h.writeString(permission)
	h.writeOptionalString(tier)
	h.writeOptionalUint64(timestamp)
	return h.sum()
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

func (h *idHasher) writeOptionalString(value *string) {
	if value == nil {
		h.bytes = append(h.bytes, 0)
		return
	}
	h.bytes = append(h.bytes, 1)
	h.writeString(*value)
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

func (h *idHasher) writeOptionalUint64(value *uint64) {
	if value == nil {
		h.bytes = append(h.bytes, 0)
		return
	}
	h.bytes = append(h.bytes, 1)
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], *value)
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
