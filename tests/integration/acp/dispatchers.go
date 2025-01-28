package test

import (
	"github.com/stretchr/testify/require"

	"github.com/sourcenetwork/sourcehub/x/acp/bearer_token"
	"github.com/sourcenetwork/sourcehub/x/acp/signed_policy_cmd"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

func dispatchPolicyCmd(ctx *TestCtx, policyId string, actor *TestActor, policyCmd *types.PolicyCmd) (result *types.PolicyCmdResult, err error) {
	switch ctx.Strategy {
	case BearerToken:
		jws := genToken(ctx, actor)
		msg := &types.MsgBearerPolicyCmd{
			Creator:     ctx.TxSigner.SourceHubAddr,
			BearerToken: jws,
			PolicyId:    policyId,
			Cmd:         policyCmd,
		}
		resp, respErr := ctx.Executor.BearerPolicyCmd(ctx, msg)
		if resp != nil {
			result = resp.Result
		}
		err = respErr
	case SignedPayload:
		var jws string
		builder := signed_policy_cmd.NewCmdBuilder(ctx.LogicalClock, ctx.GetParams())
		builder.PolicyCmd(policyCmd)
		builder.Actor(actor.DID)
		builder.IssuedAt(ctx.TokenIssueProtoTs)
		builder.PolicyID(policyId)
		builder.SetSigner(actor.Signer)
		jws, err = builder.BuildJWS(ctx)
		require.NoError(ctx.T, err)

		msg := &types.MsgSignedPolicyCmd{
			Creator: ctx.TxSigner.SourceHubAddr,
			Payload: jws,
			Type:    types.MsgSignedPolicyCmd_JWS,
		}
		resp, respErr := ctx.Executor.SignedPolicyCmd(ctx, msg)
		if resp != nil {
			result = resp.Result
		}
		err = respErr
	case Direct:
		// For Direct Authentication we use the action Action as the signer
		ctx.TxSigner = actor
		msg := &types.MsgDirectPolicyCmd{
			Creator:  actor.SourceHubAddr,
			PolicyId: policyId,
			Cmd:      policyCmd,
		}
		resp, respErr := ctx.Executor.DirectPolicyCmd(ctx, msg)
		if resp != nil {
			result = resp.Result
		}
		err = respErr
	}
	return result, err
}

func genToken(ctx *TestCtx, actor *TestActor) string {
	token := bearer_token.BearerToken{
		IssuerID:          actor.DID,
		AuthorizedAccount: ctx.TxSigner.SourceHubAddr,
		IssuedTime:        ctx.TokenIssueTs.Unix(),
		ExpirationTime:    ctx.TokenIssueTs.Add(bearer_token.DefaultExpirationTime).Unix(),
	}
	jws, err := token.ToJWS(actor.Signer)
	require.NoError(ctx.T, err)
	return jws
}
