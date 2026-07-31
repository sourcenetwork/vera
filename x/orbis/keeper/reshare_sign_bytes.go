package keeper

import (
	"crypto/sha256"
	"slices"

	"github.com/sourcenetwork/sourcehub/x/orbis/types"
)

const RingReshareFinalizeSignDocDomain = "orbis-ring-reshare-finalize"

func ringReshareFinalizeSignBytes(
	chainID string,
	currentRing *types.Ring,
	finalizedRing *types.Ring,
) ([]byte, error) {
	currentRingHash, err := ringReshareSignStateHash(currentRing)
	if err != nil {
		return nil, err
	}
	finalizedRingHash, err := ringReshareSignStateHash(finalizedRing)
	if err != nil {
		return nil, err
	}

	signDoc := types.RingReshareFinalizeSignDoc{
		Domain:              RingReshareFinalizeSignDocDomain,
		ChainId:             chainID,
		RingId:              currentRing.Id,
		RingPk:              currentRing.RingPk,
		CurrentRingSha256:   currentRingHash,
		FinalizedRingSha256: finalizedRingHash,
		BlockNumberNonce:    currentRing.BlockNumberNonce,
	}

	return signDoc.Marshal()
}

func ringReshareSignStateHash(ring *types.Ring) ([]byte, error) {
	signState := ringReshareSignState(ring)
	signStateBytes, err := signState.Marshal()
	if err != nil {
		return nil, err
	}

	hash := sha256.Sum256(signStateBytes)
	return hash[:], nil
}

func ringReshareSignState(ring *types.Ring) types.RingReshareSignState {
	signState := types.RingReshareSignState{
		RingPk:           ring.RingPk,
		PeerNodeKeys:     slices.Clone(ring.PeerNodeKeys),
		Threshold:        ring.Threshold,
		NewPeerNodeKeys:  slices.Clone(ring.NewPeerNodeKeys),
		BlockNumberNonce: ring.BlockNumberNonce,
		PolicyId:         ring.PolicyId,
	}
	if ring.XNewThreshold != nil {
		signState.XNewThreshold = &types.RingReshareSignState_NewThreshold{
			NewThreshold: ring.GetNewThreshold(),
		}
	}

	return signState
}
