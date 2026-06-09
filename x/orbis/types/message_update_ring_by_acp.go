package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgUpdateRingByAcp{}

func NewMsgUpdateRingByAcp(creator, ringID string) *MsgUpdateRingByAcp {
	return &MsgUpdateRingByAcp{
		Creator: creator,
		RingId:  ringID,
	}
}

func (msg *MsgUpdateRingByAcp) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	if msg.RingId == "" {
		return ErrInvalidRingId
	}
	hasNextVersion := msg.XNextVersion != nil
	hasActivationTime := msg.XActivationTime != nil
	if hasNextVersion != hasActivationTime {
		return errorsmod.Wrap(ErrInvalidRing, "next_version and activation_time must be supplied together")
	}
	if msg.ClearUpgrade && hasNextVersion {
		return errorsmod.Wrap(ErrInvalidRing, "clear_upgrade cannot be combined with a new upgrade schedule")
	}
	return nil
}
