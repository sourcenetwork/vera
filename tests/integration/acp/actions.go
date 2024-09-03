package test

import (
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"

	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

type CreatePolicyAction struct {
	Policy      string
	Expected    *coretypes.Policy
	Creator     *TestActor
	ExpectedErr error
}

func (a *CreatePolicyAction) Run(ctx *TestCtx) *coretypes.Policy {
	msg := &types.MsgCreatePolicy{
		Policy:       a.Policy,
		Creator:      a.Creator.SourceHubAddr,
		MarshalType:  coretypes.PolicyMarshalingType_SHORT_YAML,
		CreationTime: TimeToProto(ctx.Timestamp),
	}
	response, err := ctx.Executor.CreatePolicy(ctx, msg)

	var expected any = nil
	if a.Expected != nil {
		expected = &types.MsgCreatePolicyResponse{
			Policy: a.Expected,
		}
	}
	AssertResults(ctx, response, expected, err, a.ExpectedErr)
	if response != nil {
		ctx.State.PolicyCreator = a.Creator.SourceHubAddr
		ctx.State.PolicyId = response.Policy.Id
		return response.Policy
	}
	return nil
}

type SetRelationshipAction struct {
	PolicyId     string
	Relationship *coretypes.Relationship
	Actor        *TestActor
	Expected     *types.SetRelationshipCmdResult
	ExpectedErr  error
}

func (a *SetRelationshipAction) Run(ctx *TestCtx) *coretypes.RelationshipRecord {
	result, err := setRelationshipDispatcher(ctx, a)
	AssertResults(ctx, result, a.Expected, err, a.ExpectedErr)
	if result != nil {
		return result.Record
	}
	return nil
}

type RegisterObjectAction struct {
	PolicyId    string
	Object      *coretypes.Object
	Actor       *TestActor
	Expected    *types.RegisterObjectCmdResult
	ExpectedErr error
}

func (a *RegisterObjectAction) Run(ctx *TestCtx) *coretypes.RelationshipRecord {
	result, err := registerObjectDispatcher(ctx, a)
	AssertResults(ctx, result, a.Expected, err, a.ExpectedErr)
	if result != nil {
		return result.Record
	}
	return nil
}

type RegisterObjectsAction struct {
	PolicyId string
	Objects  []*coretypes.Object
	Actor    *TestActor
}

func (a *RegisterObjectsAction) Run(ctx *TestCtx) {
	for _, obj := range a.Objects {
		action := RegisterObjectAction{
			PolicyId: a.PolicyId,
			Object:   obj,
			Actor:    a.Actor,
		}
		action.Run(ctx)
	}
}

type SetRelationshipsAction struct {
	PolicyId      string
	Relationships []*coretypes.Relationship
	Actor         *TestActor
}

func (a *SetRelationshipsAction) Run(ctx *TestCtx) {
	for _, rel := range a.Relationships {
		action := SetRelationshipAction{
			Relationship: rel,
			PolicyId:     a.PolicyId,
			Actor:        a.Actor,
		}
		action.Run(ctx)
	}
}

type DeleteRelationshipsAction struct {
	PolicyId      string
	Relationships []*coretypes.Relationship
	Actor         *TestActor
}

func (a *DeleteRelationshipsAction) Run(ctx *TestCtx) {
	for _, rel := range a.Relationships {
		action := DeleteRelationshipAction{
			Relationship: rel,
			PolicyId:     a.PolicyId,
			Actor:        a.Actor,
		}
		action.Run(ctx)
	}
}

type DeleteRelationshipAction struct {
	PolicyId     string
	Relationship *coretypes.Relationship
	Actor        *TestActor
	Expected     *types.DeleteRelationshipCmdResult
	ExpectedErr  error
}

func (a *DeleteRelationshipAction) Run(ctx *TestCtx) *types.DeleteRelationshipCmdResult {
	result, err := deleteRelationshipDispatcher(ctx, a)
	AssertResults(ctx, result, a.Expected, err, a.ExpectedErr)
	return result
}

type UnregisterObjectAction struct {
	PolicyId    string
	Object      *coretypes.Object
	Actor       *TestActor
	Expected    *types.UnregisterObjectCmdResult
	ExpectedErr error
}

func (a *UnregisterObjectAction) Run(ctx *TestCtx) *types.UnregisterObjectCmdResult {
	result, err := unregisterObjectDispatcher(ctx, a)
	AssertResults(ctx, result, a.Expected, err, a.ExpectedErr)
	return result
}

type PolicySetupAction struct {
	Policy                string
	PolicyCreator         *TestActor
	ObjectsPerActor       map[string][]*coretypes.Object
	RelationshipsPerActor map[string][]*coretypes.Relationship
}

func (a *PolicySetupAction) Run(ctx *TestCtx) {
	polAction := CreatePolicyAction{
		Policy:  a.Policy,
		Creator: a.PolicyCreator,
	}
	policy := polAction.Run(ctx)

	for actorName, objs := range a.ObjectsPerActor {
		actor := ctx.GetActor(actorName)
		action := RegisterObjectsAction{
			Objects:  objs,
			Actor:    actor,
			PolicyId: policy.Id,
		}
		action.Run(ctx)
	}

	for actorName, rels := range a.RelationshipsPerActor {
		actor := ctx.GetActor(actorName)
		action := SetRelationshipsAction{
			Relationships: rels,
			Actor:         actor,
			PolicyId:      policy.Id,
		}
		action.Run(ctx)
	}
}

type GetPolicyAction struct {
	Id          string
	Expected    *types.QueryPolicyResponse
	ExpectedErr error
}

func (a *GetPolicyAction) Run(ctx *TestCtx) {
	msg := &types.QueryPolicyRequest{
		Id: a.Id,
	}
	result, err := ctx.Executor.Policy(ctx, msg)

	AssertResults(ctx, result, a.Expected, err, a.ExpectedErr)
}
