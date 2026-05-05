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

func ringPayloadForReshareFinalization(currentPayload []byte) (*ringPayloadJSON, error) {
	currentRingPayload, err := parseRingPayloadJSON(currentPayload)
	if err != nil {
		return nil, err
	}

	if currentRingPayload.NewPeerIDs == nil && currentRingPayload.NewThreshold == nil {
		return nil, errorsmod.Wrap(types.ErrInvalidPostPayload, "invalid ring payload: missing new_peer_ids or new_threshold for reshare finalization")
	}

	if currentRingPayload.NewPeerIDs != nil {
		currentRingPayload.PeerIDs = currentRingPayload.NewPeerIDs
	}
	if currentRingPayload.NewThreshold != nil {
		currentRingPayload.Threshold = currentRingPayload.NewThreshold
	}
	currentRingPayload.NewPeerIDs = nil
	currentRingPayload.NewThreshold = nil

	return currentRingPayload, nil
}

func deriveFinalizedRingPayloadReshare(currentPayload []byte) ([]byte, error) {
	currentRingPayload, err := ringPayloadForReshareFinalization(currentPayload)
	if err != nil {
		return nil, err
	}

	finalizedPayload, err := json.Marshal(currentRingPayload)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrInvalidPostPayload, "could not finalize ring payload: %s", err)
	}

	return finalizedPayload, nil
}

func finalizeRingPayloadReshare(currentPayload []byte, blockNumberNonce uint64) ([]byte, error) {
	currentRingPayload, err := ringPayloadForReshareFinalization(currentPayload)
	if err != nil {
		return nil, err
	}

	currentRingPayload.BlockNumberNonce = &blockNumberNonce

	finalizedPayload, err := json.Marshal(currentRingPayload)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrInvalidPostPayload, "could not finalize ring payload: %s", err)
	}

	return finalizedPayload, nil
}
