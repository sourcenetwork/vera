package types

import (
	"testing"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
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

func TestMsgCreateRingValidateBasicReportingConfig(t *testing.T) {
	msg := MsgCreateRing{
		Creator:      validMutationCreator,
		PeerNodeKeys: []string{"node-1"},
		Threshold:    1,
		PssInterval:  MinPSSIntervalSeconds,
		PolicyId:     "policy-1",
		Reporting: &ReportingConfig{
			DemeritConfig: DemeritConfig{
				NodeOfflineDemerits:           0,
				InvalidCryptoResponseDemerits: DefaultInvalidCryptoResponseDemerits,
				UnauthorizedRequestDemerits:   DefaultUnauthorizedRequestDemerits,
				ResetIntervalSeconds:          DefaultDemeritResetIntervalSecs,
			},
			KickThreshold: DefaultReportingKickThreshold,
		},
	}
	require.ErrorContains(t, msg.ValidateBasic(), "node_offline_demerits must be at least 1")

	msg.Reporting = &ReportingConfig{
		DemeritConfig: DemeritConfig{
			NodeOfflineDemerits:           DefaultNodeOfflineDemerits,
			InvalidCryptoResponseDemerits: DefaultInvalidCryptoResponseDemerits,
			UnauthorizedRequestDemerits:   DefaultUnauthorizedRequestDemerits,
			ResetIntervalSeconds:          0,
		},
		KickThreshold: DefaultReportingKickThreshold,
	}
	require.ErrorContains(t, msg.ValidateBasic(), "reset_interval_seconds must be at least 1")

	msg.Reporting = &ReportingConfig{
		DemeritConfig: DefaultDemeritConfig(),
		KickThreshold: 0,
	}
	require.ErrorContains(t, msg.ValidateBasic(), "kick_threshold must be at least 1")
}

func TestMsgSubmitReportValidateBasic(t *testing.T) {
	base := MsgSubmitReport{
		Creator: validMutationCreator,
		Report: ReportEnvelope{
			Domain:     "orbis-mpc-fault-report",
			ReportType: "node_offline",
			RingId:     "ring-1",
		},
		ReportId:        "report-id",
		SignatureScheme: "bls12381-g1-pubkey-g2-signature",
		Signature:       []byte{1},
	}
	require.NoError(t, base.ValidateBasic())

	missingReportID := base
	missingReportID.ReportId = ""
	require.ErrorIs(t, missingReportID.ValidateBasic(), ErrInvalidReport)

	missingDomain := base
	missingDomain.Report.Domain = ""
	require.ErrorIs(t, missingDomain.ValidateBasic(), ErrInvalidReport)

	missingReportType := base
	missingReportType.Report.ReportType = ""
	require.ErrorIs(t, missingReportType.ValidateBasic(), ErrInvalidReport)

	missingRingID := base
	missingRingID.Report.RingId = ""
	require.ErrorIs(t, missingRingID.ValidateBasic(), ErrInvalidRingId)
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
		&MsgCancelPendingRing{
			Creator: validMutationCreator,
			RingId:  "ring-1",
		},
		&MsgSetRingReportingByAcp{
			Creator:   validMutationCreator,
			RingId:    "ring-1",
			Reporting: DefaultReportingConfig(),
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
	require.Error(t, (&MsgCancelPendingRing{
		Creator: validMutationCreator,
	}).ValidateBasic())
	require.Error(t, (&MsgSetRingReportingByAcp{
		Creator: validMutationCreator,
	}).ValidateBasic())
	require.Error(t, (&MsgSetRingReportingByAcp{
		Creator: validMutationCreator,
		RingId:  "ring-1",
		Reporting: ReportingConfig{
			DemeritConfig: DefaultDemeritConfig(),
		},
	}).ValidateBasic())
}

func TestMsgSetRingReportingByAcpRegisteredForAnyUnpacking(t *testing.T) {
	registry := codectypes.NewInterfaceRegistry()
	RegisterInterfaces(registry)

	msg := &MsgSetRingReportingByAcp{
		Creator:   validMutationCreator,
		RingId:    "ring-1",
		Reporting: DefaultReportingConfig(),
	}
	any, err := codectypes.NewAnyWithValue(msg)
	require.NoError(t, err)

	var unpacked sdk.Msg
	require.NoError(t, registry.UnpackAny(any, &unpacked))
	require.Equal(t, msg, unpacked)
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
