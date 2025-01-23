package registration

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sourcenetwork/acp_core/pkg/auth"
	"github.com/sourcenetwork/acp_core/pkg/errors"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
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

// RegistrationService abstracts object registration operations
type RegistrationService struct {
	engine            coretypes.ACPEngineServer
	eventService      *EventService
	repository        CommitmentRepository
	commitmentService *CommitmentService
}

// UnarchiveObject flags a given object as active, effectively re-establishing the owner relationship.
// Only the previous owner can unarchive an object.
// This operation is idempotent.
//
// If no change to the state was made, returns a nil ObjectRegistrationEvent.
// If an error happened, returns nil, nil and error
func (s *RegistrationService) UnarchiveObject(ctx sdk.Context, polId string, object *coretypes.Object, actor *coretypes.Actor, msgCreator string) (*coretypes.RelationshipRecord, *types.ObjectRegistrationEvent, error) {
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

	if status.OwnerId != actor.Id {
		return nil, nil, errors.Wrap("unarchiving must be done by previous owner", errors.ErrorType_UNAUTHORIZED,
			errors.Pair("policy", polId),
			errors.Pair("resource", object.Resource),
			errors.Pair("id", object.Id))
	}

	if !status.Record.Archived {
		return status.Record, nil, nil
	}

	ev, err := s.eventService.NewEvent(ctx, types.ObjectRegistrationEventType_UNARCHIVAL, polId, object, actor, nil)
	if err != nil {
		return nil, nil, err
	}

	ctx, err = utils.InjectPrincipal(ctx, actor.Id)
	if err != nil {
		return nil, nil, err
	}
	result, err := s.engine.UnarchiveObject(ctx, &coretypes.UnarchiveObjectRequest{
		PolicyId: polId,
		Object:   object,
	})
	if err != nil {
		return nil, nil, err
	}

	return result.Record, ev, nil
}

func (s *RegistrationService) RegisterObject(ctx sdk.Context, polId string, object *coretypes.Object, actor *coretypes.Actor, msgCreator string) (*coretypes.RelationshipRecord, *types.ObjectRegistrationEvent, error) {
	return s.registerWithEvent(ctx, polId, object, actor, msgCreator, types.ObjectRegistrationEventType_REGISTRATION, nil)
}

func (s *RegistrationService) registerWithEvent(ctx sdk.Context, polId string, object *coretypes.Object, actor *coretypes.Actor, msgCreator string, eventType types.ObjectRegistrationEventType, detail *types.EventDetail) (*coretypes.RelationshipRecord, *types.ObjectRegistrationEvent, error) {
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

	ctx, err = utils.InjectPrincipal(ctx, actor.Id)
	if err != nil {
		return nil, nil, err
	}

	metadata, err := utils.BuildACPSuppliedMetadata(ctx, actor.Id, msgCreator)
	if err != nil {
		return nil, nil, err
	}

	result, err := s.engine.RegisterObject(ctx, &coretypes.RegisterObjectRequest{
		PolicyId: polId,
		Object:   object,
		Metadata: metadata,
	})
	if err != nil {
		return nil, nil, err
	}

	return result.Record, ev, nil
}

