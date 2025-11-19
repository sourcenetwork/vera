package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgDirectPolicyCmd{}

func NewMsgDirectPolicyCmd(creator string, policyId string, cmd *PolicyCmd) *MsgDirectPolicyCmd {
	return &MsgDirectPolicyCmd{
		Creator:  creator,
		PolicyId: policyId,
		Cmd:      cmd,
	}
}
