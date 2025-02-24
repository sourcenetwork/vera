package test

import (
	"github.com/stretchr/testify/require"

	"github.com/sourcenetwork/sourcehub/x/acp/bearer_token"
	"github.com/sourcenetwork/sourcehub/x/acp/signed_policy_cmd"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

func setRelationshipDispatcher(ctx *TestCtx, action *SetRelationshipAction) (result *types.SetRelationshipCmdResult, err error) {
	switch ctx.Strategy {
	case BearerToken:
		jws := genToken(ctx, action.Actor)
		msg := &types.MsgBearerPolicyCmd{
			Creator:      ctx.TxSigner.SourceHubAddr,
			BearerToken:  jws,
			PolicyId:     action.PolicyId,
			Cmd:          types.NewSetRelationshipCmd(action.Relationship),
			CreationTime: TimeToProto(ctx.Timestamp),
		}
		resp, respErr := ctx.Executor.BearerPolicyCmd(ctx, msg)
		if resp != nil {
			result = resp.Result.GetSetRelationshipResult()
		}
		err = respErr
	case SignedPayload:
		var jws string
		builder := signed_policy_cmd.NewCmdBuilder(ctx.LogicalClock, ctx.GetParams())
		builder.SetRelationship(action.Relationship)
		builder.Actor(action.Actor.DID)
		builder.CreationTimestamp(TimeToProto(ctx.Timestamp))
		builder.PolicyID(action.PolicyId)
		builder.SetSigner(action.Actor.Signer)
		jws, err = builder.BuildJWS(ctx)
		require.NoError(ctx.T, err)

		msg := &types.MsgSignedPolicyCmd{
			Creator: ctx.TxSigner.SourceHubAddr,
			Payload: jws,
			Type:    types.MsgSignedPolicyCmd_JWS,
		}
		resp, respErr := ctx.Executor.SignedPolicyCmd(ctx, msg)
		if resp != nil {
			result = resp.Result.GetSetRelationshipResult()
		}
		err = respErr
	case Direct:
		// For Direct Authentication we use the action Actor as the signer
		ctx.TxSigner = action.Actor
		msg := &types.MsgDirectPolicyCmd{
			Creator:      action.Actor.SourceHubAddr,
			PolicyId:     action.PolicyId,
			Cmd:          types.NewSetRelationshipCmd(action.Relationship),
			CreationTime: TimeToProto(ctx.Timestamp),
		}
		resp, respErr := ctx.Executor.DirectPolicyCmd(ctx, msg)
		if resp != nil {
			result = resp.Result.GetSetRelationshipResult()
		}
		err = respErr
	}
	return
}

func deleteRelationshipDispatcher(ctx *TestCtx, action *DeleteRelationshipAction) (*types.DeleteRelationshipCmdResult, error) {
	var result *types.DeleteRelationshipCmdResult
	var resultErr error
	switch ctx.Strategy {
	case BearerToken:
		jws := genToken(ctx, action.Actor)
		msg := &types.MsgBearerPolicyCmd{
			Creator:      ctx.TxSigner.SourceHubAddr,
			BearerToken:  jws,
			PolicyId:     action.PolicyId,
			Cmd:          types.NewDeleteRelationshipCmd(action.Relationship),
			CreationTime: TimeToProto(ctx.Timestamp),
		}
		resp, err := ctx.Executor.BearerPolicyCmd(ctx, msg)
		if resp != nil {
			result = resp.Result.GetDeleteRelationshipResult()
		}
		resultErr = err
	case SignedPayload:
		builder := signed_policy_cmd.NewCmdBuilder(ctx.LogicalClock, ctx.GetParams())
		builder.DeleteRelationship(action.Relationship)
		builder.Actor(action.Actor.DID)
		builder.CreationTimestamp(TimeToProto(ctx.Timestamp))
		builder.PolicyID(action.PolicyId)
		builder.SetSigner(action.Actor.Signer)
		jws, err := builder.BuildJWS(ctx)
		require.NoError(ctx.T, err)

		msg := &types.MsgSignedPolicyCmd{
			Creator: ctx.TxSigner.SourceHubAddr,
			Payload: jws,
			Type:    types.MsgSignedPolicyCmd_JWS,
		}
		resp, respErr := ctx.Executor.SignedPolicyCmd(ctx, msg)
		if resp != nil {
			result = resp.Result.GetDeleteRelationshipResult()
		}
		resultErr = respErr
	case Direct:
		// For Direct Authentication we use the action Actor as the signer
		ctx.TxSigner = action.Actor
		msg := &types.MsgDirectPolicyCmd{
			Creator:      action.Actor.SourceHubAddr,
			PolicyId:     action.PolicyId,
			Cmd:          types.NewDeleteRelationshipCmd(action.Relationship),
			CreationTime: TimeToProto(ctx.Timestamp),
		}
		resp, respErr := ctx.Executor.DirectPolicyCmd(ctx, msg)
		if resp != nil {
			result = resp.Result.GetDeleteRelationshipResult()
		}
		resultErr = respErr
	}
	return result, resultErr
}

func registerObjectDispatcher(ctx *TestCtx, action *RegisterObjectAction) (result *types.RegisterObjectCmdResult, err error) {
	switch ctx.Strategy {
	case BearerToken:
		jws := genToken(ctx, action.Actor)
		msg := &types.MsgBearerPolicyCmd{
			Creator:      ctx.TxSigner.SourceHubAddr,
			BearerToken:  jws,
			PolicyId:     action.PolicyId,
			Cmd:          types.NewRegisterObjectCmd(action.Object),
			CreationTime: TimeToProto(ctx.Timestamp),
		}
		resp, respErr := ctx.Executor.BearerPolicyCmd(ctx, msg)
		if resp != nil {
			result = resp.Result.GetRegisterObjectResult()
		}
		err = respErr
	case SignedPayload:
		var jws string
		builder := signed_policy_cmd.NewCmdBuilder(ctx.LogicalClock, ctx.GetParams())
		builder.RegisterObject(action.Object)
		builder.Actor(action.Actor.DID)
		builder.CreationTimestamp(TimeToProto(ctx.Timestamp))
		builder.PolicyID(action.PolicyId)
		builder.SetSigner(action.Actor.Signer)
		jws, err = builder.BuildJWS(ctx)
		require.NoError(ctx.T, err)

		msg := &types.MsgSignedPolicyCmd{
			Creator: ctx.TxSigner.SourceHubAddr,
			Payload: jws,
			Type:    types.MsgSignedPolicyCmd_JWS,
		}
		resp, respErr := ctx.Executor.SignedPolicyCmd(ctx, msg)
		if resp != nil {
			result = resp.Result.GetRegisterObjectResult()
		}
		err = respErr
	case Direct:
		// For Direct Authentication we use the action Actor as the signer
		ctx.TxSigner = action.Actor
		msg := &types.MsgDirectPolicyCmd{
			Creator:      action.Actor.SourceHubAddr,
			PolicyId:     action.PolicyId,
			Cmd:          types.NewRegisterObjectCmd(action.Object),
			CreationTime: TimeToProto(ctx.Timestamp),
		}
		resp, respErr := ctx.Executor.DirectPolicyCmd(ctx, msg)
		if resp != nil {
			result = resp.Result.GetRegisterObjectResult()
		}
		err = respErr
	}
	return result, err
}

func unregisterObjectDispatcher(ctx *TestCtx, action *UnregisterObjectAction) (result *types.UnregisterObjectCmdResult, err error) {
	switch ctx.Strategy {
	case BearerToken:
		jws := genToken(ctx, action.Actor)
		msg := &types.MsgBearerPolicyCmd{
			Creator:      ctx.TxSigner.SourceHubAddr,
			BearerToken:  jws,
			PolicyId:     action.PolicyId,
			Cmd:          types.NewUnregisterObjectCmd(action.Object),
			CreationTime: TimeToProto(ctx.Timestamp),
		}
		resp, respErr := ctx.Executor.BearerPolicyCmd(ctx, msg)
		if resp != nil {
			result = resp.Result.GetUnregisterObjectResult()
		}
		err = respErr
	case SignedPayload:
		var jws string
		builder := signed_policy_cmd.NewCmdBuilder(ctx.LogicalClock, ctx.GetParams())
		builder.UnregisterObject(action.Object)
		builder.Actor(action.Actor.DID)
		builder.CreationTimestamp(TimeToProto(ctx.Timestamp))
		builder.PolicyID(action.PolicyId)
		builder.SetSigner(action.Actor.Signer)
		jws, err = builder.BuildJWS(ctx)
		require.NoError(ctx.T, err)

		msg := &types.MsgSignedPolicyCmd{
			Creator: ctx.TxSigner.SourceHubAddr,
			Payload: jws,
			Type:    types.MsgSignedPolicyCmd_JWS,
		}
		resp, respErr := ctx.Executor.SignedPolicyCmd(ctx, msg)
		if resp != nil {
			result = resp.Result.GetUnregisterObjectResult()
		}
		err = respErr
	case Direct:
		// For Direct Authentication we use the action Actor as the signer
		ctx.TxSigner = action.Actor
		msg := &types.MsgDirectPolicyCmd{
			Creator:      action.Actor.SourceHubAddr,
			PolicyId:     action.PolicyId,
			Cmd:          types.NewUnregisterObjectCmd(action.Object),
			CreationTime: TimeToProto(ctx.Timestamp),
		}
		resp, respErr := ctx.Executor.DirectPolicyCmd(ctx, msg)
		if resp != nil {
			result = resp.Result.GetUnregisterObjectResult()
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
