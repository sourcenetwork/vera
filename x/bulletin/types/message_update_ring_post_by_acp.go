package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgUpdateRingPostByAcp{}

func NewMsgUpdateRingPostByAcp(creator, namespace, postId string) *MsgUpdateRingPostByAcp {
	return &MsgUpdateRingPostByAcp{
		Creator:   creator,
		Namespace: namespace,
		PostId:    postId,
	}
}

func (msg *MsgUpdateRingPostByAcp) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	if msg.Namespace == "" {
		return ErrInvalidNamespaceId
	}
	if msg.PostId == "" {
		return ErrInvalidPostId
	}
	return nil
}
