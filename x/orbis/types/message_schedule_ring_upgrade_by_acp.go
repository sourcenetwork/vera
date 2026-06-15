package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgScheduleRingUpgradeByAcp{}

func (msg *MsgScheduleRingUpgradeByAcp) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	if msg.RingId == "" {
		return ErrInvalidRingId
	}
	if msg.NextVersion == 0 {
		return errorsmod.Wrap(ErrInvalidRing, "next_version must be positive")
	}
	if msg.ActivationTime == 0 {
		return errorsmod.Wrap(ErrInvalidRing, "activation_time must be positive")
	}
	return nil
}
