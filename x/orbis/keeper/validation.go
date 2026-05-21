package keeper

import (
	errorsmod "cosmossdk.io/errors"

	"github.com/sourcenetwork/sourcehub/x/orbis/types"
)

func validateRing(ring *types.Ring) error {
	switch {
	case ring.Id == "":
		return types.ErrInvalidRingId
	case ring.Namespace == "":
		return types.ErrInvalidNamespaceId
	case ring.RingPk == "":
		return errorsmod.Wrap(types.ErrInvalidRing, "missing ring_pk")
	case len(ring.PeerIds) == 0:
		return errorsmod.Wrap(types.ErrInvalidRing, "missing peer_ids")
	case ring.Threshold == 0 || int(ring.Threshold) > len(ring.PeerIds):
		return errorsmod.Wrapf(types.ErrInvalidRing, "threshold %d is invalid for committee size %d", ring.Threshold, len(ring.PeerIds))
	}

	if err := validateUniquePeerIDs(ring.PeerIds, "peer_ids"); err != nil {
		return err
	}
	if len(ring.NewPeerIds) > 0 {
		if err := validateUniquePeerIDs(ring.NewPeerIds, "new_peer_ids"); err != nil {
			return err
		}
	}
	if ring.XNewThreshold != nil && ring.GetNewThreshold() == 0 {
		return errorsmod.Wrap(types.ErrInvalidRing, "new_threshold must be at least 1")
	}
	if len(ring.NewPeerIds) > 0 && ring.XNewThreshold != nil && uint32(len(ring.NewPeerIds)) < ring.GetNewThreshold() {
		return errorsmod.Wrapf(
			types.ErrInvalidRing,
			"new_peer_ids count (%d) is less than new_threshold (%d)",
			len(ring.NewPeerIds),
			ring.GetNewThreshold(),
		)
	}

	return nil
}

func validateRingUpdate(newPeerIDs []string, newThreshold *uint32, existing *types.Ring) error {
	reshareInProgress := len(existing.NewPeerIds) > 0 || existing.XNewThreshold != nil
	touchingReshareFields := len(newPeerIDs) > 0 || newThreshold != nil
	if reshareInProgress && touchingReshareFields {
		return types.ErrReshareInProgress
	}

	if len(newPeerIDs) > 0 {
		if err := validateUniquePeerIDs(newPeerIDs, "new_peer_ids"); err != nil {
			return err
		}
	}
	if newThreshold != nil && *newThreshold < 1 {
		return errorsmod.Wrap(types.ErrInvalidRing, "new_threshold must be at least 1")
	}
	if len(newPeerIDs) > 0 && newThreshold != nil && uint32(len(newPeerIDs)) < *newThreshold {
		return errorsmod.Wrapf(
			types.ErrInvalidRing,
			"new_peer_ids count (%d) is less than new_threshold (%d)",
			len(newPeerIDs),
			*newThreshold,
		)
	}

	return nil
}

func validateUniquePeerIDs(peerIDs []string, fieldName string) error {
	seen := make(map[string]struct{}, len(peerIDs))
	for _, id := range peerIDs {
		if id == "" {
			return errorsmod.Wrapf(types.ErrInvalidRing, "empty peer id in %s", fieldName)
		}
		if _, dup := seen[id]; dup {
			return errorsmod.Wrapf(types.ErrInvalidRing, "duplicate peer id in %s: %q", fieldName, id)
		}
		seen[id] = struct{}{}
	}
	return nil
}

func ringForReshareFinalization(currentRing *types.Ring) (*types.Ring, error) {
	finalized := *currentRing
	if len(finalized.NewPeerIds) == 0 && finalized.XNewThreshold == nil {
		return nil, errorsmod.Wrap(types.ErrInvalidRing, "missing new_peer_ids or new_threshold for reshare finalization")
	}

	if len(finalized.NewPeerIds) > 0 {
		finalized.PeerIds = append([]string(nil), finalized.NewPeerIds...)
	}
	if finalized.XNewThreshold != nil {
		finalized.Threshold = finalized.GetNewThreshold()
	}
	finalized.NewPeerIds = nil
	finalized.XNewThreshold = nil

	if err := validateRing(&finalized); err != nil {
		return nil, err
	}

	return &finalized, nil
}

func validateDocument(document *types.Document) error {
	switch {
	case document.Id == "":
		return types.ErrInvalidDocumentId
	case document.Namespace == "":
		return types.ErrInvalidNamespaceId
	case document.RingId == "":
		return types.ErrInvalidRingId
	case document.Document == "":
		return errorsmod.Wrap(types.ErrInvalidDocument, "missing document")
	case document.Proof == "":
		return errorsmod.Wrap(types.ErrInvalidDocument, "missing proof")
	case document.PolicyId == "" || document.Resource == "" || document.Permission == "":
		return errorsmod.Wrap(types.ErrInvalidDocument, "missing policy binding")
	}
	return nil
}

func validateKeyDerivation(keyDerivation *types.KeyDerivation) error {
	switch {
	case keyDerivation.Id == "":
		return types.ErrInvalidKeyDerivationId
	case keyDerivation.Namespace == "":
		return types.ErrInvalidNamespaceId
	case keyDerivation.RingId == "":
		return types.ErrInvalidRingId
	case keyDerivation.Derivation == "":
		return errorsmod.Wrap(types.ErrInvalidKeyDerivation, "missing derivation")
	case keyDerivation.PolicyId == "" || keyDerivation.Resource == "" || keyDerivation.Permission == "":
		return errorsmod.Wrap(types.ErrInvalidKeyDerivation, "missing policy binding")
	}
	return nil
}
