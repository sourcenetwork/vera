package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	prototypes "github.com/cosmos/gogoproto/types"
	"github.com/sourcenetwork/acp_core/pkg/auth"
	"github.com/sourcenetwork/acp_core/pkg/errors"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"

	"github.com/sourcenetwork/sourcehub/x/acp/registration"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

func dispatchPolicyCmd(ctx sdk.Context, k *Keeper, policyId string, authenticatedActor string, ts *prototypes.Timestamp, cmd *types.PolicyCmd) (*types.PolicyCmdResult, error) {
	engine, err := k.GetACPEngine(ctx)
	if err != nil {
		return nil, err
	}
	commitmentRepository := k.GetRegistrationsCommitmentRepository(ctx)
	objectRepository := k.GetObjectEventRepository(ctx)
	eventService := registration.NewEventService(objectRepository)
	registrationService := registration.NewRegistrationService(engine, eventService, commitmentRepository)

	result := &types.PolicyCmdResult{}

	actor := coretypes.NewActor(authenticatedActor)
	principal, err := auth.NewDIDPrincipal(authenticatedActor)
	if err != nil {
		return nil, err
	}
	goCtx := auth.InjectPrincipal(ctx.Context(), principal)

	switch c := cmd.Cmd.(type) {
	case *types.PolicyCmd_SetRelationshipCmd:
		resp, respErr := engine.SetRelationship(goCtx, &coretypes.SetRelationshipRequest{
			PolicyId:     policyId,
			CreationTime: ts,
			Relationship: c.SetRelationshipCmd.Relationship,
		})
		if respErr != nil {
			err = respErr
			break
		}
		result.Result = &types.PolicyCmdResult_SetRelationshipResult{
			SetRelationshipResult: &types.SetRelationshipCmdResult{
				RecordExisted: resp.RecordExisted,
				Record:        resp.Record,
			},
		}
	case *types.PolicyCmd_DeleteRelationshipCmd:
		resp, respErr := engine.DeleteRelationship(goCtx, &coretypes.DeleteRelationshipRequest{
			PolicyId:     policyId,
			Relationship: c.DeleteRelationshipCmd.Relationship,
		})
		if respErr != nil {
			err = respErr
			break
		}
		result.Result = &types.PolicyCmdResult_DeleteRelationshipResult{
			DeleteRelationshipResult: &types.DeleteRelationshipCmdResult{
				RecordFound: resp.RecordFound,
			},
		}
	case *types.PolicyCmd_RegisterObjectCmd:
		resp, respErr := engine.RegisterObject(goCtx, &coretypes.RegisterObjectRequest{
			PolicyId:     policyId,
			CreationTime: ts,
			Object:       c.RegisterObjectCmd.Object,
		})
		if respErr != nil {
			err = respErr
			break
		}
		result.Result = &types.PolicyCmdResult_RegisterObjectResult{
			RegisterObjectResult: &types.RegisterObjectCmdResult{
				Result: resp.Result,
				Record: resp.Record,
			},
		}
	case *types.PolicyCmd_UnregisterObjectCmd:
		resp, respErr := engine.UnregisterObject(goCtx, &coretypes.UnregisterObjectRequest{
			PolicyId: policyId,
			Object:   c.UnregisterObjectCmd.Object,
		})
		if respErr != nil {
			err = respErr
			break
		}
		result.Result = &types.PolicyCmdResult_UnregisterObjectResult{
			UnregisterObjectResult: &types.UnregisterObjectCmdResult{
				Found:                resp.Found,
				RelationshipsRemoved: resp.RelationshipsRemoved,
			},
		}
	case *types.PolicyCmd_CommitRegistrationsCmd:
		regCmd := c.CommitRegistrationsCmd
		params := k.GetParams(ctx)
		commitment, respErr := registrationService.CommitRegistration(ctx, policyId, regCmd.Commitment, actor, &params)
		if respErr != nil {
			err = respErr
			break
		}
		result.Result = &types.PolicyCmdResult_CommitRegistrationsResult{
			CommitRegistrationsResult: &types.CommitRegistrationsCmdResult{
				RegistrationsCommitment: commitment,
			},
		}
	case *types.PolicyCmd_FlagHijackAttemptCmd:
		event, respErr := eventService.FlagHijackEvent(ctx, c.FlagHijackAttemptCmd.EventId, actor)
		if respErr != nil {
			err = respErr
			break
		}
		result.Result = &types.PolicyCmdResult_FlagHijackAttemptResult{
			FlagHijackAttemptResult: &types.FlagHijackAttemptCmdResult{
				Event: event,
			},
		}
	case *types.PolicyCmd_RevealRegistrationCmd:
		regCmd := c.RevealRegistrationCmd
		rec, ev, respErr := registrationService.RevealRegistration(ctx, regCmd.RegistrationsCommitmentId, regCmd.Proof, regCmd.Object, actor)
		if respErr != nil {
			err = respErr
			break
		}
		result.Result = &types.PolicyCmdResult_RevealRegistrationResult{
			RevealRegistrationResult: &types.RevealRegistrationCmdResult{
				Record: rec,
				Event:  ev,
				Result: types.RegistrationResultStatus_UNARCHIVED, // TODO
			},
		}
	default:
		err = errors.Wrap("unsuported command", errors.ErrUnknownVariant, errors.Pair("command", c))
	}
	if err != nil {
		return nil, err
	}
	return result, nil
}
