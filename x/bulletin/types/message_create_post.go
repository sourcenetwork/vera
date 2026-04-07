package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgCreatePost{}

func NewMsgCreatePost(creator string, namespace string, payload []byte) *MsgCreatePost {
	return &MsgCreatePost{
		Creator:   creator,
		Namespace: namespace,
		Payload:   payload,
	}
}

func (msg *MsgCreatePost) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if msg.Namespace == "" {
		return ErrInvalidNamespaceId
	}

	if len(msg.Payload) == 0 {
		return ErrInvalidPostPayload
	}

	return nil
}
