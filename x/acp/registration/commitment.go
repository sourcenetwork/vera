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

func VerifyProof(root []byte, policyId string, actor *coretypes.Actor, opening *types.RegistrationProof) bool {
	leafHash := produceLeafHash(policyId, actor, opening.Object)
	computedRoot := foldl(opening.MerkleProof, leafHash, func(proofHash, acc []byte) []byte {
		return produceNodeHash(acc, proofHash)
	})
	if !slices.Equal(computedRoot, root) {
		return false
	}
	return true
}

func GenerateCommitment(policyId string, actor *coretypes.Actor, objs []*coretypes.Object) []byte {
	keys := generateByteSlice(policyId, actor, objs)
	return merkle.HashFromByteSlices(keys)
}

func generateByteSlice(policyId string, actor *coretypes.Actor, objs []*coretypes.Object) [][]byte {
	sortable := utils.FromExtractor(objs, func(o *coretypes.Object) string {
		return o.Resource + ":" + o.Id
	})
	objs = sortable.Sort()
	keys := utils.MapSlice(objs, func(o *coretypes.Object) []byte {
		return []byte(policyId + ":" + o.Resource + ":" + o.Id + ":" + actor.Id)
	})
	return keys
}

func ProofForObject(policyId string, actor *coretypes.Actor, idx uint64, objs []*coretypes.Object) *types.RegistrationProof {
	keys := generateByteSlice(policyId, actor, objs)
	_, proofs := merkle.ProofsFromByteSlices(keys)
	return &types.RegistrationProof{
		MerkleProof: proofs[idx].Aunts,
	}
}

type objectTree struct {
	polId      string
	actor      *coretypes.Actor
	objs       []*coretypes.Object
	merkleVals [][]byte
}

func (t *objectTree) ProofForObject(obj *coretypes.Object) (*types.RegistrationProof, error) {
	idx, err := t.findObj(obj)
	if err != nil {
		return nil, err
	}
	return t.Proof(idx)
}

func (t *objectTree) Proof(idx uint) (*types.RegistrationProof, error) {
	if idx >= uint(len(t.objs)) {
		return nil, errors.Wrap("index out of bounds:", errors.ErrorType_BAD_INPUT)
	}

	_, proofs := merkle.ProofsFromByteSlices(t.merkleVals)
	return &types.RegistrationProof{
		MerkleProof: proofs[idx].Aunts,
		Object:      t.objs[idx],
	}, nil
}

func (t *objectTree) findObj(obj *coretypes.Object) (uint, error) {
	idx := slices.IndexFunc(t.objs, func(o *coretypes.Object) bool {
		return o.Resource == obj.Resource && o.Id == obj.Id
	})
	if idx == -1 {
		return 0, errors.Wrap("proof does not contain object", errors.ErrorType_BAD_INPUT,
			errors.Pair("resource", obj.Resource),
			errors.Pair("obj", obj.Id))
	}
	return uint(idx), nil
}

func newObjectTree(policyId string, actor *coretypes.Actor, obj *coretypes.Object, objs []*coretypes.Object) objectTree {
	sortable := objsToSortable(objs)
	sorted := sortable.Sort()

	merkleVals := utils.MapSlice(sorted, func(o *coretypes.Object) []byte {
		return generateCommitmentValue(policyId, actor, o)
	})

	return objectTree{
		polId:      policyId,
		actor:      actor,
		objs:       sorted,
		merkleVals: merkleVals,
	}
}

// objsToStorable returns a Sortable which alphabetically sorts objects by resource:obj_id
// this ensures the determinism of generating commitments.
func objsToSortable(objs []*coretypes.Object) utils.Sortable[*coretypes.Object] {
	return utils.FromExtractor(objs, func(o *coretypes.Object) string {
		return o.Resource + ":" + o.Id
	})
}

// generateCommitmentValue produces a byte slice representing an individual object registration
// which will be commited to.
func generateCommitmentValue(policyId string, actor *coretypes.Actor, o *coretypes.Object) []byte {
	return []byte(policyId + ":" + o.Resource + ":" + o.Id + ":" + actor.Id)
}

// foldl folds / reduces a slice starting from index 0
func foldl[T, U any](ts []T, acc U, ff func(T, U) U) U {
	for _, t := range ts {
		acc = ff(t, acc)
	}
	return acc
}

// produceNodeHash hashes a Mertle Tree leaf as per RFC 6962
// https://www.rfc-editor.org/rfc/rfc6962#section-2.1
func produceLeafHash(policyId string, actor *coretypes.Actor, o *coretypes.Object) []byte {
	merkleVal := generateCommitmentValue(policyId, actor, o)
	hasher := sha256.New()
	hasher.Write([]byte{leafPrefix})
	hasher.Write(merkleVal)
	return hasher.Sum(nil)
}

// produceNodeHash hashes an inner node of a binary Merkle Tree as per RFC 6962
// https://www.rfc-editor.org/rfc/rfc6962#section-2.1
func produceNodeHash(h1, h2 []byte) []byte {
	hasher := sha256.New()
	hasher.Write([]byte{nodePrefix})
	hasher.Write(h1)
	hasher.Write(h2)
	return hasher.Sum(nil)
}
