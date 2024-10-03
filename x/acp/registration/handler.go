package registration

import (
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	prototypes "github.com/cosmos/gogoproto/types"
	"github.com/sourcenetwork/acp_core/pkg/errors"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	raccoon "github.com/sourcenetwork/raccoondb"

	"github.com/sourcenetwork/sourcehub/x/acp/types"
	"github.com/sourcenetwork/sourcehub/x/acp/utils"
)

// commitmentLen is a Sha256 Hash, meaning we expect 32 bytes
const commitmentLen int = 256 / 8

type CommitRegistrationsHandler struct{}

func (h *CommitRegistrationsHandler) Handle(
	ctx sdk.Context,
	engine coretypes.ACPEngineServer,
	repository CommitmentRepository,
	registrationIdCounter *raccoon.CounterStore,
	params *types.Params,
	actor *coretypes.Actor,
	cmd *types.CommitRegistrationsCmd) (*types.CommitRegistrationsCmdResult, error) {
	_, err := engine.GetPolicy(ctx, &coretypes.GetPolicyRequest{
		Id: cmd.PolicyId,
	})
	if err != nil {
		return nil, err
	}

	if len(cmd.Commitment) != commitmentLen {
		return nil, newErrInvalidCommitment(cmd.PolicyId, cmd.Commitment)
	}

	releaser := registrationIdCounter.Acquire()
	defer releaser.Release()

	id, err := registrationIdCounter.GetNext(ctx)
	if err != nil {
		return nil, errors.NewFromBaseError(err, errors.ErrorType_INTERNAL, "fail generating commitment id")
	}

	creationTime, err := prototypes.TimestampProto(ctx.BlockTime())
	if err != nil {
		return nil, err
	}

	expiration, err := h.calculationExpirationTime(ctx.BlockTime(), params.RegistrationsCommitmentValiditySecs)
	if err != nil {
		return nil, err
	}

	registration := &types.RegistrationsCommitment{
		Id:             fmt.Sprintf("%v", id),
		PolicyId:       cmd.PolicyId,
		Actor:          actor,
		Commitment:     cmd.Commitment,
		Expired:        false,
		TxHash:         utils.HashTx(ctx.TxBytes()),
		CreationHeight: uint64(ctx.BlockHeight()),
		CreationTime:   creationTime,
		ExpirationTime: expiration,
	}

	err = repository.Set(ctx, registration)
	if err != nil {
		return nil, err
	}

	err = registrationIdCounter.Increment(ctx)
	if err != nil {
		return nil, errors.NewFromBaseError(err, errors.ErrorType_INTERNAL, "incrementing commitment id")
	}

	return &types.CommitRegistrationsCmdResult{
		RegistrationsCommitment: registration,
	}, nil
}

func (h *CommitRegistrationsHandler) calculationExpirationTime(now time.Time, offsetSecs uint64) (*prototypes.Timestamp, error) {
	delta := time.Second * time.Duration(offsetSecs)
	return prototypes.TimestampProto(now.Add(delta))
}

// TODO finish registration handler

type RevealRegistrationHandler struct{}

func (h *RevealRegistrationHandler) Handle(
	ctx sdk.Context,
	registrationService *registrationService,
	eventService *eventService,
	engine coretypes.ACPEngineServer,
	repository CommitmentRepository,
	actor *coretypes.Actor,
	cmd *types.RevealRegistrationCmd) (*types.RevealRegistrationCmdResult, error) {
	commitment, err := repository.GetById(ctx, cmd.RegistrationsCommitmentId)
	if err != nil {
		return nil, err
	}
	if commitment == nil {
		return nil, errors.Wrap("RegistrationsCommimtnet", errors.ErrorType_NOT_FOUND,
			errors.Pair("id", cmd.RegistrationsCommitmentId))
	}

	ok := VerifyProof(commitment.Commitment, commitment.PolicyId, commitment.Actor, cmd.Proof)
	if !ok {
		return nil, errors.Wrap("invalid proof", errors.ErrorType_BAD_INPUT)
	}

	registrationRecord, err := engine.GetObjectRegistration(ctx, &coretypes.GetObjectRegistrationRequest{
		PolicyId: commitment.PolicyId,
		Object:   cmd.Object,
	})
	if err != nil {
		return nil, err
	}

	if registrationRecord.IsRegistered {
		return h.registeredStrategy(
			ctx,
			registrationService,
			eventService,
			engine,
			repository,
			actor,
			commitment,
			registrationRecord,
			cmd,
		)
	} else {
		return h.unregisteredStrategy(
			ctx,
			registrationService,
			engine,
			repository,
			actor,
			commitment,
			registrationRecord,
			cmd,
		)
	}
}

