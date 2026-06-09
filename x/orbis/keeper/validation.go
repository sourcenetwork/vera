package keeper

import (
	"encoding/hex"
	"strings"

	errorsmod "cosmossdk.io/errors"
	"github.com/sourcenetwork/immutable"

	"github.com/sourcenetwork/sourcehub/x/orbis/types"
)

const MinRingUpgradeLeadBlocks int64 = 100

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
	if err := validateUpgradeInfo(&ring.UpgradeInfo); err != nil {
		return err
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

func validateRingUpdate(
	newPeerNodeKeys []string,
	newThreshold immutable.Option[uint32],
	nextVersion immutable.Option[uint64],
	activationHeight immutable.Option[int64],
	clearUpgrade bool,
	currentHeight int64,
	existing *types.Ring,
) error {
	reshareInProgress := len(existing.NewPeerNodeKeys) > 0 || existing.XNewThreshold != nil
	touchingReshareFields := len(newPeerNodeKeys) > 0 || newThreshold.HasValue()
	if reshareInProgress && touchingReshareFields {
		return types.ErrReshareInProgress
	}

	if len(newPeerNodeKeys) > 0 {
		if err := validateUniquePeerNodeKeys(newPeerNodeKeys); err != nil {
			return err
		}
	}
	if newThreshold.HasValue() && newThreshold.Value() < 1 {
		return errorsmod.Wrap(types.ErrInvalidRing, "new_threshold must be at least 1")
	}
	if len(newPeerNodeKeys) == 0 && newThreshold.HasValue() && newThreshold.Value() > uint32(len(existing.PeerNodeKeys)) {
		return errorsmod.Wrapf(
			types.ErrInvalidRing,
			"new_threshold (%d) cannot exceed existing committee size (%d)",
			newThreshold.Value(),
			len(existing.PeerNodeKeys),
		)
	}
	if len(newPeerNodeKeys) > 0 {
		targetThreshold := existing.Threshold
		if newThreshold.HasValue() {
			targetThreshold = newThreshold.Value()
		}
		if err := validateEffectiveReshareThreshold(targetThreshold, len(newPeerNodeKeys)); err != nil {
			return err
		}
	}
	if nextVersion.HasValue() != activationHeight.HasValue() {
		return errorsmod.Wrap(types.ErrInvalidRing, "next_version and activation_height must be supplied together")
	}
	if clearUpgrade && nextVersion.HasValue() {
		return errorsmod.Wrap(types.ErrInvalidRing, "clear_upgrade cannot be combined with a new upgrade schedule")
	}
	if nextVersion.HasValue() {
		if nextVersion.Value() <= existing.UpgradeInfo.CurrentVersion {
			return errorsmod.Wrapf(
				types.ErrInvalidRing,
				"next_version (%d) must be greater than current_version (%d)",
				nextVersion.Value(),
				existing.UpgradeInfo.CurrentVersion,
			)
		}
		minimumActivationHeight := currentHeight + MinRingUpgradeLeadBlocks
		if activationHeight.Value() < minimumActivationHeight {
			return errorsmod.Wrapf(
				types.ErrInvalidRing,
				"activation_height (%d) must be at least %d",
				activationHeight.Value(),
				minimumActivationHeight,
			)
		}
	}

	return nil
}

func validateUpgradeInfo(info *types.UpgradeInfo) error {
	if (info.XNextVersion == nil) != (info.XActivationHeight == nil) {
		return errorsmod.Wrap(types.ErrInvalidRing, "upgrade next_version and activation_height must both be set or both be absent")
	}
	if info.XNextVersion != nil {
		if info.GetNextVersion() <= info.CurrentVersion {
			return errorsmod.Wrap(types.ErrInvalidRing, "upgrade next_version must be greater than current_version")
		}
		if info.GetActivationHeight() <= 0 {
			return errorsmod.Wrap(types.ErrInvalidRing, "upgrade activation_height must be positive")
		}
	}
	return nil
}

func normalizeRingUpgrade(ring *types.Ring, currentHeight int64) (bool, uint64, int64) {
	if ring.UpgradeInfo.XNextVersion == nil || ring.UpgradeInfo.XActivationHeight == nil {
		return false, 0, 0
	}
	activationHeight := ring.UpgradeInfo.GetActivationHeight()
	if currentHeight < activationHeight {
		return false, 0, 0
	}
	previousVersion := ring.UpgradeInfo.CurrentVersion
	ring.UpgradeInfo.CurrentVersion = ring.UpgradeInfo.GetNextVersion()
	clearRingUpgrade(ring)
	return true, previousVersion, activationHeight
}

func clearRingUpgrade(ring *types.Ring) {
	ring.UpgradeInfo.XNextVersion = nil
	ring.UpgradeInfo.XActivationHeight = nil
}

func setRingUpgrade(ring *types.Ring, nextVersion uint64, activationHeight int64) {
	ring.UpgradeInfo.XNextVersion = &types.UpgradeInfo_NextVersion{NextVersion: nextVersion}
	ring.UpgradeInfo.XActivationHeight = &types.UpgradeInfo_ActivationHeight{ActivationHeight: activationHeight}
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
	keyHex := strings.ToLower(strings.TrimPrefix(nodeInfo.ControllerKey, "0x"))
	decoded, err := hex.DecodeString(keyHex)
	if err != nil || len(decoded) != compressedPubKeyLen {
		return errorsmod.Wrap(types.ErrInvalidNodeInfo, "invalid controller_key encoding")
	}
	nodeInfo.ControllerKey = keyHex
	return nil
}
