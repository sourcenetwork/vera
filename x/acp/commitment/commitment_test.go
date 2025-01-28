package commitment

import (
	"testing"

	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/stretchr/testify/require"
)

const testPolicyID = "095d86e559a22cef11c7aa7240560b9fd3acf8657fc3743fb16c271a854171e3"

// testCorrectness is a minimal test case which given setup data,
// asserts that generating a commitment, a proof for an object within it
// and finally verifying the proof is a valid operation.
func testCorrectness(t *testing.T, policyId string, idx int, objs []*coretypes.Object, actor *coretypes.Actor) {
	commitment, err := GenerateCommitmentWithoutValidation(policyId, actor, objs)
	require.NoError(t, err)

	proof, err := ProofForObject(policyId, actor, idx, objs)
	require.NoError(t, err)

	ok, err := VerifyProof(commitment, policyId, actor, proof)
	require.NoError(t, err)
	require.True(t, ok)
}

func TestCommitmentEvenObjects(t *testing.T) {
	actor := coretypes.NewActor("did:example:bob")
	objs := []*coretypes.Object{
		coretypes.NewObject("file", "foo"),
		coretypes.NewObject("file", "tester"),
	}
	for idx, obj := range objs {
		t.Run(obj.String(), func(t *testing.T) {
			testCorrectness(t, testPolicyID, idx, objs, actor)
		})
	}
}

func TestCommitmentOddObjects(t *testing.T) {
	actor := coretypes.NewActor("did:example:bob")
	objs := []*coretypes.Object{
		coretypes.NewObject("file", "bar"),
		coretypes.NewObject("file", "deez"),
		coretypes.NewObject("file", "zaz"),
	}
	for idx, obj := range objs {
		t.Run(obj.String(), func(t *testing.T) {
			testCorrectness(t, testPolicyID, idx, objs, actor)
		})
	}
}

func TestCommitmentSingleObject(t *testing.T) {
	actor := coretypes.NewActor("did:example:bob")
	objs := []*coretypes.Object{
		coretypes.NewObject("file", "foo"),
	}
	testCorrectness(t, testPolicyID, 0, objs, actor)
}
