package policy_cmd

import (
	"fmt"

	"github.com/sourcenetwork/sourcehub/x/acp/commitment"
	"github.com/sourcenetwork/sourcehub/x/acp/registration"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
	"github.com/sourcenetwork/sourcehub/x/acp/utils"

	"github.com/sourcenetwork/acp_core/pkg/errors"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
)

// Handler acts as a common entrypoint to handle PolicyCmd objects.
// Handler can be used for direct, bearer and signed PolicyCmds
type Handler struct {
	engine              coretypes.ACPEngineServer
	registrationService *registration.RegistrationService
	commitmentService   *commitment.CommitmentService
}

// NewPolicyCmdHandler returns a handler for PolicyCmds
func NewPolicyCmdHandler(engine coretypes.ACPEngineServer,
	registrationService *registration.RegistrationService,
	commitmentService *commitment.CommitmentService,
) *Handler {
	return &Handler{
		engine:              engine,
		registrationService: registrationService,
		commitmentService:   commitmentService,
	}
}

// Dispatch consumes a PolicyCmd and returns its result
func (h *Handler) Dispatch(ctx *PolicyCmdCtx, cmd *types.PolicyCmd) (*types.PolicyCmdResult, error) {
	var err error
	ctx.Ctx, err = utils.InjectPrincipal(ctx.Ctx, ctx.PrincipalDID)
	if err != nil {
		return nil, err
	}

	switch c := cmd.Cmd.(type) {
	case *types.PolicyCmd_SetRelationshipCmd:
		return h.setRelationship(ctx, c.SetRelationshipCmd)
	case *types.PolicyCmd_DeleteRelationshipCmd:
		return h.deleteRelationship(ctx, c.DeleteRelationshipCmd)
	case *types.PolicyCmd_RegisterObjectCmd:
		return h.registerObject(ctx, c.RegisterObjectCmd)
	case *types.PolicyCmd_ArchiveObjectCmd:
		return h.archiveObject(ctx, c.ArchiveObjectCmd)
	case *types.PolicyCmd_CommitRegistrationsCmd:
		return h.commitRegistrations(ctx, c.CommitRegistrationsCmd)
	case *types.PolicyCmd_FlagHijackAttemptCmd:
		return h.flagHijackAttempt(ctx, c.FlagHijackAttemptCmd)
	case *types.PolicyCmd_RevealRegistrationCmd:
		return h.revealRegistration(ctx, c.RevealRegistrationCmd)
	case *types.PolicyCmd_UnarchiveObjectCmd:
		return h.unarchiveObject(ctx, c.UnarchiveObjectCmd)
	default:
		return nil, errors.Wrap("unsuported command", errors.ErrUnknownVariant, errors.Pair("command", c))
	}
}

func (h *Handler) setRelationship(ctx *PolicyCmdCtx, cmd *types.SetRelationshipCmd) (*types.PolicyCmdResult, error) {
	metadata, err := types.BuildACPSuppliedMetadata(ctx.Ctx, ctx.PrincipalDID, ctx.Signer)
	if err != nil {
		return nil, fmt.Errorf("marshaling metadata: %w", err)
	}

	resp, err := h.engine.SetRelationship(ctx.Ctx, &coretypes.SetRelationshipRequest{
		PolicyId:     ctx.PolicyId,
		Relationship: cmd.Relationship,
		Metadata:     metadata,
	})
	if err != nil {
		return nil, err
	}

	rec, err := types.MapRelationshipRecord(resp.Record)
	if err != nil {
		return nil, fmt.Errorf("mapping relationship record: %w", err)

	}

	return &types.PolicyCmdResult{
		Result: &types.PolicyCmdResult_SetRelationshipResult{
			SetRelationshipResult: &types.SetRelationshipCmdResult{
				RecordExisted: resp.RecordExisted,
				Record:        rec,
			},
		},
	}, nil
}

func (h *Handler) deleteRelationship(ctx *PolicyCmdCtx, cmd *types.DeleteRelationshipCmd) (*types.PolicyCmdResult, error) {
	resp, err := h.engine.DeleteRelationship(ctx.Ctx, &coretypes.DeleteRelationshipRequest{
		PolicyId:     ctx.PolicyId,
		Relationship: cmd.Relationship,
	})
	if err != nil {
		return nil, err
	}
	return &types.PolicyCmdResult{
		Result: &types.PolicyCmdResult_DeleteRelationshipResult{
			DeleteRelationshipResult: &types.DeleteRelationshipCmdResult{
				RecordFound: resp.RecordFound,
			},
		},
	}, nil
}

