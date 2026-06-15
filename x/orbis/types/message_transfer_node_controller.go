package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgTransferNodeController{}

func (msg *MsgTransferNodeController) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	if msg.NodeKey == "" {
		return errorsmod.Wrap(ErrInvalidNodeInfo, "missing node_key")
	}
	if msg.ControllerKey == "" {
		return errorsmod.Wrap(ErrInvalidNodeInfo, "missing controller_key")
	}
	return nil
}
