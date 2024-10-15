package registration

import (
	"fmt"
	"slices"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sourcenetwork/acp_core/pkg/auth"
	"github.com/sourcenetwork/acp_core/pkg/errors"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	acputils "github.com/sourcenetwork/acp_core/pkg/utils"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
	"github.com/sourcenetwork/sourcehub/x/acp/utils"
)

// commitmentLen is a Sha256 Hash, meaning we expect 32 bytes
const commitmentLen int = 256 / 8

func NewEventService(repo RegistrationEventRepository) *EventService {
	return &EventService{
		repo: repo,
	}
}

// EventService provides operations over Object Status Events
type EventService struct {
	repo RegistrationEventRepository
}

// NewEvent creates and stores a new ObjectStatusEvents with the given information
func (s *EventService) NewEvent(ctx sdk.Context, t types.ObjectRegistrationEventType, polId string, object *coretypes.Object, actor *coretypes.Actor, detail *types.EventDetail) (*types.ObjectRegistrationEvent, error) {
	id, err := s.repo.IncrementId(ctx)
	if err != nil {
		return nil, err
	}

	ts, err := types.TimestampFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	event := &types.ObjectRegistrationEvent{
		Id:       fmt.Sprintf("%v", id),
		Type:     t,
		TxHash:   utils.HashTx(ctx.TxBytes()),
		Ts:       ts,
		PolicyId: polId,
		Object:   object,
		Actor:    actor,
		Detail:   detail,
	}

	err = s.repo.Set(ctx, event)
	if err != nil {
		return nil, err
	}

	return event, nil
}

func (s *EventService) GetLatestRegistrationEvent(ctx sdk.Context, polId string, object *coretypes.Object) (*types.ObjectRegistrationEvent, error) {
	evs, err := s.repo.GetObjectEvents(ctx, polId, object)
	if err != nil {
		return nil, err
	}
	evs = acputils.FilterSlice(evs, func(ev *types.ObjectRegistrationEvent) bool {
		return slices.Contains(types.ObjectClaimEvents, ev.Type)
	})

	if len(evs) == 0 {
		return nil, nil
	}

	// ACP should maintain an invariant where, after the first claim event,
	// any further object amendment can only reference an ealier commitment.
	// this means that the lastest event is the one that corresponds to the earliest registration"
	sortable := acputils.FromExtractor(evs, func(ev *types.ObjectRegistrationEvent) uint64 {
		return ev.Ts.BlockHeight
	})
	sortable.SortInPlace()

	return evs[0], nil
}

func (s *EventService) FlagHijackEvent(ctx sdk.Context, eventId string, actor *coretypes.Actor) (*types.ObjectRegistrationEvent, error) {
	event, err := s.repo.GetById(ctx, eventId)
	if err != nil {
		return nil, err
	}
	if event.Type != types.ObjectRegistrationEventType_AMENDMENT {
		return nil, errors.Wrap("event must be of type AMENDMENT", errors.ErrorType_OPERATION_FORBIDDEN,
			errors.Pair("event", eventId),
		)
	}

	if event.Actor != actor {
		return nil, errors.Wrap("event actor missmatch: principal must be event subject",
			errors.ErrorType_UNAUTHORIZED,
			errors.Pair("event", eventId),
			errors.Pair("expected_actor", actor.Id),
		)
	}

	event.Detail.GetAmendmentEvent().HijackFlag = true

	err = s.repo.Set(ctx, event)
	if err != nil {
		return nil, err
	}

	return event, nil
}

func NewRegistrationService(engine coretypes.ACPEngineServer, eventService *EventService, repo CommitmentRepository) *RegistrationService {
	return &RegistrationService{
		engine:       engine,
		eventService: eventService,
		repository:   repo,
	}
}

type RegistrationService struct {
	engine       coretypes.ACPEngineServer
	eventService *EventService
	repository   CommitmentRepository
}

func (s *RegistrationService) UnarchiveObject(ctx sdk.Context, polId string, object *coretypes.Object, actor *coretypes.Actor) (*coretypes.RelationshipRecord, *types.ObjectRegistrationEvent, error) {
	status, err := s.engine.GetObjectRegistration(ctx, &coretypes.GetObjectRegistrationRequest{
		PolicyId: polId,
		Object:   object,
	})
	if err != nil {
		return nil, nil, err
	}
	if !status.IsRegistered {
		return nil, nil, errors.Wrap("object not registered", errors.ErrorType_BAD_INPUT,
			errors.Pair("policy", polId),
			errors.Pair("resource", object.Resource),
			errors.Pair("id", object.Id))
	}
	if !status.Record.Archived {
		return nil, nil, errors.Wrap("object not archived", errors.ErrorType_BAD_INPUT,
			errors.Pair("policy", polId),
			errors.Pair("resource", object.Resource),
			errors.Pair("id", object.Id))
	}

	ev, err := s.eventService.NewEvent(ctx, types.ObjectRegistrationEventType_UNARCHIVAL, polId, object, actor, nil)
	if err != nil {
		return nil, nil, err
	}

	result, err := s.engine.RegisterObject(ctx, &coretypes.RegisterObjectRequest{
		PolicyId:     polId,
		Object:       object,
		CreationTime: ev.Ts.ProtoTs,
	})
	if err != nil {
		return nil, nil, err
	}

	return result.Record, ev, nil
}

