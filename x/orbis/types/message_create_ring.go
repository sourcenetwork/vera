package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgCreateRing{}

func NewMsgCreateRing(creator string, peerNodeKeys []string, threshold uint32, policyID string, currentVersion uint64) *MsgCreateRing {
	return &MsgCreateRing{
		Creator:        creator,
		PeerNodeKeys:   peerNodeKeys,
		Threshold:      threshold,
		PolicyId:       policyID,
		CurrentVersion: currentVersion,
	}
}

func (msg *MsgCreateRing) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	if len(msg.PeerNodeKeys) == 0 {
		return errorsmod.Wrap(ErrInvalidRing, "missing peer_node_keys")
	}
	if msg.Threshold == 0 || int(msg.Threshold) > len(msg.PeerNodeKeys) {
		return errorsmod.Wrapf(ErrInvalidRing, "threshold %d is invalid for committee size %d", msg.Threshold, len(msg.PeerNodeKeys))
	}
	if msg.PolicyId == "" {
		return errorsmod.Wrap(ErrInvalidRing, "missing policy_id")
	}
	return nil
}
