package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgRemoveCollaborator{}

func NewMsgRemoveCollaborator(creator string, namespace string, collaborator string) *MsgRemoveCollaborator {
	return &MsgRemoveCollaborator{
		Creator:      creator,
		Namespace:    namespace,
		Collaborator: collaborator,
	}
}

func (msg *MsgRemoveCollaborator) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if msg.Namespace == "" {
		return ErrInvalidNamespaceId
	}

	_, err = sdk.AccAddressFromBech32(msg.Collaborator)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid collaborator address (%s)", err)
	}

	return nil
}
