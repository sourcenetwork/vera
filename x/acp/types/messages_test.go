package types

import (
	"testing"

	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/stretchr/testify/require"
)

func TestNewMsgBearerPolicyCmd(t *testing.T) {
	cmd := NewRegisterObjectCmd(coretypes.NewObject("res", "obj1"))
	msg := NewMsgBearerPolicyCmd("cosmos1creator", "bearer-token", "policy-1", cmd)
	require.Equal(t, "cosmos1creator", msg.Creator)
	require.Equal(t, "bearer-token", msg.BearerToken)
	require.Equal(t, "policy-1", msg.PolicyId)
	require.Equal(t, cmd, msg.Cmd)
}

func TestNewMsgCheckAccess(t *testing.T) {
	accessReq := &coretypes.AccessRequest{
		Operations: []*coretypes.Operation{
			{
				Object:     coretypes.NewObject("res", "obj1"),
				Permission: "read",
			},
		},
		Actor: coretypes.NewActor("did:example:alice"),
	}
	msg := NewMsgCheckAccess("cosmos1creator", "policy-1", accessReq)
	require.Equal(t, "cosmos1creator", msg.Creator)
	require.Equal(t, "policy-1", msg.PolicyId)
	require.Equal(t, accessReq, msg.AccessRequest)
}

func TestNewMsgCreatePolicy(t *testing.T) {
	msg := NewMsgCreatePolicy("cosmos1creator", "policy-yaml", coretypes.PolicyMarshalingType_YAML)
	require.Equal(t, "cosmos1creator", msg.Creator)
	require.Equal(t, "policy-yaml", msg.Policy)
	require.Equal(t, coretypes.PolicyMarshalingType_YAML, msg.MarshalType)
}

func TestNewMsgDirectPolicyCmd(t *testing.T) {
	cmd := NewRegisterObjectCmd(coretypes.NewObject("res", "obj1"))
	msg := NewMsgDirectPolicyCmd("cosmos1creator", "policy-1", cmd)
	require.Equal(t, "cosmos1creator", msg.Creator)
	require.Equal(t, "policy-1", msg.PolicyId)
	require.Equal(t, cmd, msg.Cmd)
}

func TestNewMsgEditPolicy(t *testing.T) {
	msg := NewMsgEditPolicy("cosmos1creator", "policy-1", "new-policy-yaml", coretypes.PolicyMarshalingType_YAML)
	require.Equal(t, "cosmos1creator", msg.Creator)
	require.Equal(t, "policy-1", msg.PolicyId)
	require.Equal(t, "new-policy-yaml", msg.Policy)
	require.Equal(t, coretypes.PolicyMarshalingType_YAML, msg.MarshalType)
}

func TestNewMsgSignedPolicyCmd(t *testing.T) {
	msg := NewMsgSignedPolicyCmd("cosmos1creator", "payload", MsgSignedPolicyCmd_JWS)
	require.Equal(t, "cosmos1creator", msg.Creator)
	require.Equal(t, "payload", msg.Payload)
	require.Equal(t, MsgSignedPolicyCmd_JWS, msg.Type)
}

func TestNewMsgSignedPolicyCmdFromJWS(t *testing.T) {
	msg := NewMsgSignedPolicyCmdFromJWS("cosmos1creator", "jws-payload")
	require.Equal(t, "cosmos1creator", msg.Creator)
	require.Equal(t, "jws-payload", msg.Payload)
	require.Equal(t, MsgSignedPolicyCmd_JWS, msg.Type)
}

func TestMsgUpdateParamsValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		msg     MsgUpdateParams
		wantErr bool
	}{
		{
			"valid authority",
			MsgUpdateParams{
				Authority: "cosmos10d07y265gmmuvt4z0w9aw880jnsr700j6zn9kn",
				Params:    DefaultParams(),
			},
			false,
		},
		{
			"invalid authority",
			MsgUpdateParams{
				Authority: "invalid",
				Params:    DefaultParams(),
			},
			true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.msg.ValidateBasic()
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