func (s *RegistrationService) amendRegistration(ctx sdk.Context, commitment *types.RegistrationsCommitment, record *coretypes.RelationshipRecord, actor *coretypes.Actor, msgCreator string) (*coretypes.RelationshipRecord, *types.ObjectRegistrationEvent, error) {
	object := record.Relationship.Object

	metadata := &types.RecordMetadata{}
	err := metadata.Unmarshal(record.Metadata.Supplied.Blob)
	if err != nil {
		return nil, nil, err
	}

	// registration is older than commitment
	if metadata.CreationTs.BlockHeight < commitment.Metadata.CreationTs.BlockHeight {
		return nil, nil, errors.Wrap("amendment failed: current registration older than commitment", errors.ErrorType_OPERATION_FORBIDDEN,
			errors.Pair("policy", commitment.PolicyId),
			errors.Pair("resource", object.Resource),
			errors.Pair("object", object.Id),
		)
	}

	goCtx := auth.InjectPrincipal(ctx, coretypes.RootPrincipal())
	ctx = ctx.WithContext(goCtx)
	metadata = &types.RecordMetadata{
		CreationTs: commitment.Metadata.CreationTs,
		TxHash:     utils.HashTx(ctx.TxBytes()),
		OwnerDid:   actor.Id,
		TxSigner:   msgCreator,
	}
	blob, err := metadata.Marshal()
	if err != nil {
		return nil, nil, err
	}
	result, err := s.engine.AmendRegistration(ctx, &coretypes.AmendRegistrationRequest{
		PolicyId:      commitment.PolicyId,
		Object:        object,
		NewOwner:      actor,
		NewCreationTs: commitment.Metadata.CreationTs.ProtoTs,
		Metadata: &coretypes.SuppliedMetadata{
			Blob: blob,
		},
	})
	if err != nil {
		return nil, nil, err
	}

	event := &types.EventDetail{
		Detail: &types.EventDetail_AmendmentEvent{
			AmendmentEvent: &types.AmendmentEventDetail{
				RevealRegistrationEventId: commitment.Id,
				CommitmentTimestamp:       commitment.Metadata.CreationTs,
				HijackFlag:                false,
				PreviousOwner: &coretypes.Actor{
					Id: record.Metadata.Creator.Identifier,
				},
			},
		},
	}
	ev, err := s.eventService.NewEvent(ctx, types.ObjectRegistrationEventType_AMENDMENT, commitment.PolicyId, object, actor, event)
	if err != nil {
		return nil, nil, err
	}

	return result.Record, ev, nil
}

// RevealRegistation attempts to register an object from a commitment opening.
// If the opening is valid, registers the object.
//
// In the event where the opening is valid and the object was already registered,
// if the commitment is older than the registration, run the amendment protocol
// which transfers the object's ownership to the commitment author.
func (s *RegistrationService) RevealRegistration(ctx sdk.Context, commitmentId uint64, proof *types.RegistrationProof, actor *coretypes.Actor, msgSigner string) (*coretypes.RelationshipRecord, *types.ObjectRegistrationEvent, error) {
	ok, err := s.commitmentService.ValidateOpening(ctx, commitmentId, proof, actor)
	if err != nil {
		return nil, nil, err
	}
	if !ok {
		return nil, nil, errors.Wrap("invalid registration opening", errors.ErrorType_UNAUTHORIZED)
	}

	opt, err := s.repository.GetById(ctx, commitmentId)
	if err != nil {
		return nil, nil, err
	}
	if opt.Empty() {
		return nil, nil, errors.Wrap("RegistrationsCommimtnet", errors.ErrorType_NOT_FOUND,
			errors.Pair("id", commitmentId))
	}
	commitment := opt.GetValue()

	registrationRecord, err := s.engine.GetObjectRegistration(ctx, &coretypes.GetObjectRegistrationRequest{
		PolicyId: opt.GetValue().PolicyId,
		Object:   proof.Object,
	})
	if err != nil {
		return nil, nil, err
	}

	if !registrationRecord.IsRegistered {
		detail := &types.EventDetail{
			Detail: &types.EventDetail_RevealEvent{
				RevealEvent: &types.RevealRegistrationDetail{
					CommitmentTimestamp:      commitment.Metadata.CreationTs,
					RegistrationCommitmentId: commitment.Id,
				},
			},
		}
		// FIXME this is a problem, i need to fiddle with the current timestamp
		// which means i'll have to fiddle with the engine and the time service
		panic("fixme")
		return s.registerWithEvent(ctx, commitment.PolicyId, proof.Object, actor, msgSigner, types.ObjectRegistrationEventType_REVEAL_REGISTRATION, detail)
	}

	return s.amendRegistration(ctx, commitment, registrationRecord.Record, actor, msgSigner)
}

// CommitmentService abstracts registration commitment operations
type CommitmentService struct {
	engine       coretypes.ACPEngineServer
	eventService *EventService
	repository   CommitmentRepository
}

