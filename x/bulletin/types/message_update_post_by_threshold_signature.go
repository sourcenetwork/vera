package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgUpdatePostByThresholdSignature{}

func NewMsgUpdatePostByThresholdSignature(
	creator string,
	namespace string,
	postId string,
	payload []byte,
	signatureScheme string,
	signature []byte,
) *MsgUpdatePostByThresholdSignature {
	return &MsgUpdatePostByThresholdSignature{
		Creator:         creator,
		Namespace:       namespace,
		PostId:          postId,
		Payload:         payload,
		SignatureScheme: signatureScheme,
		Signature:       signature,
	}
}

func (msg *MsgUpdatePostByThresholdSignature) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	if msg.Namespace == "" {
		return ErrInvalidNamespaceId
	}
	if msg.PostId == "" {
		return ErrPostNotFound
	}
	if len(msg.Payload) == 0 {
		return ErrInvalidPostPayload
	}
	if len(msg.Signature) == 0 {
		return ErrInvalidThresholdSignature
	}
	return nil
}
