package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgEditPolicy{}

func NewMsgMsgEditPolicy(creator string, policyId string) *MsgEditPolicy {
	return &MsgEditPolicy{
		Creator:  creator,
		PolicyId: policyId,
	}
}
