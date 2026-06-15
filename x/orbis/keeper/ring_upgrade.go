package keeper

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/sourcenetwork/sourcehub/x/orbis/types"
)

const MinRingUpgradeLeadSeconds uint64 = 600

func currentBlockUnixTime(ctx sdk.Context) (uint64, error) {
	blockUnixTime := ctx.BlockTime().Unix()
	if blockUnixTime < 0 {
		return 0, errorsmod.Wrap(types.ErrInvalidRing, "current block time is before the Unix epoch")
	}
	return uint64(blockUnixTime), nil
}

func normalizeMaturedRingUpgrade(ring *types.Ring, currentTime uint64) *types.EventRingUpgradeNormalized {
	if ring.UpgradeInfo.XNextVersion == nil || ring.UpgradeInfo.XActivationTime == nil {
		return nil
	}
	activationTime := ring.UpgradeInfo.GetActivationTime()
	if currentTime < activationTime {
		return nil
	}

	previousVersion := ring.UpgradeInfo.CurrentVersion
	ring.UpgradeInfo.CurrentVersion = ring.UpgradeInfo.GetNextVersion()
	clearRingUpgrade(ring)

	return &types.EventRingUpgradeNormalized{
		RingId:          ring.Id,
		PreviousVersion: previousVersion,
		CurrentVersion:  ring.UpgradeInfo.CurrentVersion,
		ActivationTime:  activationTime,
	}
}

func validateRingUpgradeSchedule(nextVersion, activationTime, currentTime uint64, existing *types.Ring) error {
	if nextVersion <= existing.UpgradeInfo.CurrentVersion {
		return errorsmod.Wrapf(
			types.ErrInvalidRing,
			"next_version (%d) must be greater than current_version (%d)",
			nextVersion,
			existing.UpgradeInfo.CurrentVersion,
		)
	}
	if currentTime > ^uint64(0)-MinRingUpgradeLeadSeconds {
		return errorsmod.Wrap(types.ErrInvalidRing, "current block time is too large to schedule an upgrade")
	}
	minimumActivationTime := currentTime + MinRingUpgradeLeadSeconds
	if activationTime < minimumActivationTime {
		return errorsmod.Wrapf(
			types.ErrInvalidRing,
			"activation_time (%d) must be at least %d",
			activationTime,
			minimumActivationTime,
		)
	}
	return nil
}

func validateUpgradeInfo(info *types.UpgradeInfo) error {
	if (info.XNextVersion == nil) != (info.XActivationTime == nil) {
		return errorsmod.Wrap(types.ErrInvalidRing, "upgrade next_version and activation_time must both be set or both be absent")
	}
	if info.XNextVersion != nil {
		if info.GetNextVersion() <= info.CurrentVersion {
			return errorsmod.Wrap(types.ErrInvalidRing, "upgrade next_version must be greater than current_version")
		}
		if info.GetActivationTime() == 0 {
			return errorsmod.Wrap(types.ErrInvalidRing, "upgrade activation_time must be positive")
		}
	}
	return nil
}

func clearRingUpgrade(ring *types.Ring) {
	ring.UpgradeInfo.XNextVersion = nil
	ring.UpgradeInfo.XActivationTime = nil
}

func setRingUpgrade(ring *types.Ring, nextVersion uint64, activationTime uint64) {
	ring.UpgradeInfo.XNextVersion = &types.UpgradeInfo_NextVersion{NextVersion: nextVersion}
	ring.UpgradeInfo.XActivationTime = &types.UpgradeInfo_ActivationTime{ActivationTime: activationTime}
}
