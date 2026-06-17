package types

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

var validMutationCreator = sdk.AccAddress(make([]byte, 20)).String()

func TestMsgCreateRingValidateBasicPSSInterval(t *testing.T) {
	base := MsgCreateRing{
		Creator:      validMutationCreator,
		PeerNodeKeys: []string{"node-1"},
		Threshold:    1,
		PssInterval:  MinPSSIntervalSeconds,
		PolicyId:     "policy-1",
	}
	require.NoError(t, base.ValidateBasic())

	for _, pssInterval := range []uint64{0, MinPSSIntervalSeconds - 1} {
		msg := base
		msg.PssInterval = pssInterval
		require.ErrorContains(t, msg.ValidateBasic(), "pss_interval must be at least 86400 seconds")
	}
}

func TestRingMutationMessagesValidateBasic(t *testing.T) {
	validMessages := []interface{ ValidateBasic() error }{
		&MsgStartRingReshareByAcp{
			Creator:         validMutationCreator,
			RingId:          "ring-1",
			NewPeerNodeKeys: []string{"node-1"},
		},
		&MsgSetRingPssIntervalByAcp{
			Creator:     validMutationCreator,
			RingId:      "ring-1",
			PssInterval: MinPSSIntervalSeconds,
		},
		&MsgScheduleRingUpgradeByAcp{
			Creator:        validMutationCreator,
			RingId:         "ring-1",
			NextVersion:    1,
			ActivationTime: 100,
		},
		&MsgCancelRingUpgradeByAcp{
			Creator: validMutationCreator,
			RingId:  "ring-1",
		},
	}
	for _, msg := range validMessages {
		require.NoError(t, msg.ValidateBasic())
	}

	require.Error(t, (&MsgStartRingReshareByAcp{
		Creator: validMutationCreator,
		RingId:  "ring-1",
	}).ValidateBasic())
	require.Error(t, (&MsgSetRingPssIntervalByAcp{
		Creator: validMutationCreator,
		RingId:  "ring-1",
	}).ValidateBasic())
	require.Error(t, (&MsgSetRingPssIntervalByAcp{
		Creator:     validMutationCreator,
		RingId:      "ring-1",
		PssInterval: MinPSSIntervalSeconds - 1,
	}).ValidateBasic())
	require.Error(t, (&MsgScheduleRingUpgradeByAcp{
		Creator:        validMutationCreator,
		RingId:         "ring-1",
		ActivationTime: 100,
	}).ValidateBasic())
}

func TestNodeMutationMessagesValidateBasic(t *testing.T) {
	validMessages := []interface{ ValidateBasic() error }{
		&MsgUpdateNodePeerId{
			Creator: validMutationCreator,
			NodeKey: "node-1",
			PeerId:  "peer-1",
		},
		&MsgTransferNodeController{
			Creator:       validMutationCreator,
			NodeKey:       "node-1",
			ControllerKey: "controller-1",
		},
		&MsgAddNodeToWhitelist{
			Creator: validMutationCreator,
			NodeKey: "node-1",
			Target:  &MsgAddNodeToWhitelist_PolicyId{PolicyId: "policy-1"},
		},
		&MsgRemoveNodeFromWhitelist{
			Creator: validMutationCreator,
			NodeKey: "node-1",
			Target:  &MsgRemoveNodeFromWhitelist_RingId{RingId: "ring-1"},
		},
	}
	for _, msg := range validMessages {
		require.NoError(t, msg.ValidateBasic())
	}

	require.Error(t, (&MsgUpdateNodePeerId{
		Creator: validMutationCreator,
		NodeKey: "node-1",
	}).ValidateBasic())
	require.Error(t, (&MsgTransferNodeController{
		Creator: validMutationCreator,
		NodeKey: "node-1",
	}).ValidateBasic())
	require.Error(t, (&MsgAddNodeToWhitelist{
		Creator: validMutationCreator,
		NodeKey: "node-1",
	}).ValidateBasic())
	require.Error(t, (&MsgRemoveNodeFromWhitelist{
		Creator: validMutationCreator,
		NodeKey: "node-1",
		Target:  &MsgRemoveNodeFromWhitelist_PolicyId{},
	}).ValidateBasic())
}
