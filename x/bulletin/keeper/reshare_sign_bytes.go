package keeper

import (
	"crypto/sha256"

	"github.com/sourcenetwork/sourcehub/x/bulletin/types"
)

const RingReshareFinalizeSignDocDomain = "orbis-ring-reshare-finalize"

func ringReshareFinalizeSignBytes(
	chainID string,
	namespace string,
	postID string,
	currentPayload []byte,
	finalizedPayload []byte,
) ([]byte, error) {
	currentRingPayload, err := parseRingPayloadJSON(currentPayload)
	if err != nil {
		return nil, err
	}

	currentPayloadHash := sha256.Sum256(currentPayload)
	finalizedPayloadHash := sha256.Sum256(finalizedPayload)

	signDoc := types.RingReshareFinalizeSignDoc{
		Domain:                 RingReshareFinalizeSignDocDomain,
		ChainId:                chainID,
		Namespace:              namespace,
		PostId:                 postID,
		RingPk:                 *currentRingPayload.RingPK,
		CurrentPayloadSha256:   currentPayloadHash[:],
		FinalizedPayloadSha256: finalizedPayloadHash[:],
		BlockNumberNonce:       *currentRingPayload.BlockNumberNonce,
	}

	return signDoc.Marshal()
}
