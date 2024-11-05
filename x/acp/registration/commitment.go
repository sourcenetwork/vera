package registration

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
	err := proof.Verify(root, generateLeafValue(policyId, actor, opening.Object))
	if err != nil {
		return false, nil
	}
	return true, nil
}

func GenerateCommitmentWithoutValidation(policyId string, actor *coretypes.Actor, objs []*coretypes.Object) ([]byte, error) {
	t, err := NewObjectCommitmentTree(policyId, actor, objs)
	if err != nil {
		return nil, err
	}
	return t.GetCommitment(), nil
}

func ProofForObject(policyId string, actor *coretypes.Actor, idx int, objs []*coretypes.Object) (*types.RegistrationProof, error) {
	t, err := NewObjectCommitmentTree(policyId, actor, objs)
	if err != nil {
		return nil, err
	}
	return t.GetProofForIdx(idx)
}

// generateLeafValue produces a byte slice representing an individual object registration
// which will be commited to.
func generateLeafValue(policyId string, actor *coretypes.Actor, o *coretypes.Object) []byte {
	return []byte(policyId + ":" + o.Resource + ":" + o.Id + ":" + actor.Id)
}

// produceNodeHash hashes a Mertle Tree leaf as per RFC 6962
// https://www.rfc-editor.org/rfc/rfc6962#section-2.1
func produceLeafHash(policyId string, actor *coretypes.Actor, o *coretypes.Object) []byte {
	merkleVal := generateLeafValue(policyId, actor, o)
	hasher := sha256.New()
	hasher.Write([]byte{leafPrefix})
	hasher.Write(merkleVal)
	return hasher.Sum(nil)
}

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

type RegistrationCommitmentTree struct {
	policyId   string
	actor      *coretypes.Actor
	objs       []*coretypes.Object
	commitment []byte
	leaves     [][]byte
}

func (t *RegistrationCommitmentTree) genCommitment() {
	t.leaves = utils.MapSlice(t.objs, func(o *coretypes.Object) []byte {
		return generateLeafValue(t.policyId, t.actor, o)
	})
	t.commitment = merkle.HashFromByteSlices(t.leaves)
}

func (t *RegistrationCommitmentTree) GetCommitment() []byte {
	return t.commitment
}

func (t *RegistrationCommitmentTree) GetProofForObj(obj *coretypes.Object) (*types.RegistrationProof, error) {
	idx, err := t.findIdx(obj)
	if err != nil {
		return nil, err
	}
	return t.proofForIdx(idx)
}

func (t *RegistrationCommitmentTree) GetProofForIdx(idx int) (*types.RegistrationProof, error) {
	return t.proofForIdx(idx)
}

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
