package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sourcenetwork/acp_core/pkg/types"
)

var _ sdk.Msg = &MsgCheckAccess{}

func NewMsgCheckAccess(creator string, policyId string, accesReq *types.AccessRequest) *MsgCheckAccess {
	return &MsgCheckAccess{
		Creator:       creator,
		PolicyId:      policyId,
		AccessRequest: accesReq,
	}
}
