package signed_policy_cmd

import (
	"crypto/sha256"
	"encoding/base32"
	"encoding/binary"
	"fmt"

	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

// ComputePayloadID deterministically hashes a payload to produce an ID used for replay protection.
func ComputePayloadID(payload *types.SignedPolicyCmdPayload) string {
	const domain = "sourcehub.acp.SignedPolicyCmdPayload:v1"
	hasher := sha256.New()
	hasher.Write([]byte(domain))

	writeBytes := func(bz []byte) {
		var lenBuf [8]byte
		binary.BigEndian.PutUint64(lenBuf[:], uint64(len(bz)))
		hasher.Write(lenBuf[:])
		if len(bz) > 0 {
			hasher.Write(bz)
		}
	}
	writeBytes([]byte(payload.Actor))

	var u64buf [8]byte
	binary.BigEndian.PutUint64(u64buf[:], payload.IssuedHeight)
	hasher.Write(u64buf[:])
	binary.BigEndian.PutUint64(u64buf[:], payload.ExpirationDelta)
	hasher.Write(u64buf[:])
	writeBytes([]byte(payload.PolicyId))

	if payload.Cmd != nil {
		if bz, err := payload.Cmd.Marshal(); err == nil {
			writeBytes(bz)
		} else {
			writeBytes([]byte(fmt.Sprintf("%T", payload.Cmd)))
		}
	} else {
		writeBytes(nil)
	}

	return base32.StdEncoding.EncodeToString(hasher.Sum(nil))
}
