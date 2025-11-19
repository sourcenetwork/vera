package test

import (
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/stretchr/testify/require"

	"github.com/sourcenetwork/sourcehub/x/acp/commitment"
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
		Policy:      a.Policy,
		Creator:     a.Creator.SourceHubAddr,
		MarshalType: coretypes.PolicyMarshalingType_SHORT_YAML,
	}
	response, err := ctx.Executor.CreatePolicy(ctx, msg)

	AssertError(ctx, err, a.ExpectedErr)
	if a.Expected != nil {
		require.NotNil(ctx.T, response)
		AssertValue(ctx, response.Record.Policy, a.Expected)
	}
	if response != nil {
		ctx.State.PolicyCreator = a.Creator.SourceHubAddr
		ctx.State.PolicyId = response.Record.Policy.Id
		return response.Record.Policy
	}
	return nil
}

type EditPolicyAction struct {
	Id          string
	Policy      string
	Creator     *TestActor
	Expected    *coretypes.Policy
	ExpectedErr error
	Response    *types.MsgEditPolicyResponse
}

func (a *EditPolicyAction) Run(ctx *TestCtx) *coretypes.Policy {
	msg := &types.MsgEditPolicy{
		PolicyId:    a.Id,
		Policy:      a.Policy,
		Creator:     a.Creator.SourceHubAddr,
		MarshalType: coretypes.PolicyMarshalingType_SHORT_YAML,
	}
	response, err := ctx.Executor.EditPolicy(ctx, msg)
	a.Response = response

	AssertError(ctx, err, a.ExpectedErr)
	if a.Expected != nil {
		require.NotNil(ctx.T, response)
		AssertValue(ctx, response.Record.Policy, a.Expected)

		getResponse, getErr := ctx.Executor.Policy(ctx, &types.QueryPolicyRequest{
			Id: a.Id,
		})
		require.NoError(ctx.T, getErr)
		require.Equal(ctx.T, a.Expected, getResponse.Record.Policy)
	}
	if response != nil {
		ctx.State.PolicyCreator = a.Creator.SourceHubAddr
		ctx.State.PolicyId = response.Record.Policy.Id
		return response.Record.Policy
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

func (a *SetRelationshipAction) Run(ctx *TestCtx) *types.RelationshipRecord {
	cmd := types.NewSetRelationshipCmd(a.Relationship)
	result, err := dispatchPolicyCmd(ctx, a.PolicyId, a.Actor, cmd)
	if a.Expected != nil {
		a.Expected.Record.Metadata = ctx.GetRecordMetadataForActor(a.Actor.Name)
		want := &types.PolicyCmdResult{
			Result: &types.PolicyCmdResult_SetRelationshipResult{
				SetRelationshipResult: a.Expected,
			},
		}
		require.Equal(ctx.T, want, result)
	}
	AssertError(ctx, err, a.ExpectedErr)
	if err != nil {
		return nil
	}
	return result.GetSetRelationshipResult().Record
}

type RegisterObjectAction struct {
	PolicyId    string
	Object      *coretypes.Object
	Actor       *TestActor
	Expected    *types.RelationshipRecord
	ExpectedErr error
}

func (a *RegisterObjectAction) Run(ctx *TestCtx) *types.RelationshipRecord {
	cmd := types.NewRegisterObjectCmd(a.Object)
	result, err := dispatchPolicyCmd(ctx, a.PolicyId, a.Actor, cmd)
	if a.Expected != nil {
		a.Expected.Metadata = ctx.GetRecordMetadataForActor(a.Actor.Name)
		want := &types.PolicyCmdResult{
			Result: &types.PolicyCmdResult_RegisterObjectResult{
				RegisterObjectResult: &types.RegisterObjectCmdResult{
					Record: a.Expected,
				},
			},
		}
		ctx.T.Logf("want: %v", want)
		require.Equal(ctx.T, want, result)
	}
	AssertError(ctx, err, a.ExpectedErr)
	if err != nil {
		return nil
	}
	return result.GetRegisterObjectResult().Record
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
			PolicyId:     a.PolicyId,
			Relationship: rel,
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

type ArchiveObjectAction struct {
	PolicyId    string
	Object      *coretypes.Object
	Actor       *TestActor
	Expected    *types.ArchiveObjectCmdResult
	ExpectedErr error
}

func (a *ArchiveObjectAction) Run(ctx *TestCtx) *types.ArchiveObjectCmdResult {
	cmd := types.NewArchiveObjectCmd(a.Object)
	result, err := dispatchPolicyCmd(ctx, a.PolicyId, a.Actor, cmd)
	got := (*types.ArchiveObjectCmdResult)(nil)
	if result != nil {
		got = result.GetArchiveObjectResult()
	}
	AssertResults(ctx, got, a.Expected, err, a.ExpectedErr)
	return got
}

type UnarchiveObjectAction struct {
	PolicyId    string
	Object      *coretypes.Object
	Actor       *TestActor
	Expected    *types.UnarchiveObjectCmdResult
	ExpectedErr error
}

func (a *UnarchiveObjectAction) Run(ctx *TestCtx) *types.UnarchiveObjectCmdResult {
	cmd := types.NewUnarchiveObjectCmd(a.Object)
	result, err := dispatchPolicyCmd(ctx, a.PolicyId, a.Actor, cmd)
	if a.Expected != nil {
		want := &types.PolicyCmdResult{
			Result: &types.PolicyCmdResult_UnarchiveObjectResult{
				UnarchiveObjectResult: a.Expected,
			},
		}
		require.Equal(ctx.T, want, result)
	}
	AssertError(ctx, err, a.ExpectedErr)
	if err != nil {
		return nil
	}
	return result.GetUnarchiveObjectResult()
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
		action := RegisterObjectsAction{
			PolicyId: policy.Id,
			Objects:  objs,
			Actor:    ctx.GetActor(actorName),
		}
		action.Run(ctx)
	}

	for actorName, rels := range a.RelationshipsPerActor {
		action := SetRelationshipsAction{
			PolicyId:      policy.Id,
			Relationships: rels,
			Actor:         ctx.GetActor(actorName),
		}
		action.Run(ctx)
	}
}

type GetPolicyAction struct {
	Id          string
	Expected    *types.PolicyRecord
	ExpectedErr error
}

func (a *GetPolicyAction) Run(ctx *TestCtx) {
	msg := &types.QueryPolicyRequest{
		Id: a.Id,
	}
	result, err := ctx.Executor.Policy(ctx, msg)
	AssertError(ctx, err, a.ExpectedErr)
	if result != nil {
		a.Expected.Metadata.CreationTs = result.Record.Metadata.CreationTs
		AssertValue(ctx, a.Expected, result.Record)
	}
}

type CommitRegistrationsAction struct {
	PolicyId string
	Objects  []*coretypes.Object
	Actor    *TestActor
	Expected *types.RegistrationsCommitment
	// Commitment is optional and is automatically generated if Objects is provided
	Commitment  []byte
	ExpectedErr error
}

func (a *CommitRegistrationsAction) Run(ctx *TestCtx) *types.RegistrationsCommitment {
	if a.Objects != nil {
		actor := coretypes.NewActor(a.Actor.DID)
		commitment, err := commitment.GenerateCommitmentWithoutValidation(a.PolicyId, actor, a.Objects)
		require.NoError(ctx.T, err)
		a.Commitment = commitment
	}
	cmd := types.NewCommitRegistrationCmd(a.Commitment)
	result, err := dispatchPolicyCmd(ctx, a.PolicyId, a.Actor, cmd)
	AssertError(ctx, err, a.ExpectedErr)
	if a.ExpectedErr != nil {
		return nil
	}
	require.NotNil(ctx.T, result)
	comm := result.GetCommitRegistrationsResult().RegistrationsCommitment

	if a.Expected != nil {
		a.Expected.Metadata.CreationTs = comm.Metadata.CreationTs
		AssertValue(ctx, comm, a.Expected)
	}
	return comm
}

func (a *CommitRegistrationsAction) GetCommitment(ctx *TestCtx) []byte {
	actor := coretypes.NewActor(a.Actor.DID)
	commitment, err := commitment.GenerateCommitmentWithoutValidation(a.PolicyId, actor, a.Objects)
	require.NoError(ctx.T, err)
	return commitment
}

type RevealRegistrationAction struct {
	PolicyId     string
	CommitmentId uint64
	Objects      []*coretypes.Object
	Index        int
	Actor        *TestActor
	Expected     *types.RegisterObjectCmdResult
	ExpectedErr  error
}

func (a *RevealRegistrationAction) Run(ctx *TestCtx) *types.RevealRegistrationCmdResult {
	actor := coretypes.NewActor(a.Actor.DID)
	proof, err := commitment.ProofForObject(a.PolicyId, actor, a.Index, a.Objects)
	require.NoError(ctx.T, err)
	cmd := types.NewRevealRegistrationCmd(a.CommitmentId, proof)
	result, err := dispatchPolicyCmd(ctx, a.PolicyId, a.Actor, cmd)
	got := (*types.RevealRegistrationCmdResult)(nil)
	if result != nil {
		got = result.GetRevealRegistrationResult()
	}
	AssertResults(ctx, got, a.Expected, err, a.ExpectedErr)
	return got
}
