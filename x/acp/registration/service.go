package registration

import (
	"fmt"
	"slices"

	prototypes "github.com/cosmos/gogoproto/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sourcenetwork/acp_core/pkg/errors"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	acputils "github.com/sourcenetwork/acp_core/pkg/utils"
	raccoon "github.com/sourcenetwork/raccoondb"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
	"github.com/sourcenetwork/sourcehub/x/acp/utils"
)

type eventService struct {
	counter *raccoon.CounterStore
	repo    RegistrationEventRepository
}

func (s *eventService) NewEvent(ctx sdk.Context, t types.ObjectRegistrationEventType, polId string, object *coretypes.Object, actor *coretypes.Actor, detail *types.EventDetail) (*types.ObjectRegistrationEvent, error) {
	id, err := s.counter.GetNextAndIncrement(ctx)
	if err != nil {
		return nil, err
	}

	ts, err := prototypes.TimestampProto(ctx.BlockTime())
	if err != nil {
		return nil, err
	}

	event := &types.ObjectRegistrationEvent{
		Id:       fmt.Sprintf("%v", id),
		Type:     t,
		TxHash:   utils.HashTx(ctx.TxBytes()),
		Height:   uint64(ctx.BlockHeight()),
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

func (s *eventService) GetLatestRegistrationEvent(ctx sdk.Context, polId string, object *coretypes.Object) (*types.ObjectRegistrationEvent, error) {
	evs, err := s.repo.GetObjectEvents(ctx, polId, *object)
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
		return ev.Height
	})
	sortable.SortInPlace()

	return evs[0], nil
}

func (s *eventService) FlagHijackEvent(ctx sdk.Context, eventId string, actor *coretypes.Actor) (*types.ObjectRegistrationEvent, error) {
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
			errors.ErrorType_OPERATION_FORBIDDEN,
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

type registrationService struct {
	engine       coretypes.ACPEngineServer
	eventService *eventService
	repository   CommitmentRepository
}

func (s *registrationService) UnarchiveObject(ctx sdk.Context, polId string, object *coretypes.Object, actor *coretypes.Actor) (*coretypes.RelationshipRecord, *types.ObjectRegistrationEvent, error) {
	status, err := s.engine.GetObjectRegistration(ctx, &coretypes.GetObjectRegistrationRequest{
		PolicyId: polId,
		Object:   object,
	})
	if err != nil {
		return nil, nil, err
	}
	if !status.IsRegistered {
		return nil, nil, nil //TODO return protocol exception
	}
	if !status.Record.Archived {
		return nil, nil, nil //TODO return protocol exception
	}

	ev, err := s.eventService.NewEvent(ctx, types.ObjectRegistrationEventType_UNARCHIVAL, polId, object, actor, nil)
	if err != nil {
		return nil, nil, err
	}

	result, err := s.engine.RegisterObject(ctx, &coretypes.RegisterObjectRequest{
		PolicyId:     polId,
		Object:       object,
		CreationTime: ev.Ts,
	})
	if err != nil {
		return nil, nil, err
	}

	return result.Record, ev, nil
}

func (s *registrationService) RegisterObject(ctx sdk.Context, polId string, object *coretypes.Object, actor *coretypes.Actor) (*coretypes.RelationshipRecord, *types.ObjectRegistrationEvent, error) {
	status, err := s.engine.GetObjectRegistration(ctx, &coretypes.GetObjectRegistrationRequest{
		PolicyId: polId,
		Object:   object,
	})
	if err != nil {
		return nil, nil, err
	}
	if status.IsRegistered {
		return nil, nil, nil //TODO return protocol exception
	}
	if status.Record.Archived {
		return nil, nil, nil //TODO return protocol exception
	}

	ev, err := s.eventService.NewEvent(ctx, types.ObjectRegistrationEventType_REGISTRATION, polId, object, actor, nil)
	if err != nil {
		return nil, nil, err
	}

	result, err := s.engine.RegisterObject(ctx, &coretypes.RegisterObjectRequest{
		PolicyId:     polId,
		Object:       object,
		CreationTime: ev.Ts,
	})
	if err != nil {
		return nil, nil, err
	}

	return result.Record, ev, nil
}

func (s *registrationService) AmendRegistration(ctx sdk.Context, commitmentId string, polId string, object *coretypes.Object, actor *coretypes.Actor) (*coretypes.RelationshipRecord, *types.ObjectRegistrationEvent, error) {
	status, err := s.engine.GetObjectRegistration(ctx, &coretypes.GetObjectRegistrationRequest{
		PolicyId: polId,
		Object:   object,
	})
	if err != nil {
		return nil, nil, err
	}
	if !status.IsRegistered {
		return nil, nil, nil //TODO return protocol exception
	}
	if status.OwnerId == actor.Id {
		return nil, nil, nil // TODO wrap
	}

	commitment, err := s.repository.GetById(ctx, commitmentId)
	if err != nil {
		return nil, nil, err
	}

	metadata := &types.EventDetail{
		Detail: &types.EventDetail_AmendmentEvent{
			AmendmentEvent: &types.AmendmentEventDetail{
				RevealRegistrationEventId: commitment.Id,
				CommitmentCreationHeight:  commitment.CreationHeight,
				HijackFlag:                false,
				PreviousOwner: &coretypes.Actor{
					Id: status.OwnerId,
				},
			},
		},
	}
	ev, err := s.eventService.NewEvent(ctx, types.ObjectRegistrationEventType_AMENDMENT, polId, object, actor, metadata)
	if err != nil {
		return nil, nil, err
	}

	result, err := s.engine.TransferObject(ctx, &coretypes.TransferObjectRequest{
		PolicyId: polId,
		Object:   object,
		NewOwner: actor,
	})
	if err != nil {
		return nil, nil, err
	}

	return result.Record, ev, nil
}
