package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgRegisterNamespace{}

func NewMsgRegisterNamespace(creator string, namespace string) *MsgRegisterNamespace {
	return &MsgRegisterNamespace{
		Creator:   creator,
		Namespace: namespace,
	}
}

func (msg *MsgRegisterNamespace) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if msg.Namespace == "" {
		return ErrInvalidNamespaceId
	}

	return nil
}
