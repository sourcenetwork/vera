package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgSetRingPssIntervalByAcp{}

func (msg *MsgSetRingPssIntervalByAcp) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	if msg.RingId == "" {
		return ErrInvalidRingId
	}
	if msg.PssInterval == 0 {
		return errorsmod.Wrap(ErrInvalidRing, "pss_interval must be positive")
	}
	return nil
}