func (s *RegistrationService) RegisterObject(ctx sdk.Context, polId string, object *coretypes.Object, actor *coretypes.Actor) (*coretypes.RelationshipRecord, *types.ObjectRegistrationEvent, error) {
	return s.registerWithEvent(ctx, polId, object, actor, types.ObjectRegistrationEventType_REGISTRATION, nil)
}

func (s *RegistrationService) registerWithEvent(ctx sdk.Context, polId string, object *coretypes.Object, actor *coretypes.Actor, eventType types.ObjectRegistrationEventType, detail *types.EventDetail) (*coretypes.RelationshipRecord, *types.ObjectRegistrationEvent, error) {
	status, err := s.engine.GetObjectRegistration(ctx, &coretypes.GetObjectRegistrationRequest{
		PolicyId: polId,
		Object:   object,
	})
	if err != nil {
		return nil, nil, err
	}
	if status.IsRegistered {
		return nil, nil, errors.Wrap("object already registered", errors.ErrorType_OPERATION_FORBIDDEN,
			errors.Pair("policy", polId),
			errors.Pair("resource", object.Resource),
			errors.Pair("id", object.Id))
	}
	if status.Record != nil && status.Record.Archived {
		return nil, nil, errors.Wrap("object archived", errors.ErrorType_OPERATION_FORBIDDEN,
			errors.Pair("policy", polId),
			errors.Pair("resource", object.Resource),
			errors.Pair("id", object.Id))
	}

	ev, err := s.eventService.NewEvent(ctx, eventType, polId, object, actor, detail)
	if err != nil {
		return nil, nil, err
	}

	result, err := s.engine.RegisterObject(ctx, &coretypes.RegisterObjectRequest{
		PolicyId:     polId,
		Object:       object,
		CreationTime: ev.Ts.ProtoTs,
	})
	if err != nil {
		return nil, nil, err
	}

	return result.Record, ev, nil
}

func (s *RegistrationService) amendRegistration(ctx sdk.Context, commitment *types.RegistrationsCommitment, object *coretypes.Object, actor *coretypes.Actor) (*coretypes.RelationshipRecord, *types.ObjectRegistrationEvent, error) {
	// FIXME this name is bad because due to how i called all events are registration events.
	// refactor event to simply ObjectEvent
	event, err := s.eventService.GetLatestRegistrationEvent(ctx, commitment.PolicyId, object)
	if err != nil {
		return nil, nil, err
	}
	// this shouldn't happen because the object was verified
	// to be registered beforehand.
	// return an internal error if it ever does
	if event == nil {
		return nil, nil, errors.Wrap("no registration event for object", errors.ErrorType_INTERNAL,
			errors.Pair("policy", commitment.PolicyId),
			errors.Pair("resource", object.Resource),
			errors.Pair("id", object.Id))
	}

	claimTs := event.Ts
	if event.Type == types.ObjectRegistrationEventType_REVEAL_REGISTRATION {
		claimTs = event.Detail.GetRevealEvent().CommitmentTimestamp
	} else if event.Type == types.ObjectRegistrationEventType_AMENDMENT {
		claimTs = event.Detail.GetAmendmentEvent().CommitmentTimestamp
	}

	if commitment.CreationTs.BlockHeight > claimTs.BlockHeight {
		return nil, nil, errors.Wrap("amendment failed: current registration older than commitment", errors.ErrorType_OPERATION_FORBIDDEN,
			errors.Pair("policy", commitment.PolicyId),
			errors.Pair("resource", object.Resource),
			errors.Pair("object", object.Id),
		)
	}

	registrationRecord, err := s.engine.GetObjectRegistration(ctx, &coretypes.GetObjectRegistrationRequest{
		PolicyId: commitment.PolicyId,
		Object:   object,
	})
	if err != nil {
		return nil, nil, err
	}

	metadata := &types.EventDetail{
		Detail: &types.EventDetail_AmendmentEvent{
			AmendmentEvent: &types.AmendmentEventDetail{
				RevealRegistrationEventId: commitment.Id,
				CommitmentTimestamp:       commitment.CreationTs,
				HijackFlag:                false,
				PreviousOwner: &coretypes.Actor{
					Id: registrationRecord.OwnerId,
				},
			},
		},
	}
	ev, err := s.eventService.NewEvent(ctx, types.ObjectRegistrationEventType_AMENDMENT, commitment.PolicyId, object, actor, metadata)
	if err != nil {
		return nil, nil, err
	}

	goCtx := auth.InjectPrincipal(ctx, auth.RootPrincipal())
	ctx = ctx.WithContext(goCtx)
	result, err := s.engine.AmendRegistration(ctx, &coretypes.AmendRegistrationRequest{
		PolicyId: commitment.PolicyId,
		Object:   object,
		NewOwner: actor,
	})
	if err != nil {
		return nil, nil, err
	}

	return result.Record, ev, nil
}

