package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgStoreKeyDerivation{}

func NewMsgStoreKeyDerivation(creator, ringID, derivation, policyID, resource, permission string) *MsgStoreKeyDerivation {
	return &MsgStoreKeyDerivation{
		Creator:    creator,
		RingId:     ringID,
		Derivation: derivation,
		PolicyId:   policyID,
		Resource:   resource,
		Permission: permission,
	}
}

func (msg *MsgStoreKeyDerivation) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	if msg.RingId == "" {
		return ErrInvalidRingId
	}
	if msg.Derivation == "" {
		return errorsmod.Wrap(ErrInvalidKeyDerivation, "missing derivation")
	}
	if msg.PolicyId == "" || msg.Resource == "" || msg.Permission == "" {
		return errorsmod.Wrap(ErrInvalidKeyDerivation, "missing policy binding")
	}
	return nil
}
