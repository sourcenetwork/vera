package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgRemoveNodeFromWhitelist{}

func (msg *MsgRemoveNodeFromWhitelist) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	if msg.NodeKey == "" {
		return errorsmod.Wrap(ErrInvalidNodeInfo, "missing node_key")
	}
	switch target := msg.Target.(type) {
	case *MsgRemoveNodeFromWhitelist_PolicyId:
		if target.PolicyId == "" {
			return errorsmod.Wrap(ErrInvalidNodeInfo, "missing policy_id")
		}
	case *MsgRemoveNodeFromWhitelist_RingId:
		if target.RingId == "" {
			return errorsmod.Wrap(ErrInvalidNodeInfo, "missing ring_id")
		}
	default:
		return errorsmod.Wrap(ErrInvalidNodeInfo, "missing whitelist target")
	}
	return nil
}