func (s *RegistrationService) CommitRegistration(ctx sdk.Context, policyId string, commitment []byte, actor *coretypes.Actor, params *types.Params) (*types.RegistrationsCommitment, error) {
	_, err := s.engine.GetPolicy(ctx, &coretypes.GetPolicyRequest{
		Id: policyId,
	})
	if err != nil {
		return nil, err
	}

	if len(commitment) != commitmentLen {
		return nil, newErrInvalidCommitment(policyId, commitment)
	}

	id, err := s.repository.IncrementId(ctx)
	if err != nil {
		return nil, errors.NewFromBaseError(err, errors.ErrorType_INTERNAL, "fail generating commitment id")
	}

	now, err := types.TimestampFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	registration := &types.RegistrationsCommitment{
		Id:         fmt.Sprintf("%v", id),
		PolicyId:   policyId,
		Actor:      actor,
		Commitment: commitment,
		Expired:    false,
		TxHash:     utils.HashTx(ctx.TxBytes()),
		CreationTs: now,
		Validity:   params.RegistrationsCommitmentValidity,
	}

	err = s.repository.Set(ctx, registration)
	if err != nil {
		return nil, err
	}

	return registration, nil
}

// FlagExpiredCommitments iterates over stored commitments,
// filters for expired commitments wrt the current block time,
// flags them as expired and returns the expired commitments
func (s *RegistrationService) FlagExpiredCommitments(ctx sdk.Context) ([]*types.RegistrationsCommitment, error) {
	now, err := types.TimestampFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	commitments, err := s.repository.GetExpiredCommitments(ctx, now)
	if err != nil {
		return nil, err
	}
	processed := make([]*types.RegistrationsCommitment, 0, len(commitments))
	for _, c := range commitments {
		if c.Expired {
			continue
		}
		c.Expired = true
		err := s.repository.Set(ctx, c)
		if err != nil {
			return nil, err
		}
		processed = append(processed, c)
	}
	return commitments, nil
}

func (s *RegistrationService) RevealRegistration(ctx sdk.Context, commitmentId string, proof *types.RegistrationProof, object *coretypes.Object, actor *coretypes.Actor) (*coretypes.RelationshipRecord, *types.ObjectRegistrationEvent, error) {
	commitment, err := s.repository.GetById(ctx, commitmentId)
	if err != nil {
		return nil, nil, err
	}
	if commitment == nil {
		return nil, nil, errors.Wrap("RegistrationsCommimtnet", errors.ErrorType_NOT_FOUND,
			errors.Pair("id", commitmentId))
	}

	now, err := types.TimestampFromCtx(ctx)
	if err != nil {
		return nil, nil, errors.NewFromBaseError(err, errors.ErrorType_INTERNAL, "failed determining current timestamp")
	}
	after, err := types.IsAfter(commitment.CreationTs, commitment.Validity, now)
	if err != nil {
		return nil, nil, errors.NewFromBaseError(err, errors.ErrorType_INTERNAL, "invalid timestmap format")
	}
	if after {
		return nil, nil, errors.Wrap("commitment expired", errors.ErrorType_OPERATION_FORBIDDEN,
			errors.Pair("commitment", commitmentId))
	}

	ok, err := VerifyProof(commitment.Commitment, commitment.PolicyId, commitment.Actor, proof)
	if err != nil {
		return nil, nil, errors.Wrap("invalid registration opening", err)
	} else if !ok {
		return nil, nil, errors.Wrap("invalid registration opening", errors.ErrorType_BAD_INPUT)
	}

	registrationRecord, err := s.engine.GetObjectRegistration(ctx, &coretypes.GetObjectRegistrationRequest{
		PolicyId: commitment.PolicyId,
		Object:   object,
	})
	if err != nil {
		return nil, nil, err
	}

	if !registrationRecord.IsRegistered {
		detail := &types.EventDetail{
			Detail: &types.EventDetail_RevealEvent{
				RevealEvent: &types.RevealRegistrationDetail{
					CommitmentTimestamp:      commitment.CreationTs,
					RegistrationCommitmentId: commitment.Id,
				},
			},
		}
		return s.registerWithEvent(ctx, commitment.PolicyId, object, actor, types.ObjectRegistrationEventType_REVEAL_REGISTRATION, detail)
	}

	return s.amendRegistration(ctx, commitment, object, actor)
}
