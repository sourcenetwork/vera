package test

import (
	"time"

	prototypes "github.com/cosmos/gogoproto/types"
	"github.com/stretchr/testify/require"

	"github.com/sourcenetwork/sourcehub/x/acp/bearer_token"
	"github.com/sourcenetwork/sourcehub/x/acp/signed_policy_cmd"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

func dispatchPolicyCmd(ctx *TestCtx, policyId string, actor *TestActor, policyCmd *types.PolicyCmd) (result *types.PolicyCmdResult, err error) {
	ctx.State.TokenIssueTs = time.Now()
	ctx.State.TokenIssueProtoTs = prototypes.TimestampNow()
	switch ctx.Strategy {
	case BearerToken:
		ts := ctx.State.TokenIssueTs
		token := bearer_token.BearerToken{
			IssuerID:          actor.DID,
			AuthorizedAccount: ctx.TxSigner.SourceHubAddr,
			IssuedTime:        ts.Unix(),
			ExpirationTime:    ts.Add(bearer_token.DefaultExpirationTime).Unix(),
		}
		jws, jwsErr := token.ToJWS(actor.Signer)
		require.NoError(ctx.T, jwsErr)
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
		builder := signed_policy_cmd.NewCmdBuilder(ctx.Executor, ctx.GetParams())
		builder.PolicyCmd(policyCmd)
		builder.Actor(actor.DID)
		builder.IssuedAt(ctx.State.TokenIssueProtoTs)
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
		// For Direct Authentication we use the action Actor as the signer
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
