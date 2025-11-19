package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
)

var _ sdk.Msg = &MsgEditPolicy{}

func NewMsgEditPolicy(creator string, policyId string, policy string, marshalType coretypes.PolicyMarshalingType) *MsgEditPolicy {
	return &MsgEditPolicy{
		Creator:     creator,
		PolicyId:    policyId,
		Policy:      policy,
		MarshalType: marshalType,
	}
}
