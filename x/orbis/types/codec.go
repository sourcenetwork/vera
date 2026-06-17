package types

import (
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

// RegisterInterfaces registers module messages.
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgCreateRing{},
		&MsgFinalizeRing{},
		&MsgStartRingReshareByAcp{},
		&MsgSetRingPssIntervalByAcp{},
		&MsgScheduleRingUpgradeByAcp{},
		&MsgCancelRingUpgradeByAcp{},
		&MsgFinalizeRingReshareByThresholdSignature{},
		&MsgStoreDocument{},
		&MsgStoreKeyDerivation{},
		&MsgCreateNodeInfo{},
		&MsgUpdateNodePeerId{},
		&MsgTransferNodeController{},
		&MsgAddNodeToWhitelist{},
		&MsgRemoveNodeFromWhitelist{},
		&MsgUpdateParams{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
