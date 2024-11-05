package types

import (
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
	// this line is used by starport scaffolding # 1
)

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgCreatePolicy{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgCheckAccess{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgSignedPolicyCmd{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgBearerPolicyCmd{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgDirectPolicyCmd{},
	)
	// this line is used by starport scaffolding # 3

	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgUpdateParams{},
	)

	registerResponses(registry)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

func registerResponses(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil), &MsgCreatePolicyResponse{})
	registry.RegisterImplementations((*sdk.Msg)(nil), &MsgCheckAccessResponse{})
	registry.RegisterImplementations((*sdk.Msg)(nil), &MsgSignedPolicyCmdResponse{})
	registry.RegisterImplementations((*sdk.Msg)(nil), &MsgDirectPolicyCmdResponse{})
	registry.RegisterImplementations((*sdk.Msg)(nil), &MsgUpdateParamsResponse{})
}
