package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgStartRingReshareByAcp{}

func (msg *MsgStartRingReshareByAcp) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	if msg.RingId == "" {
		return ErrInvalidRingId
	}
	if len(msg.NewPeerNodeKeys) == 0 && msg.XNewThreshold == nil {
		return errorsmod.Wrap(ErrInvalidRing, "reshare must change the committee or threshold")
	}
	if msg.XNewThreshold != nil && msg.GetNewThreshold() == 0 {
		return errorsmod.Wrap(ErrInvalidRing, "new_threshold must be at least 1")
	}
	return nil
}
