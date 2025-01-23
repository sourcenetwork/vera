package policy_cmd

import (
	"fmt"

	"github.com/sourcenetwork/sourcehub/x/acp/registration"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
	"github.com/sourcenetwork/sourcehub/x/acp/utils"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sourcenetwork/acp_core/pkg/errors"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
)

func NewRecordMetadata(ctx sdk.Context, txSigner string, actor string) (*types.RecordMetadata, error) {
	ts, err := types.TimestampFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	return &types.RecordMetadata{
		CreationTs: ts,
		TxHash:     utils.HashTx(ctx.TxBytes()),
		TxSigner:   txSigner,
		OwnerDid:   actor,
	}, nil
}

type Handler struct {
	engine              coretypes.ACPEngineServer
	eventService        registration.EventService
	registrationService registration.RegistrationService
	commitmentService   registration.CommitmentService
}

func (h *Handler) Dispatch(ctx *PolicyCmdCtx, cmd *types.PolicyCmd) (*types.PolicyCmdResult, error) {
	switch c := cmd.Cmd.(type) {
	case *types.PolicyCmd_SetRelationshipCmd:
		return h.SetRelationship(ctx, c.SetRelationshipCmd)
	case *types.PolicyCmd_DeleteRelationshipCmd:
		return h.DeleteRelationship(ctx, c.DeleteRelationshipCmd)
	case *types.PolicyCmd_RegisterObjectCmd:
		return h.RegisterObject(ctx, c.RegisterObjectCmd)
	case *types.PolicyCmd_ArchiveObjectCmd:
		return h.ArchiveObject(ctx, c.ArchiveObjectCmd)
	case *types.PolicyCmd_CommitRegistrationsCmd:
		return h.CommitRegistrations(ctx, c.CommitRegistrationsCmd)
	case *types.PolicyCmd_FlagHijackAttemptCmd:
		return h.FlagHijackAttempt(ctx, c.FlagHijackAttemptCmd)
	case *types.PolicyCmd_RevealRegistrationCmd:
		return h.RevealRegistration(ctx, c.RevealRegistrationCmd)
	case *types.PolicyCmd_UnarchiveObjectCmd:
		return h.UnarchiveObject(ctx, c.UnarchiveObjectCmd)
	default:
		return nil, errors.Wrap("unsuported command", errors.ErrUnknownVariant, errors.Pair("command", c))
	}
}

func (h *Handler) SetRelationship(ctx *PolicyCmdCtx, cmd *types.SetRelationshipCmd) (*types.PolicyCmdResult, error) {
	metadata, err := NewRecordMetadata(ctx.SDKCtx, ctx.Signer, ctx.PrincipalDID)
	if err != nil {
		return nil, err
	}
	bytes, err := metadata.Marshal()
	if err != nil {
		return nil, fmt.Errorf("marshaling metadata: %w", err)
	}

	resp, err := h.engine.SetRelationship(ctx.EngineContext, &coretypes.SetRelationshipRequest{
		PolicyId:     ctx.PolicyId,
		Relationship: cmd.Relationship,
		Metadata: &coretypes.SuppliedMetadata{
			Blob: bytes,
		},
	})
	if err != nil {
		return nil, err
	}

	rec, err := MapRelationshipRecord(resp.Record)
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

func (h *Handler) DeleteRelationship(ctx *PolicyCmdCtx, cmd *types.DeleteRelationshipCmd) (*types.PolicyCmdResult, error) {
	resp, err := h.engine.DeleteRelationship(ctx.EngineContext, &coretypes.DeleteRelationshipRequest{
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

func (h *Handler) RegisterObject(ctx *PolicyCmdCtx, cmd *types.RegisterObjectCmd) (*types.PolicyCmdResult, error) {
	actor := coretypes.NewActor(ctx.PrincipalDID)
	rec, _, err := h.registrationService.RegisterObject(ctx.SDKCtx, ctx.PolicyId, cmd.Object, actor, ctx.Signer)
	if err != nil {
		return nil, err
	}

	r, err := MapRelationshipRecord(rec)
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

func (h *Handler) ArchiveObject(ctx *PolicyCmdCtx, cmd *types.ArchiveObjectCmd) (*types.PolicyCmdResult, error) {
	resp, err := h.engine.ArchiveObject(ctx.EngineContext, &coretypes.ArchiveObjectRequest{
		PolicyId: ctx.PolicyId,
		Object:   cmd.Object,
	})
	if err != nil {
		return nil, err
	}
	return &types.PolicyCmdResult{

		Result: &types.PolicyCmdResult_ArchiveObjectResult{
			ArchiveObjectResult: &types.ArchiveObjectCmdResult{
				Found:                true,
				RelationshipsRemoved: resp.RelationshipsRemoved,
			},
		},
	}, nil
}

func (h *Handler) CommitRegistrations(ctx *PolicyCmdCtx, cmd *types.CommitRegistrationsCmd) (*types.PolicyCmdResult, error) {
	actor := coretypes.NewActor(ctx.PrincipalDID)
	commitment, err := h.commitmentService.SetNewCommitment(ctx.SDKCtx, ctx.PolicyId, cmd.Commitment, actor, &ctx.Params, ctx.Signer)
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

func (h *Handler) RevealRegistration(ctx *PolicyCmdCtx, cmd *types.RevealRegistrationCmd) (*types.PolicyCmdResult, error) {
	actor := coretypes.NewActor(ctx.PrincipalDID)
	rec, ev, err := h.registrationService.RevealRegistration(ctx.SDKCtx, cmd.RegistrationsCommitmentId, cmd.Proof, actor, ctx.Signer)
	if err != nil {
		return nil, err
	}
	r, err := MapRelationshipRecord(rec)
	if err != nil {
		return nil, err
	}

	return &types.PolicyCmdResult{
		Result: &types.PolicyCmdResult_RevealRegistrationResult{
			RevealRegistrationResult: &types.RevealRegistrationCmdResult{
				Record: r,
				Event:  ev,
				Result: types.RegistrationResultStatus_UNARCHIVED, // TODO
			},
		},
	}, nil
}

func (h *Handler) FlagHijackAttempt(ctx *PolicyCmdCtx, cmd *types.FlagHijackAttemptCmd) (*types.PolicyCmdResult, error) {
	actor := coretypes.NewActor(ctx.PrincipalDID)
	event, err := h.eventService.FlagHijackEvent(ctx.SDKCtx, cmd.EventId, actor)
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

func (h *Handler) UnarchiveObject(ctx *PolicyCmdCtx, cmd *types.UnarchiveObjectCmd) (*types.PolicyCmdResult, error) {
	actor := coretypes.NewActor(ctx.PrincipalDID)
	rec, ev, err := h.registrationService.UnarchiveObject(ctx.SDKCtx, ctx.PolicyId, cmd.Object, actor, ctx.Signer)
	if err != nil {
		return nil, err
	}
	r, err := MapRelationshipRecord(rec)
	if err != nil {
		return nil, err
	}

	return &types.PolicyCmdResult{
		Result: &types.PolicyCmdResult_UnarchiveObjectResult{
			UnarchiveObjectResult: &types.UnarchiveObjectCmdResult{
				Record:               r,
				RelationshipModified: ev != nil,
			},
		},
	}, nil
}