func (h *RevealRegistrationHandler) registeredStrategy(
	ctx sdk.Context,
	registrationService *registrationService,
	eventService *eventService,
	engine coretypes.ACPEngineServer,
	repository CommitmentRepository,
	actor *coretypes.Actor,
	commitment *types.RegistrationsCommitment,
	registrationStatus *coretypes.GetObjectRegistrationResponse,
	cmd *types.RevealRegistrationCmd) (*types.RevealRegistrationCmdResult, error) {
	if registrationStatus.OwnerId == actor.Id {
		isArchived := false // FIXME use data in registration status instead
		if !isArchived {
			// NOOP
			return &types.RevealRegistrationCmdResult{
				Result: types.RegistrationResultStatus_NO_OP,
				Record: nil, // FIXME registrationStatus.Record
				Event:  nil,
			}, nil
		}

		record, event, err := registrationService.UnarchiveObject(ctx, commitment.PolicyId, cmd.Object, actor)
		if err != nil {
			return nil, err
		}
		return &types.RevealRegistrationCmdResult{
			Result: types.RegistrationResultStatus_UNARCHIVED,
			Record: record,
			Event:  event,
		}, nil
	}

	// FIXME this name is bad because due to how i called all events are registration events.
	// refactor event to simply ObjectEvent
	event, err := eventService.GetLatestRegistrationEvent(ctx, commitment.PolicyId, cmd.Object)
	if err != nil {
		return nil, err
	}
	// this shouldn't happen because the object was verified
	// to be registered beforehand.
	// return an internal error if it ever does
	if event == nil {
		return nil, errors.ErrorType_INTERNAL
	}

	claimHeight := event.Height
	if event.Type == types.ObjectRegistrationEventType_REVEAL_REGISTRATION {
		claimHeight = event.Detail.GetRevealEvent().CommitmentCreationHeight
	} else if event.Type == types.ObjectRegistrationEventType_AMENDMENT {
		claimHeight = event.Detail.GetAmendmentEvent().CommitmentCreationHeight
	}

	if commitment.CreationHeight < claimHeight {
		rec, ev, err := registrationService.AmendRegistration(ctx, commitment.Id, commitment.PolicyId, cmd.Object, actor)
		if err != nil {
			return nil, err
		}
		return &types.RevealRegistrationCmdResult{
			Record: rec,
			Event:  ev,
			Result: types.RegistrationResultStatus_AMENDED,
		}, nil
	}

	return nil, errors.Wrap("object already registered", errors.ErrorType_UNAUTHORIZED,
		errors.Pair("policy", commitment.PolicyId),
		errors.Pair("resource", cmd.Object.Resource),
		errors.Pair("object", cmd.Object.Id),
	)
}

func (h *RevealRegistrationHandler) unregisteredStrategy(
	ctx sdk.Context,
	registrationService *registrationService,
	engine coretypes.ACPEngineServer,
	repository CommitmentRepository,
	actor *coretypes.Actor,
	commitment *types.RegistrationsCommitment,
	registrationStatus *coretypes.GetObjectRegistrationResponse,
	cmd *types.RevealRegistrationCmd) (*types.RevealRegistrationCmdResult, error) {
	record, event, err := registrationService.RegisterObject(ctx, commitment.PolicyId, cmd.Object, actor)
	if err != nil {
		return nil, err
	}

	return &types.RevealRegistrationCmdResult{
		Result: types.RegistrationResultStatus_OK,
		Record: record,
		Event:  event,
	}, nil
}

type FlagHijackAttemptHandler struct{}

func (h *FlagHijackAttemptHandler) Handle(
	ctx sdk.Context,
	service *eventService,
	actor *coretypes.Actor,
	cmd *types.FlagHijackAttemptCmd) (*types.FlagHijackAttemptCmdResult, error) {
	event, err := service.FlagHijackEvent(ctx, cmd.EventId, actor)
	if err != nil {
		return nil, err
	}

	return &types.FlagHijackAttemptCmdResult{
		Event: event,
	}, nil
}

// FlagExpiredCommitments iterates over stored commitments,
// filters for expired commitments wrt the current block time,
// flags them as expired and returns the expired commitments
func FlagExpiredCommitments(ctx sdk.Context, repository CommitmentRepository) ([]*types.RegistrationsCommitment, error) {
	commitments, err := repository.GetExpiredCommitments(ctx, ctx.BlockTime())
	if err != nil {
		return nil, err
	}
	processed := make([]*types.RegistrationsCommitment, 0, len(commitments))
	for _, c := range commitments {
		if c.Expired {
			continue
		}
		c.Expired = true
		err := repository.Set(ctx, c)
		if err != nil {
			return nil, err
		}
		processed = append(processed, c)

	}
	return commitments, nil
}
