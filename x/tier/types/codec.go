package types

import (
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgLock{},
		&MsgLockAuto{},
		&MsgUnlock{},
		&MsgRedelegate{},
		&MsgCancelUnlocking{},
		&MsgCreateDeveloper{},
		&MsgUpdateDeveloper{},
		&MsgRemoveDeveloper{},
		&MsgAddUserSubscription{},
		&MsgUpdateUserSubscription{},
		&MsgRemoveUserSubscription{},
		&MsgUpdateParams{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
