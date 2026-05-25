package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgCreateRing{}

func NewMsgCreateRing(creator, namespace string, peerIDs []string, threshold uint32) *MsgCreateRing {
	return &MsgCreateRing{
		Creator:   creator,
		Namespace: namespace,
		PeerIds:   peerIDs,
		Threshold: threshold,
	}
}

func (msg *MsgCreateRing) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	if msg.Namespace == "" {
		return ErrInvalidNamespaceId
	}
	if len(msg.PeerIds) == 0 {
		return errorsmod.Wrap(ErrInvalidRing, "missing peer_ids")
	}
	if msg.Threshold == 0 || int(msg.Threshold) > len(msg.PeerIds) {
		return errorsmod.Wrapf(ErrInvalidRing, "threshold %d is invalid for committee size %d", msg.Threshold, len(msg.PeerIds))
	}
	return nil
}
