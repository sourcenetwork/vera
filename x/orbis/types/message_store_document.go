package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgStoreDocument{}

func NewMsgStoreDocument(creator, ringID, document, proof, policyID, resource, permission string) *MsgStoreDocument {
	return &MsgStoreDocument{
		Creator:    creator,
		RingId:     ringID,
		Document:   document,
		Proof:      proof,
		PolicyId:   policyID,
		Resource:   resource,
		Permission: permission,
	}
}

func (msg *MsgStoreDocument) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	if msg.RingId == "" {
		return ErrInvalidRingId
	}
	if msg.Document == "" {
		return errorsmod.Wrap(ErrInvalidDocument, "missing document")
	}
	if msg.Proof == "" {
		return errorsmod.Wrap(ErrInvalidDocument, "missing proof")
	}
	if msg.PolicyId == "" || msg.Resource == "" || msg.Permission == "" {
		return errorsmod.Wrap(ErrInvalidDocument, "missing policy binding")
	}
	return nil
}
