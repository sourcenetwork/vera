package keeper

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/sourcenetwork/sourcehub/x/acp/signed_policy_cmd"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

func TestSignedPolicyCmd_ReplayProtection(t *testing.T) {
	ctx, k, accK := setupKeeper(t)
	creator := accK.GenAccount().GetAddress().String()

	policyStr := `
description: ok
name: policy
resources:
- name: file
  permissions:
  - doc: own doc
    expr: owner
    name: own
  - expr: owner + reader
    name: read
  relations:
  - doc: owner owns
    name: owner
    types:
    - actor
  - name: reader
    types:
    - actor
`

	msg := types.MsgCreatePolicy{
		Creator:     creator,
		Policy:      policyStr,
		MarshalType: coretypes.PolicyMarshalingType_YAML,
	}
	resp, err := k.CreatePolicy(ctx, &msg)
	require.Nil(t, err)

	actor, signer := mustGenerateActor()

	registerCmd := types.NewRegisterObjectCmd(coretypes.NewObject("file", "foo"))
	builder := signed_policy_cmd.NewCmdBuilder(&logicalClock, params)
	builder.Actor(actor)
	builder.PolicyID(resp.Record.Policy.Id)
	builder.PolicyCmd(registerCmd)
	builder.SetSigner(signer)
	registerJWS, err := builder.BuildJWS(context.Background())
	require.NoError(t, err)

	// First command should succeed
	_, err = k.SignedPolicyCmd(ctx, types.NewMsgSignedPolicyCmdFromJWS(creator, registerJWS))
	require.NoError(t, err)

	// Submitting the same command again should fail
	_, err = k.SignedPolicyCmd(ctx, types.NewMsgSignedPolicyCmdFromJWS(creator, registerJWS))
	require.Error(t, err)
	require.ErrorIs(t, err, signed_policy_cmd.ErrPayloadAlreadyProcessed)

	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)

	// Submitting the same command with a different block height should fail
	_, err = k.SignedPolicyCmd(ctx, types.NewMsgSignedPolicyCmdFromJWS(creator, registerJWS))
	require.Error(t, err)
	require.ErrorIs(t, err, signed_policy_cmd.ErrPayloadAlreadyProcessed)

	relationship := coretypes.NewActorRelationship("file", "foo", "reader", "did:key:alice")
	cmd := types.NewSetRelationshipCmd(relationship)
	builder.PolicyCmd(cmd)
	relationshipJWS, err := builder.BuildJWS(context.Background())
	require.NoError(t, err)

	// Submitting a different command should succeed
	_, err = k.SignedPolicyCmd(ctx, types.NewMsgSignedPolicyCmdFromJWS(creator, relationshipJWS))
	require.NoError(t, err)

	// Submitting the same command again should fail
	_, err = k.SignedPolicyCmd(ctx, types.NewMsgSignedPolicyCmdFromJWS(creator, relationshipJWS))
	require.Error(t, err)
	require.ErrorIs(t, err, signed_policy_cmd.ErrPayloadAlreadyProcessed)
}
