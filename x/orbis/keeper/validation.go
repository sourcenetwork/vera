package keeper

import (
	"encoding/hex"
	"strings"

	errorsmod "cosmossdk.io/errors"

	"github.com/sourcenetwork/sourcehub/x/orbis/types"
)

func requireRingFinalized(ring *types.Ring) error {
	if ring.RingPk == "" {
		return types.ErrRingNotFinalized
	}
	return nil
}

func validateRing(ring *types.Ring) error {
	switch {
	case ring.Id == "":
		return types.ErrInvalidRingId
	case len(ring.PeerNodeKeys) == 0:
		return errorsmod.Wrap(types.ErrInvalidRing, "missing peer_node_keys")
	case ring.Threshold == 0 || int(ring.Threshold) > len(ring.PeerNodeKeys):
		return errorsmod.Wrapf(types.ErrInvalidRing, "threshold %d is invalid for committee size %d", ring.Threshold, len(ring.PeerNodeKeys))
	case ring.PolicyId == "":
		return errorsmod.Wrap(types.ErrInvalidRing, "missing policy_id")
	}

	if err := validateUniquePeerNodeKeys(ring.PeerNodeKeys); err != nil {
		return err
	}
	if len(ring.NewPeerNodeKeys) > 0 {
		if err := validateUniquePeerNodeKeys(ring.NewPeerNodeKeys); err != nil {
			return err
		}
	}
	if ring.XNewThreshold != nil && ring.GetNewThreshold() == 0 {
		return errorsmod.Wrap(types.ErrInvalidRing, "new_threshold must be at least 1")
	}
	if len(ring.NewPeerNodeKeys) > 0 || ring.XNewThreshold != nil {
		targetCommitteeSize := len(ring.PeerNodeKeys)
		if len(ring.NewPeerNodeKeys) > 0 {
			targetCommitteeSize = len(ring.NewPeerNodeKeys)
		}
		targetThreshold := ring.Threshold
		if ring.XNewThreshold != nil {
			targetThreshold = ring.GetNewThreshold()
		}
		if err := validateEffectiveReshareThreshold(targetThreshold, targetCommitteeSize); err != nil {
			return err
		}
	}

	return nil
}

func validateRingUpdate(newPeerNodeKeys []string, newThreshold *uint32, existing *types.Ring) error {
	reshareInProgress := len(existing.NewPeerNodeKeys) > 0 || existing.XNewThreshold != nil
	touchingReshareFields := len(newPeerNodeKeys) > 0 || newThreshold != nil
	if reshareInProgress && touchingReshareFields {
		return types.ErrReshareInProgress
	}

	if len(newPeerNodeKeys) > 0 {
		if err := validateUniquePeerNodeKeys(newPeerNodeKeys); err != nil {
			return err
		}
	}
	if newThreshold != nil && *newThreshold < 1 {
		return errorsmod.Wrap(types.ErrInvalidRing, "new_threshold must be at least 1")
	}
	if len(newPeerNodeKeys) == 0 && newThreshold != nil && *newThreshold > uint32(len(existing.PeerNodeKeys)) {
		return errorsmod.Wrapf(
			types.ErrInvalidRing,
			"new_threshold (%d) cannot exceed existing committee size (%d)",
			*newThreshold,
			len(existing.PeerNodeKeys),
		)
	}
	if len(newPeerNodeKeys) > 0 {
		targetThreshold := existing.Threshold
		if newThreshold != nil {
			targetThreshold = *newThreshold
		}
		if err := validateEffectiveReshareThreshold(targetThreshold, len(newPeerNodeKeys)); err != nil {
			return err
		}
	}

	return nil
}

func validateEffectiveReshareThreshold(threshold uint32, committeeSize int) error {
	if threshold > uint32(committeeSize) {
		return errorsmod.Wrapf(
			types.ErrInvalidRing,
			"effective new_threshold (%d) cannot exceed target committee size (%d)",
			threshold,
			committeeSize,
		)
	}
	return nil
}

// validateUniquePeerNodeKeys checks that each node key is a unique, validly-encoded
// 33-byte compressed secp256k1 public key.
func validateUniquePeerNodeKeys(nodeKeys []string) error {
	seen := make(map[string]struct{}, len(nodeKeys))
	for _, key := range nodeKeys {
		if err := validatePeerNodeKeyFormat(key); err != nil {
			return err
		}
		if _, dup := seen[key]; dup {
			return errorsmod.Wrapf(types.ErrInvalidRing, "duplicate peer_node_key: %q", key)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func validatePeerNodeKeyFormat(key string) error {
	if key == "" {
		return errorsmod.Wrap(types.ErrInvalidRing, "empty peer_node_key")
	}
	keyHex := strings.TrimPrefix(key, "0x")
	decoded, err := hex.DecodeString(keyHex)
	if err != nil || len(decoded) != compressedPubKeyLen {
		return errorsmod.Wrapf(types.ErrInvalidRing, "invalid peer_node_key encoding: %q", key)
	}
	return nil
}

func ringForReshareFinalization(currentRing *types.Ring) (*types.Ring, error) {
	finalized := *currentRing
	if len(finalized.NewPeerNodeKeys) == 0 && finalized.XNewThreshold == nil {
		return nil, errorsmod.Wrap(types.ErrInvalidRing, "missing new_peer_node_keys or new_threshold for reshare finalization")
	}

	if len(finalized.NewPeerNodeKeys) > 0 {
		finalized.PeerNodeKeys = append([]string(nil), finalized.NewPeerNodeKeys...)
	}
	if finalized.XNewThreshold != nil {
		finalized.Threshold = finalized.GetNewThreshold()
	}
	finalized.NewPeerNodeKeys = nil
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
	case keyDerivation.RingId == "":
		return types.ErrInvalidRingId
	case keyDerivation.Derivation == "":
		return errorsmod.Wrap(types.ErrInvalidKeyDerivation, "missing derivation")
	case keyDerivation.PolicyId == "" || keyDerivation.Resource == "" || keyDerivation.Permission == "":
		return errorsmod.Wrap(types.ErrInvalidKeyDerivation, "missing policy binding")
	}
	return nil
}

// secp256k1 compressed public key length in bytes
const compressedPubKeyLen = 33

func validateNodeInfo(nodeInfo *types.NodeInfo) error {
	switch {
	case nodeInfo.PeerId == "":
		return errorsmod.Wrap(types.ErrInvalidNodeInfo, "missing peer_id")
	case nodeInfo.ControllerKey == "":
		return errorsmod.Wrap(types.ErrInvalidNodeInfo, "missing controller_key")
	}
	keyHex := strings.TrimPrefix(nodeInfo.ControllerKey, "0x")
	decoded, err := hex.DecodeString(keyHex)
	if err != nil || len(decoded) != compressedPubKeyLen {
		return errorsmod.Wrap(types.ErrInvalidNodeInfo, "invalid controller_key encoding")
	}
	return nil
}
