package sdk

import (
	acptypes "github.com/sourcenetwork/sourcehub/x/acp/types"
	bulletintypes "github.com/sourcenetwork/sourcehub/x/bulletin/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// MsgSet acts as set of Msgs to be added to a Tx
// Each added Msg returns a Mapper which can be used to extract associated data from
// the events emitted after Tx execution
type MsgSet struct {
	msgs []sdk.Msg
}

func (b *MsgSet) GetMsgs() []sdk.Msg {
	return b.msgs
}

func (b *MsgSet) addMsg(msg sdk.Msg) (idx int) {
	b.msgs = append(b.msgs, msg)
	idx = len(b.msgs) - 1
	return
}

// WithCreatePolicy includes a MsgCreatePolicy to the Tx
func (b *MsgSet) WithCreatePolicy(msg *acptypes.MsgCreatePolicy) Mapper[*acptypes.MsgCreatePolicyResponse] {
	idx := b.addMsg(msg)
	return newMapper(idx, &acptypes.MsgCreatePolicyResponse{})
}

// WithEditPolicy includes a MsgEditPolicy to the Tx
func (b *MsgSet) WithEditPolicy(msg *acptypes.MsgEditPolicy) Mapper[*acptypes.MsgEditPolicyResponse] {
	idx := b.addMsg(msg)
	return newMapper(idx, &acptypes.MsgEditPolicyResponse{})
}

// WithCheckAccess includes a MsgCheckAcces to the Tx
func (b *MsgSet) WithCheckAccess(msg *acptypes.MsgCheckAccess) Mapper[*acptypes.MsgCheckAccessResponse] {
	idx := b.addMsg(msg)
	return newMapper(idx, &acptypes.MsgCheckAccessResponse{})
}

// WithSignedPolicyCmd includes a MsgSignedPolicyCmd to the Tx
func (b *MsgSet) WithSignedPolicyCmd(msg *acptypes.MsgSignedPolicyCmd) Mapper[*acptypes.MsgSignedPolicyCmdResponse] {
	idx := b.addMsg(msg)
	return newMapper(idx, &acptypes.MsgSignedPolicyCmdResponse{})
}

// WithCreatePost includes a MsgCreatePost to the Tx
func (b *MsgSet) WithCreatePost(msg *bulletintypes.MsgCreatePost) Mapper[*bulletintypes.MsgCreatePostResponse] {
	idx := b.addMsg(msg)
	return newMapper(idx, &bulletintypes.MsgCreatePostResponse{})
}

func (b *MsgSet) WithBearerPolicyCmd(msg *acptypes.MsgBearerPolicyCmd) Mapper[*acptypes.MsgBearerPolicyCmdResponse] {
	idx := b.addMsg(msg)
	return newMapper(idx, &acptypes.MsgBearerPolicyCmdResponse{})
}

func (b *MsgSet) WithDirectPolicyCmd(msg *acptypes.MsgDirectPolicyCmd) Mapper[*acptypes.MsgDirectPolicyCmdResponse] {
	idx := b.addMsg(msg)
	return newMapper(idx, &acptypes.MsgDirectPolicyCmdResponse{})
}
