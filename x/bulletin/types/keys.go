package types

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

const (
	// ModuleName defines the module name
	ModuleName = "bulletin"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_bulletin"

	PolicyIdKey = "policy_id"

	PostKeyPrefix = "post/"

	NamespaceKeyPrefix = "namespace/"

	CollaboratorKeyPrefix = "collaborator/"
)

var (
	ParamsKey = []byte("p_bulletin")
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}

// SanitizeKeyPart replaces "/" with "|" in key parts to prevent key collisions.
func SanitizeKeyPart(part string) string {
	return strings.ReplaceAll(part, "/", "|")
}

// unsanitizeKeyPart restores "/" from "|" in key parts.
func unsanitizeKeyPart(part string) string {
	return strings.ReplaceAll(part, "|", "/")
}

// PostKey builds and returns the store key to store/retrieve the Post.
func PostKey(namespaceId string, postId string) []byte {
	sanitizedNamespaceId := SanitizeKeyPart(namespaceId)
	sanitizedPostId := SanitizeKeyPart(postId)

	// Precompute buffer size
	size := len(sanitizedNamespaceId) + 1 + len(sanitizedPostId)
	buf := make([]byte, 0, size)

	// Append bytes to the buffer
	buf = append(buf, sanitizedNamespaceId...)
	buf = append(buf, '/')
	buf = append(buf, sanitizedPostId...)

	return buf
}

// CollaboratorKey builds and returns the store key to store/retrieve the Actor.
func CollaboratorKey(namespaceId string, collaboratorDID string) []byte {
	sanitizedNamespaceId := SanitizeKeyPart(namespaceId)
	sanitizedCollaboratorDID := SanitizeKeyPart(collaboratorDID)

	// Precompute buffer size
	size := len(sanitizedNamespaceId) + 1 + len(sanitizedCollaboratorDID)
	buf := make([]byte, 0, size)

	// Append bytes to the buffer
	buf = append(buf, sanitizedNamespaceId...)
	buf = append(buf, '/')
	buf = append(buf, sanitizedCollaboratorDID...)

	return buf
}

// ParsePostKey retrieves namespaceId and postId from the Post key.
func ParsePostKey(key []byte) (namespaceId string, postId string) {
	parts := bytes.Split(key, []byte{'/'})
	if len(parts) != 2 {
		panic("invalid post key format: expected format namespaceId/postId")
	}

	namespaceId = unsanitizeKeyPart(string(parts[0]))
	postId = unsanitizeKeyPart(string(parts[1]))

	return namespaceId, postId
}

// ParseCollaboratorKey retrieves namespaceId and actorDID from the Collaborator key.
func ParseCollaboratorKey(key []byte) (namespaceId string, actorDID string) {
	parts := bytes.Split(key, []byte{'/'})
	if len(parts) != 2 {
		panic("invalid post key format: expected format namespaceId/actorDID")
	}

	namespaceId = unsanitizeKeyPart(string(parts[0]))
	actorDID = unsanitizeKeyPart(string(parts[1]))

	return namespaceId, actorDID
}

// GeneratePostId generates deterministic post id from namespaceId and post payload.
func GeneratePostId(namespaceId string, payload []byte) string {
	hash := sha256.Sum256([]byte(namespaceId + string(payload)))
	return hex.EncodeToString(hash[:])
}
