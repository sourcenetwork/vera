package keeper

import (
	"encoding/json"

	errorsmod "cosmossdk.io/errors"

	"github.com/sourcenetwork/sourcehub/x/bulletin/types"
)

type ringPayloadJSON struct {
	RingPK           *string   `json:"ring_pk"`
	NewPeerIDs       *[]string `json:"new_peer_ids,omitempty"`
	NewThreshold     *uint32   `json:"new_threshold,omitempty"`
	PeerIDs          *[]string `json:"peer_ids"`
	Threshold        *uint32   `json:"threshold"`
	PSSInterval      *uint64   `json:"pss_interval,omitempty"`
	BlockNumberNonce *uint64   `json:"block_number_nonce"`
	PolicyID         *string   `json:"policy_id,omitempty"`
}

func validateRingPayloadJSON(payload []byte) error {
	_, err := parseRingPayloadJSON(payload)
	return err
}

func parseRingPayloadJSON(payload []byte) (*ringPayloadJSON, error) {
	var ringPayload ringPayloadJSON
	if err := json.Unmarshal(payload, &ringPayload); err != nil {
		return nil, errorsmod.Wrapf(types.ErrInvalidPostPayload, "invalid ring payload JSON: %s", err)
	}

	switch {
	case ringPayload.RingPK == nil:
		return nil, errorsmod.Wrap(types.ErrInvalidPostPayload, "invalid ring payload: missing ring_pk")
	case ringPayload.PeerIDs == nil:
		return nil, errorsmod.Wrap(types.ErrInvalidPostPayload, "invalid ring payload: missing peer_ids")
	case ringPayload.Threshold == nil:
		return nil, errorsmod.Wrap(types.ErrInvalidPostPayload, "invalid ring payload: missing threshold")
	case ringPayload.BlockNumberNonce == nil:
		return nil, errorsmod.Wrap(types.ErrInvalidPostPayload, "invalid ring payload: missing block_number_nonce")
	}

	return &ringPayload, nil
}

// validateRingPostUpdate checks that the fields being set via UpdateRingPostByAcp
// are internally consistent: new_peer_ids must have no duplicates, and
// new_threshold (if provided) must be at least 1.
func validateRingPostUpdate(newPeerIDs []string, newThreshold *uint32) error {
	if len(newPeerIDs) > 0 {
		seen := make(map[string]struct{}, len(newPeerIDs))
		for _, id := range newPeerIDs {
			if _, dup := seen[id]; dup {
				return errorsmod.Wrapf(types.ErrInvalidPostPayload, "duplicate peer id in new_peer_ids: %q", id)
			}
			seen[id] = struct{}{}
		}
	}
	if newThreshold != nil && *newThreshold < 1 {
		return errorsmod.Wrap(types.ErrInvalidPostPayload, "new_threshold must be at least 1")
	}
	return nil
}

func ringPayloadForReshareFinalization(currentRingPayload *ringPayloadJSON) (*ringPayloadJSON, error) {
	finalizedRingPayload := *currentRingPayload
	if finalizedRingPayload.NewPeerIDs == nil && finalizedRingPayload.NewThreshold == nil {
		return nil, errorsmod.Wrap(types.ErrInvalidPostPayload, "invalid ring payload: missing new_peer_ids or new_threshold for reshare finalization")
	}

	if finalizedRingPayload.NewPeerIDs != nil {
		finalizedRingPayload.PeerIDs = finalizedRingPayload.NewPeerIDs
	}
	if finalizedRingPayload.NewThreshold != nil {
		finalizedRingPayload.Threshold = finalizedRingPayload.NewThreshold
	}
	finalizedRingPayload.NewPeerIDs = nil
	finalizedRingPayload.NewThreshold = nil

	return &finalizedRingPayload, nil
}

func deriveFinalizedRingPayloadReshare(currentRingPayload *ringPayloadJSON) ([]byte, error) {
	finalizedRingPayload, err := ringPayloadForReshareFinalization(currentRingPayload)
	if err != nil {
		return nil, err
	}

	finalizedPayload, err := json.Marshal(finalizedRingPayload)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrInvalidPostPayload, "could not finalize ring payload: %s", err)
	}

	return finalizedPayload, nil
}

func finalizeRingPayloadReshare(currentRingPayload *ringPayloadJSON, blockNumberNonce uint64) ([]byte, error) {
	finalizedRingPayload, err := ringPayloadForReshareFinalization(currentRingPayload)
	if err != nil {
		return nil, err
	}

	finalizedRingPayload.BlockNumberNonce = &blockNumberNonce

	finalizedPayload, err := json.Marshal(finalizedRingPayload)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrInvalidPostPayload, "could not finalize ring payload: %s", err)
	}

	return finalizedPayload, nil
}
