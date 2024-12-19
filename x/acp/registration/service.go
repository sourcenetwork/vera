package registration

import (
	"slices"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sourcenetwork/acp_core/pkg/auth"
	"github.com/sourcenetwork/acp_core/pkg/errors"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	acputils "github.com/sourcenetwork/acp_core/pkg/utils"
	"github.com/sourcenetwork/raccoondb/v2/iterator"
	rctypes "github.com/sourcenetwork/raccoondb/v2/types"
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
	ts, err := types.TimestampFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	event := &types.ObjectRegistrationEvent{
		Id:       0,
		Type:     t,
		TxHash:   utils.HashTx(ctx.TxBytes()),
		Ts:       ts,
		PolicyId: polId,
		Object:   object,
		Actor:    actor,
		Detail:   detail,
	}

	err = s.repo.Create(ctx, event)
	if err != nil {
		return nil, err
	}

	return event, nil
}

func (s *EventService) GetLatestRegistrationEvent(ctx sdk.Context, polId string, object *coretypes.Object) (rctypes.Option[*types.ObjectRegistrationEvent], error) {
	iter, err := s.repo.GetObjectEvents(ctx, polId, object)
	if err != nil {
		return rctypes.None[*types.ObjectRegistrationEvent](), err
	}
	iter = iterator.Filter(iter, func(ev *types.ObjectRegistrationEvent) bool {
		return slices.Contains(types.ObjectClaimEvents, ev.Type)
	})

	evs, err := iterator.Consume(ctx, iter)
	if err != nil {
		return rctypes.None[*types.ObjectRegistrationEvent](), err
	}
	if len(evs) == 0 {
		return rctypes.None[*types.ObjectRegistrationEvent](), nil
	}

	// ACP should maintain an invariant where, after the first claim event,
	// any further object amendment can only reference an ealier commitment.
	// this means that the lastest event is the one that corresponds to the earliest registration"
	sortable := acputils.FromExtractor(evs, func(ev *types.ObjectRegistrationEvent) uint64 {
		return ev.Ts.BlockHeight
	})
	sortable.SortInPlace()

	return rctypes.Some(evs[0]), nil
}

func (s *EventService) FlagHijackEvent(ctx sdk.Context, eventId uint64, actor *coretypes.Actor) (*types.ObjectRegistrationEvent, error) {
	opt, err := s.repo.GetById(ctx, eventId)
	if err != nil {
		return nil, err
	}
	if opt.Empty() {
		return nil, errors.Wrap("event not found", errors.ErrorType_NOT_FOUND, errors.Pair("event", eventId))
	}

	event := opt.GetValue()
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
		PolicyId: polId,
		Object:   object,
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
		PolicyId: polId,
		Object:   object,
	})
	if err != nil {
		return nil, nil, err
	}

	return result.Record, ev, nil
}

func (s *RegistrationService) amendRegistration(ctx sdk.Context, commitment *types.RegistrationsCommitment, object *coretypes.Object, actor *coretypes.Actor) (*coretypes.RelationshipRecord, *types.ObjectRegistrationEvent, error) {
	// FIXME this name is bad because due to how i called all events are registration events.
	// refactor event to simply ObjectEvent
	opt, err := s.eventService.GetLatestRegistrationEvent(ctx, commitment.PolicyId, object)
	if err != nil {
		return nil, nil, err
	}
	// this shouldn't happen because the object was verified
	// to be registered beforehand.
	// return an internal error if it ever does
	if opt.Empty() {
		return nil, nil, errors.Wrap("no registration event for object", errors.ErrorType_INTERNAL,
			errors.Pair("policy", commitment.PolicyId),
			errors.Pair("resource", object.Resource),
			errors.Pair("id", object.Id))
	}

	event := opt.GetValue()
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

	goCtx := auth.InjectPrincipal(ctx, coretypes.RootPrincipal())
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
	rec, err := s.engine.GetPolicy(ctx, &coretypes.GetPolicyRequest{
		Id: policyId,
	})
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, errors.NewPolicyNotFound(policyId)
	}

	if len(commitment) != commitmentLen {
		return nil, newErrInvalidCommitment(policyId, commitment)
	}

	now, err := types.TimestampFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	registration := &types.RegistrationsCommitment{
		Id:         0,
		PolicyId:   policyId,
		Actor:      actor,
		Commitment: commitment,
		Expired:    false,
		TxHash:     utils.HashTx(ctx.TxBytes()),
		CreationTs: now,
		Validity:   params.RegistrationsCommitmentValidity,
	}

	err = s.repository.Create(ctx, registration)
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
	iter, err := s.repository.GetExpiredCommitments(ctx, now)
	if err != nil {
		return nil, err
	}
	commitments, err := iterator.Consume(ctx, iter)
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
			return nil, errors.Wrap("expiring commitment", err, errors.Pair("commitment", c.Id))
		}
		processed = append(processed, c)
	}
	return commitments, nil
}

func (s *RegistrationService) RevealRegistration(ctx sdk.Context, commitmentId uint64, proof *types.RegistrationProof, actor *coretypes.Actor) (*coretypes.RelationshipRecord, *types.ObjectRegistrationEvent, error) {
	opt, err := s.repository.GetById(ctx, commitmentId)
	if err != nil {
		return nil, nil, err
	}
	if opt.Empty() {
		return nil, nil, errors.Wrap("RegistrationsCommimtnet", errors.ErrorType_NOT_FOUND,
			errors.Pair("id", commitmentId))
	}

	commitment := opt.GetValue()
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
		Object:   proof.Object,
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
		return s.registerWithEvent(ctx, commitment.PolicyId, proof.Object, actor, types.ObjectRegistrationEventType_REVEAL_REGISTRATION, detail)
	}

	return s.amendRegistration(ctx, commitment, proof.Object, actor)
}

func (s *RegistrationService) GenerateCommitment(ctx sdk.Context, policyId string, actor *coretypes.Actor, objects []*coretypes.Object) ([]byte, error) {
	rec, err := s.engine.GetPolicy(ctx, &coretypes.GetPolicyRequest{
		Id: policyId,
	})
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, errors.NewPolicyNotFound(policyId)
	}

	for _, obj := range objects {
		resource := rec.Record.Policy.GetResourceByName(obj.Resource)
		if resource == nil {
			return nil, errors.Wrap("resource not found", errors.ErrorType_BAD_INPUT,
				errors.Pair("policy", policyId),
				errors.Pair("resource", obj.Resource),
			)
		}
	}

	return GenerateCommitmentWithoutValidation(policyId, actor, objects)
}
