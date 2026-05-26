package keeper

import (
	"crypto/sha256"

	"github.com/sourcenetwork/sourcehub/x/orbis/types"
)

const RingReshareFinalizeSignDocDomain = "orbis-ring-reshare-finalize"

func ringReshareFinalizeSignBytes(
	chainID string,
	currentRing *types.Ring,
	currentRingBytes []byte,
	finalizedRingBytes []byte,
) ([]byte, error) {
	currentRingHash := sha256.Sum256(currentRingBytes)
	finalizedRingHash := sha256.Sum256(finalizedRingBytes)

	signDoc := types.RingReshareFinalizeSignDoc{
		Domain:              RingReshareFinalizeSignDocDomain,
		ChainId:             chainID,
		RingId:              currentRing.Id,
		RingPk:              currentRing.RingPk,
		CurrentRingSha256:   currentRingHash[:],
		FinalizedRingSha256: finalizedRingHash[:],
		BlockNumberNonce:    currentRing.BlockNumberNonce,
	}

	return signDoc.Marshal()
}
