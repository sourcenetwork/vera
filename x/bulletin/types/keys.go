package types

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
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

// PostKey builds and returns the store key to store/retrieve the Post.
func PostKey(namespaceId string, postId string) []byte {
	// Precompute buffer size
	size := len(namespaceId) + 1 + len(postId)
	buf := make([]byte, 0, size)

	// Append bytes to the buffer
	buf = append(buf, namespaceId...)
	buf = append(buf, '/')
	buf = append(buf, postId...)

	return buf
}

// CollaboratorKey builds and returns the store key to store/retrieve the Actor.
func CollaboratorKey(namespaceId string, collaboratorDID string) []byte {
	// Precompute buffer size
	size := len(namespaceId) + 1 + len(collaboratorDID)
	buf := make([]byte, 0, size)

	// Append bytes to the buffer
	buf = append(buf, namespaceId...)
	buf = append(buf, '/')
	buf = append(buf, collaboratorDID...)

	return buf
}

// ParsePostKey retrieves namespaceId and postId from the Post key.
func ParsePostKey(key []byte) (namespaceId string, postId string) {
	parts := bytes.Split(key, []byte{'/'})
	if len(parts) != 2 {
		panic("invalid post key format: expected format namespaceId/postId")
	}

	return string(parts[0]), string(parts[1])
}

// ParseCollaboratorKey retrieves namespaceId and actorDID from the Collaborator key.
func ParseCollaboratorKey(key []byte) (namespaceId string, actorDID string) {
	parts := bytes.Split(key, []byte{'/'})
	if len(parts) != 2 {
		panic("invalid post key format: expected format namespaceId/actorDID")
	}

	return string(parts[0]), string(parts[1])
}

// GeneratePostId generates deterministic post id from namespaceId and post payload.
func GeneratePostId(namespaceId string, payload []byte) string {
	hash := sha256.Sum256([]byte(namespaceId + string(payload)))
	return hex.EncodeToString(hash[:])
}
