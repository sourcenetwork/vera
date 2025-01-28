package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgDirectPolicyCmd{}

func NewMsgBearerPolicyCmd(creator string, token string, policyId string, cmd *PolicyCmd) *MsgBearerPolicyCmd {
	return &MsgBearerPolicyCmd{
		Creator:     creator,
		BearerToken: token,
		PolicyId:    policyId,
		Cmd:         cmd,
	}
}
