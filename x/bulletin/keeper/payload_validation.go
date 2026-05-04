package keeper

import (
	"encoding/json"

	errorsmod "cosmossdk.io/errors"

	"github.com/sourcenetwork/sourcehub/x/bulletin/types"
)

type ringPayloadJSON struct {
	RingPK       *string   `json:"ring_pk"`
	NextPeerIDs  *[]string `json:"next_peer_ids,omitempty"`
	NewThreshold *uint32   `json:"new_threshold,omitempty"`
	PeerIDs      *[]string `json:"peer_ids"`
	Threshold    *uint32   `json:"threshold"`
	PSSInterval  *uint64   `json:"pss_interval,omitempty"`
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
	}

	return &ringPayload, nil
}

func finalizeRingPayloadReshare(currentPayload []byte) ([]byte, error) {
	currentRingPayload, err := parseRingPayloadJSON(currentPayload)
	if err != nil {
		return nil, err
	}

	if currentRingPayload.NextPeerIDs == nil {
		return nil, errorsmod.Wrap(types.ErrInvalidPostPayload, "invalid ring payload: missing next_peer_ids for reshare finalization")
	}
	if currentRingPayload.NewThreshold == nil {
		return nil, errorsmod.Wrap(types.ErrInvalidPostPayload, "invalid ring payload: missing new_threshold for reshare finalization")
	}

	currentRingPayload.PeerIDs = currentRingPayload.NextPeerIDs
	currentRingPayload.Threshold = currentRingPayload.NewThreshold
	currentRingPayload.NextPeerIDs = nil
	currentRingPayload.NewThreshold = nil

	finalizedPayload, err := json.Marshal(currentRingPayload)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrInvalidPostPayload, "could not finalize ring payload: %s", err)
	}

	return finalizedPayload, nil
}
