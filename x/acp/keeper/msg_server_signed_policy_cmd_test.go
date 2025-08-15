package keeper

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/sourcenetwork/sourcehub/x/acp/signed_policy_cmd"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

// Test that submitting the same signed payload twice in the same block fails the second time
func TestSignedPolicyCmd_ReplayRejected(t *testing.T) {
	ctx, k, accK := setupKeeper(t)
	creator := accK.GenAccount().GetAddress().String()

	policyStr := `
	name: policy
	description: ok
	resources:
		file:
			relations: 
				owner:
					doc: owner owns
					types:
						- actor-resource
				reader:
				admin:
					manages:
						- reader
			permissions: 
				own:
					expr: owner
					doc: own doc
				read: 
					expr: owner + reader
	actor:
		name: actor-resource
		doc: my actor
						`

	msg := types.MsgCreatePolicy{
		Creator:     creator,
		Policy:      policyStr,
		MarshalType: coretypes.PolicyMarshalingType_SHORT_YAML,
	}
	resp, err := k.CreatePolicy(ctx, &msg)
	require.Nil(t, err)

	cmd := types.NewRegisterObjectCmd(coretypes.NewObject("file", "foo"))
	actor, signer := mustGenerateActor()
	builder := signed_policy_cmd.NewCmdBuilder(&logicalClock, params)
	builder.Actor(actor)
	builder.PolicyID(resp.Record.Policy.Id)
	builder.PolicyCmd(cmd)
	builder.SetSigner(signer)
	jws, err := builder.BuildJWS(context.Background())
	require.NoError(t, err)

	_, err = k.SignedPolicyCmd(ctx, types.NewMsgSignedPolicyCmdFromJWS(creator, jws))
	require.NoError(t, err)

	_, err = k.SignedPolicyCmd(ctx, types.NewMsgSignedPolicyCmdFromJWS(creator, jws))
	require.Error(t, err)
}
