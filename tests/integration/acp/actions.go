package test

import (
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/stretchr/testify/require"

	"github.com/sourcenetwork/sourcehub/x/acp/registration"
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
	cmd := types.NewSetRelationshipCmd(a.Relationship)
	result, err := dispatchPolicyCmd(ctx, a.PolicyId, a.Actor, cmd)
	got := (*types.SetRelationshipCmdResult)(nil)
	if result != nil {
		got = result.GetSetRelationshipResult()
	}
	AssertResults(ctx, got, a.Expected, err, a.ExpectedErr)
	if got != nil {
		return got.Record
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
	cmd := types.NewRegisterObjectCmd(a.Object)
	result, err := dispatchPolicyCmd(ctx, a.PolicyId, a.Actor, cmd)
	got := (*types.RegisterObjectCmdResult)(nil)
	if result != nil {
		got = result.GetRegisterObjectResult()
	}
	AssertResults(ctx, got, a.Expected, err, a.ExpectedErr)
	if result != nil {
		return got.Record
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
	cmd := types.NewDeleteRelationshipCmd(a.Relationship)
	result, err := dispatchPolicyCmd(ctx, a.PolicyId, a.Actor, cmd)
	got := (*types.DeleteRelationshipCmdResult)(nil)
	if result != nil {
		got = result.GetDeleteRelationshipResult()
	}
	AssertResults(ctx, got, a.Expected, err, a.ExpectedErr)
	return got
}

type UnregisterObjectAction struct {
	PolicyId    string
	Object      *coretypes.Object
	Actor       *TestActor
	Expected    *types.ArchiveObjectCmdResult
	ExpectedErr error
}

func (a *UnregisterObjectAction) Run(ctx *TestCtx) *types.ArchiveObjectCmdResult {
	cmd := types.NewUnregisterObjectCmd(a.Object)
	result, err := dispatchPolicyCmd(ctx, a.PolicyId, a.Actor, cmd)
	got := (*types.ArchiveObjectCmdResult)(nil)
	if result != nil {
		got = result.GetUnregisterObjectResult()
	}
	AssertResults(ctx, got, a.Expected, err, a.ExpectedErr)
	return got
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

type CommitRegistrationsAction struct {
	PolicyId    string
	Objects     []*coretypes.Object
	Actor       *TestActor
	Expected    *types.RegistrationsCommitment
	commitment  []byte
	ExpectedErr error
}

func (a *CommitRegistrationsAction) Run(ctx *TestCtx) *types.RegistrationsCommitment {
	actor := coretypes.NewActor(a.Actor.DID)
	commitment, err := registration.GenerateCommitmentWithoutValidation(a.PolicyId, actor, a.Objects)
	require.NoError(ctx.T, err)
	cmd := types.NewCommitRegistrationCmd(commitment)
	result, err := dispatchPolicyCmd(ctx, a.PolicyId, a.Actor, cmd)
	got := (*types.CommitRegistrationsCmdResult)(nil)
	if result != nil {
		got = result.GetCommitRegistrationsResult()
	}
	AssertResults(ctx, got, a.Expected, err, a.ExpectedErr)
	if result != nil {
		return got.RegistrationsCommitment
	}
	return nil
}

type RevealRegistrationAction struct {
	PolicyId     string
	CommitmentId string
	Objects      []*coretypes.Object
	Index        int
	Actor        *TestActor
	Expected     *types.RegisterObjectCmdResult
	ExpectedErr  error
}

func (a *RevealRegistrationAction) Run(ctx *TestCtx) *types.RevealRegistrationCmdResult {
	actor := coretypes.NewActor(a.Actor.DID)
	proof, err := registration.ProofForObject(a.PolicyId, actor, a.Index, a.Objects)
	require.NoError(ctx.T, err)
	cmd := types.NewRevealRegistrationCmd(a.CommitmentId, proof, a.Objects[a.Index])
	result, err := dispatchPolicyCmd(ctx, a.PolicyId, a.Actor, cmd)
	got := (*types.RevealRegistrationCmdResult)(nil)
	if result != nil {
		got = result.GetRevealRegistrationResult()
	}
	AssertResults(ctx, got, a.Expected, err, a.ExpectedErr)
	return got
}
