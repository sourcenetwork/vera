package signed_policy_cmd

import (
	"testing"

	"github.com/stretchr/testify/require"

	gogotypes "github.com/cosmos/gogoproto/types"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

// buildTestPayload builds a minimal payload with defaults.
func buildTestPayload(t *testing.T, did string) (payload types.SignedPolicyCmdPayload) {
	t.Helper()
	payload = types.SignedPolicyCmdPayload{
		Actor:           did,
		IssuedHeight:    10,
		ExpirationDelta: 100,
		PolicyId:        "policy-1",
		Cmd:             &types.PolicyCmd{},
	}
	return
}

func TestComputePayloadID_StableAndUnique(t *testing.T) {
	actor := "did:key:test-actor"
	payloadA := buildTestPayload(t, actor)
	payloadB := buildTestPayload(t, actor)
	payloadB.IssuedHeight++
	idA1 := ComputePayloadID(&payloadA)
	idA2 := ComputePayloadID(&payloadA)
	idB := ComputePayloadID(&payloadB)
	require.Equal(t, idA1, idA2)
	require.NotEqual(t, idA1, idB)
}

func TestComputePayloadID_ActorAffectsID(t *testing.T) {
	p1 := buildTestPayload(t, "did:key:alice")
	p2 := buildTestPayload(t, "did:key:bob")
	id1 := ComputePayloadID(&p1)
	id2 := ComputePayloadID(&p2)
	require.NotEqual(t, id1, id2)
}

func TestComputePayloadID_PolicyIDAffectsID(t *testing.T) {
	p1 := buildTestPayload(t, "did:key:actor")
	p2 := buildTestPayload(t, "did:key:actor")
	p2.PolicyId = "policy-2"
	id1 := ComputePayloadID(&p1)
	id2 := ComputePayloadID(&p2)
	require.NotEqual(t, id1, id2)
}

func TestComputePayloadID_ExpirationDeltaAffectsID(t *testing.T) {
	p1 := buildTestPayload(t, "did:key:actor")
	p2 := buildTestPayload(t, "did:key:actor")
	p2.ExpirationDelta++
	id1 := ComputePayloadID(&p1)
	id2 := ComputePayloadID(&p2)
	require.NotEqual(t, id1, id2)
}

func TestComputePayloadID_CommandContentAffectsID(t *testing.T) {
	p1 := buildTestPayload(t, "did:key:actor")
	p2 := buildTestPayload(t, "did:key:actor")
	p1.Cmd = types.NewRegisterObjectCmd(coretypes.NewObject("file", "foo"))
	p2.Cmd = types.NewRegisterObjectCmd(coretypes.NewObject("file", "bar"))
	id1 := ComputePayloadID(&p1)
	id2 := ComputePayloadID(&p2)
	require.NotEqual(t, id1, id2)
}

func TestComputePayloadID_IssuedAtDoesNotAffectID(t *testing.T) {
	p1 := buildTestPayload(t, "did:key:actor")
	p2 := buildTestPayload(t, "did:key:actor")
	p1.IssuedAt = &gogotypes.Timestamp{Seconds: 1234567890}
	p2.IssuedAt = &gogotypes.Timestamp{Seconds: 1234567899}
	id1 := ComputePayloadID(&p1)
	id2 := ComputePayloadID(&p2)
	require.Equal(t, id1, id2)
}