// BuildCommitment produces a byte commitment for actor and objects.
// The commitment is guaranteed to be valid, as we verify that no object has been registered yet.
func (s *CommitmentService) BuildCommitment(ctx sdk.Context, policyId string, actor *coretypes.Actor, objects []*coretypes.Object) ([]byte, error) {
	rec, err := s.engine.GetPolicy(ctx, &coretypes.GetPolicyRequest{
		Id: policyId,
	})
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, errors.ErrPolicyNotFound(policyId)
	}

	for _, obj := range objects {
		status, err := s.engine.GetObjectRegistration(ctx, &coretypes.GetObjectRegistrationRequest{
			PolicyId: policyId,
			Object:   obj,
		})
		if err != nil {
			return nil, err
		}
		if status.IsRegistered {
			return nil, errors.Wrap("object already registered", errors.ErrorType_BAD_INPUT,
				errors.Pair("policy", policyId),
				errors.Pair("resource", obj.Resource),
				errors.Pair("object", obj.Id),
			)
		}
	}

	return GenerateCommitmentWithoutValidation(policyId, actor, objects)
}

// FlagExpiredCommitments iterates over stored commitments,
// filters for expired commitments wrt the current block time,
// flags them as expired and returns the newly expired commitments
func (s *CommitmentService) FlagExpiredCommitments(ctx sdk.Context) ([]*types.RegistrationsCommitment, error) {
	now, err := types.TimestampFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	iter, err := s.repository.GetNonExpiredCommitments(ctx)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	var processed []*types.RegistrationsCommitment
	for !iter.Finished() {
		commitment, err := iter.Value()
		if err != nil {
			return nil, err
		}
		expired, err := commitment.IsExpiredAgainst(now)
		if err != nil {
			return nil, err
		}
		if expired {
			commitment.Expired = true
			processed = append(processed, commitment)
		}

		err = iter.Next(ctx)
		if err != nil {
			return nil, err
		}
	}

	for _, commitment := range processed {
		err := s.repository.Set(ctx, commitment)
		if err != nil {
			return nil, errors.Wrap("expiring commitment", err, errors.Pair("commitment", commitment.Id))
		}
	}

	return processed, nil
}

// SetNewCommitment sets a new RegistrationCommitment
func (s *CommitmentService) SetNewCommitment(ctx sdk.Context, policyId string, commitment []byte, actor *coretypes.Actor, params *types.Params, msgCreator string) (*types.RegistrationsCommitment, error) {
	rec, err := s.engine.GetPolicy(ctx, &coretypes.GetPolicyRequest{
		Id: policyId,
	})
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, errors.ErrPolicyNotFound(policyId)
	}

	if len(commitment) != commitmentLen {
		return nil, errInvalidCommitment(policyId, commitment)
	}

	metadata, err := utils.BuildRecordMetadata(ctx, actor.Id, msgCreator)
	if err != nil {
		return nil, err
	}

	registration := &types.RegistrationsCommitment{
		Id:         0, // doesn't matter since it will be auto-generated
		PolicyId:   policyId,
		Commitment: commitment,
		Expired:    false,
		Validity:   params.RegistrationsCommitmentValidity,
		Metadata:   metadata,
	}

	err = s.repository.Create(ctx, registration)
	if err != nil {
		return nil, err
	}
	return registration, nil
}

// ValidateOpening verifies whether the given opening proof is valid for the authenticated actor and
// the objects
// returns true if opening is valid
func (s *CommitmentService) ValidateOpening(ctx sdk.Context, commitmentId uint64, proof *types.RegistrationProof, actor *coretypes.Actor) (bool, error) {
	opt, err := s.repository.GetById(ctx, commitmentId)
	if err != nil {
		return false, err
	}
	if opt.Empty() {
		return false, errors.Wrap("RegistrationsCommimtnet", errors.ErrorType_NOT_FOUND,
			errors.Pair("id", commitmentId))
	}

	commitment := opt.GetValue()
	now, err := types.TimestampFromCtx(ctx)
	if err != nil {
		return false, errors.NewFromBaseError(err, errors.ErrorType_INTERNAL, "failed determining current timestamp")
	}
	after, err := types.IsAfter(commitment.Metadata.CreationTs, commitment.Validity, now)
	if err != nil {
		return false, errors.NewFromBaseError(err, errors.ErrorType_INTERNAL, "invalid timestmap format")
	}
	if after {
		return false, errors.Wrap("commitment expired", errors.ErrorType_OPERATION_FORBIDDEN,
			errors.Pair("commitment", commitmentId))
	}

	ok, err := VerifyProof(commitment.Commitment, commitment.PolicyId, actor, proof)
	if err != nil {
		return false, errors.Wrap("invalid registration opening", err)
	}
	return ok, nil
}