func (h *Handler) registerObject(ctx *PolicyCmdCtx, cmd *types.RegisterObjectCmd) (*types.PolicyCmdResult, error) {
	actor := coretypes.NewActor(ctx.PrincipalDID)
	resp, err := h.registrationService.RegisterObject(ctx.Ctx, ctx.PolicyId, cmd.Object, actor, ctx.Signer)
	if err != nil {
		return nil, err
	}

	r, err := types.MapRelationshipRecord(resp.Record)
	if err != nil {
		return nil, err
	}

	return &types.PolicyCmdResult{
		Result: &types.PolicyCmdResult_RegisterObjectResult{
			RegisterObjectResult: &types.RegisterObjectCmdResult{
				Record: r,
			},
		},
	}, nil
}

func (h *Handler) archiveObject(ctx *PolicyCmdCtx, cmd *types.ArchiveObjectCmd) (*types.PolicyCmdResult, error) {
	response, err := h.registrationService.ArchiveObject(ctx.Ctx, ctx.PolicyId, cmd.Object)
	if err != nil {
		return nil, err
	}
	return &types.PolicyCmdResult{
		Result: &types.PolicyCmdResult_ArchiveObjectResult{
			ArchiveObjectResult: &types.ArchiveObjectCmdResult{
				Found:                true,
				RelationshipsRemoved: response.RelationshipsRemoved,
			},
		},
	}, nil
}

func (h *Handler) commitRegistrations(ctx *PolicyCmdCtx, cmd *types.CommitRegistrationsCmd) (*types.PolicyCmdResult, error) {
	actor := coretypes.NewActor(ctx.PrincipalDID)
	commitment, err := h.commitmentService.SetNewCommitment(ctx.Ctx, ctx.PolicyId, cmd.Commitment, actor, &ctx.Params, ctx.Signer)
	if err != nil {
		return nil, err
	}

	return &types.PolicyCmdResult{
		Result: &types.PolicyCmdResult_CommitRegistrationsResult{
			CommitRegistrationsResult: &types.CommitRegistrationsCmdResult{
				RegistrationsCommitment: commitment,
			},
		},
	}, nil
}

func (h *Handler) revealRegistration(ctx *PolicyCmdCtx, cmd *types.RevealRegistrationCmd) (*types.PolicyCmdResult, error) {
	actor := coretypes.NewActor(ctx.PrincipalDID)

	rec, ev, err := h.registrationService.RevealRegistration(ctx.Ctx, cmd.RegistrationsCommitmentId, cmd.Proof, actor, ctx.Signer)
	if err != nil {
		return nil, err
	}
	r, err := types.MapRelationshipRecord(rec)
	if err != nil {
		return nil, err
	}

	return &types.PolicyCmdResult{
		Result: &types.PolicyCmdResult_RevealRegistrationResult{
			RevealRegistrationResult: &types.RevealRegistrationCmdResult{
				Record: r,
				Event:  ev,
			},
		},
	}, nil
}

func (h *Handler) flagHijackAttempt(ctx *PolicyCmdCtx, cmd *types.FlagHijackAttemptCmd) (*types.PolicyCmdResult, error) {
	actor := coretypes.NewActor(ctx.PrincipalDID)
	event, err := h.registrationService.FlagHijackEvent(ctx.Ctx, cmd.EventId, actor)
	if err != nil {
		return nil, err
	}
	return &types.PolicyCmdResult{
		Result: &types.PolicyCmdResult_FlagHijackAttemptResult{
			FlagHijackAttemptResult: &types.FlagHijackAttemptCmdResult{
				Event: event,
			},
		},
	}, nil
}

func (h *Handler) unarchiveObject(ctx *PolicyCmdCtx, cmd *types.UnarchiveObjectCmd) (*types.PolicyCmdResult, error) {
	resp, err := h.registrationService.UnarchiveObject(ctx.Ctx, ctx.PolicyId, cmd.Object)
	if err != nil {
		return nil, err
	}
	r, err := types.MapRelationshipRecord(resp.Record)
	if err != nil {
		return nil, err
	}

	return &types.PolicyCmdResult{
		Result: &types.PolicyCmdResult_UnarchiveObjectResult{
			UnarchiveObjectResult: &types.UnarchiveObjectCmdResult{
				Record:               r,
				RelationshipModified: resp.RecordModified,
			},
		},
	}, nil
}
