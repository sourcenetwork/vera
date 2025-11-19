package commitment

import (
	"crypto/sha256"
	"slices"

	"github.com/tendermint/tendermint/crypto/merkle"

	"github.com/sourcenetwork/acp_core/pkg/errors"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/sourcenetwork/acp_core/pkg/utils"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

const leafPrefix byte = 0x00
const nodePrefix byte = 0x01

// VerifyProof consumes a root hash of a merkle tree, policy id, actor and an opening proof
// and verify whether the proof is a valid merkle tree proof.
//
// Returns true if the proof is valid, false if it's not or an error if the opening is malformed
func VerifyProof(root []byte, policyId string, actor *coretypes.Actor, opening *types.RegistrationProof) (bool, error) {
	if actor == nil || actor.Id == "" {
		return false, errors.Wrap("invalid actor", errors.ErrorType_BAD_INPUT)
	}
	if len(root) != sha256.Size {
		return false, errors.Wrap("invalid root commitment", errors.ErrorType_BAD_INPUT)
	}
	if opening == nil || opening.Object == nil || opening.Object.Id == "" ||
		opening.Object.Resource == "" {
		return false, errors.Wrap("invalid opening", errors.ErrorType_BAD_INPUT)
	}

	proof := merkle.Proof{
		Total:    int64(opening.LeafCount),
		Index:    int64(opening.LeafIndex),
		LeafHash: produceLeafHash(policyId, actor, opening.Object),
		Aunts:    opening.MerkleProof,
	}
	err := proof.Verify(root, GenerateLeafValue(policyId, actor, opening.Object))
	if err != nil {
		return false, nil
	}
	return true, nil
}

// GenerateCommitmentWithoutValidation generates a byte commitment (merkle root) of the given
// set of objects.
// It does not verify whether the given objects are registered or valid against the given policy.
func GenerateCommitmentWithoutValidation(policyId string, actor *coretypes.Actor, objs []*coretypes.Object) ([]byte, error) {
	t, err := NewObjectCommitmentTree(policyId, actor, objs)
	if err != nil {
		return nil, err
	}
	return t.GetCommitment(), nil
}

// ProofForObject generated an opening Proof for the given objects, actor and policy id
func ProofForObject(policyId string, actor *coretypes.Actor, idx int, objs []*coretypes.Object) (*types.RegistrationProof, error) {
	t, err := NewObjectCommitmentTree(policyId, actor, objs)
	if err != nil {
		return nil, err
	}
	return t.GetProofForIdx(idx)
}

// GenerateLeafValue produces a byte slice representing an individual object registration
// which will be commited to.
//
// The leaf value is the concatenation of PolicyId, object resource, object id and actor Id
func GenerateLeafValue(policyId string, actor *coretypes.Actor, o *coretypes.Object) []byte {
	return []byte(policyId + o.Resource + o.Id + actor.Id)
}

// produceNodeHash hashes a Mertle Tree leaf as per RFC 6962
// https://www.rfc-editor.org/rfc/rfc6962#section-2.1
func produceLeafHash(policyId string, actor *coretypes.Actor, o *coretypes.Object) []byte {
	merkleVal := GenerateLeafValue(policyId, actor, o)
	hasher := sha256.New()
	hasher.Write([]byte{leafPrefix})
	hasher.Write(merkleVal)
	return hasher.Sum(nil)
}

// NewObjectCommitmentTree returns a RegistrationCommitmentTree for the given objects
func NewObjectCommitmentTree(policyId string, actor *coretypes.Actor, objs []*coretypes.Object) (*RegistrationCommitmentTree, error) {
	if len(objs) == 0 {
		return nil, errors.Wrap("cannot generate commitment to empty object set", errors.ErrorType_BAD_INPUT)
	}
	tree := &RegistrationCommitmentTree{
		policyId: policyId,
		actor:    actor,
		objs:     objs,
	}
	tree.genCommitment()
	return tree, nil
}

// RegistrationCommitmentTree is a helper to generate openings and roots for a set of objects
type RegistrationCommitmentTree struct {
	policyId   string
	actor      *coretypes.Actor
	objs       []*coretypes.Object
	commitment []byte
	leaves     [][]byte
}

// genCommitment computes the merkle root
func (t *RegistrationCommitmentTree) genCommitment() {
	t.leaves = utils.MapSlice(t.objs, func(o *coretypes.Object) []byte {
		return GenerateLeafValue(t.policyId, t.actor, o)
	})
	t.commitment = merkle.HashFromByteSlices(t.leaves)
}

// GetCommitment returns the merkle tree for the tree
func (t *RegistrationCommitmentTree) GetCommitment() []byte {
	return t.commitment
}

// GetProofForObject returns an opening proof for the given object
// If the object was not included in the tree, errors
func (t *RegistrationCommitmentTree) GetProofForObj(obj *coretypes.Object) (*types.RegistrationProof, error) {
	idx, err := t.findIdx(obj)
	if err != nil {
		return nil, err
	}
	return t.proofForIdx(idx)
}

// GetProofForIdx returns an opening proof for the i-th object
func (t *RegistrationCommitmentTree) GetProofForIdx(i int) (*types.RegistrationProof, error) {
	return t.proofForIdx(i)
}

// proofForIdx returns the RegistrationProof for object at idx
func (t *RegistrationCommitmentTree) proofForIdx(idx int) (*types.RegistrationProof, error) {
	if idx >= len(t.objs) || idx < 0 {
		return nil, errors.Wrap("index out of bounds:", errors.ErrorType_BAD_INPUT)
	}

	_, proofs := merkle.ProofsFromByteSlices(t.leaves)
	return &types.RegistrationProof{
		MerkleProof: proofs[idx].Aunts,
		Object:      t.objs[idx],
		LeafCount:   uint64(len(t.objs)),
		LeafIndex:   uint64(idx),
	}, nil
}

// finxIdx looks up the idx of obj in the current tree
func (t *RegistrationCommitmentTree) findIdx(obj *coretypes.Object) (int, error) {
	i := slices.IndexFunc(t.objs, func(o *coretypes.Object) bool {
		return objEq(obj, o)
	})
	if i == -1 {
		return 0, errors.Wrap("proof does not contain object", errors.ErrorType_BAD_INPUT)
	}

	return i, nil
}

func objEq(a, b *coretypes.Object) bool {
	return a.Id == b.Id && a.Resource == b.Resource
}
