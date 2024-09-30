package registration

import (
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	prototypes "github.com/cosmos/gogoproto/types"
	"github.com/sourcenetwork/acp_core/pkg/errors"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	raccoon "github.com/sourcenetwork/raccoondb"

	"github.com/sourcenetwork/sourcehub/x/acp/metadata"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
	"github.com/sourcenetwork/sourcehub/x/acp/utils"
)

// commitmentLen is a Sha256 Hash, meaning we expect 32 bytes
const commitmentLen int = 256 / 8

type CommitRegistrationsHandler struct{}

func (h *CommitRegistrationsHandler) Handle(
	ctx sdk.Context,
	engine coretypes.ACPEngineServer,
	repository RegistrationsRepository,
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

type RevealRegistrationHandler struct{}

func (h *RevealRegistrationHandler) Handle(
	ctx sdk.Context,
	engine coretypes.ACPEngineServer,
	repository RegistrationsRepository,
	registrationIdCounter *raccoon.CounterStore,
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

	ok := VerifyProof(commitment.Commitment, cmd.Proof)
	if !ok {
		return nil, errors.Wrap("invalid proof", errors.ErrorType_BAD_INPUT)
	}

	registraionRecord, err := engine.GetObjectRegistration(ctx, &coretypes.GetObjectRegistrationRequest{
		PolicyId: commitment.PolicyId,
		Object:   cmd.Object,
	})
	if err != nil {
		return nil, err
	}
	if registraionRecord.IsRegistered {
		return nil, errors.Wrap("object already registered", errors.ErrorType_UNAUTHORIZED,
			errors.Pair("policy", commitment.PolicyId),
			errors.Pair("resource", cmd.Object.Resource),
			errors.Pair("object", cmd.Object.Id),
		)
	}
	// TODO object registration should return object status / archived status

	// TODO verify object is archived by the owner then unarchive.

	// either amendment strategy
	// register strategy

	return nil, nil

}

func (h *RevealRegistrationHandler) registeredStrategy(
	ctx sdk.Context,
	engine coretypes.ACPEngineServer,
	repository RegistrationsRepository,
	registrationIdCounter *raccoon.CounterStore,
	actor *coretypes.Actor,
	commitment *types.RegistrationsCommitment,
	cmd *types.RevealRegistrationCmd) (*types.RevealRegistrationCmdResult, error) {
	// if owned by user
	//    if archived: unarchive, return
	//    if active: return
	// if owned by someone else
	//    if commitment older than registration, return
	//    else return unauthorized
	return nil, nil
}

func (h *RevealRegistrationHandler) amendmentStrategy(
	ctx sdk.Context,
	engine coretypes.ACPEngineServer,
	repository RegistrationsRepository,
	registrationIdCounter *raccoon.CounterStore,
	actor *coretypes.Actor,
	commitment *types.RegistrationsCommitment,
	cmd *types.RevealRegistrationCmd) (*types.RevealRegistrationCmdResult, error) {
	// transfer object ownership
	// create amendment event
	return nil, nil
}

func (h *RevealRegistrationHandler) unregisteredStrategy(
	ctx sdk.Context,
	engine coretypes.ACPEngineServer,
	repository RegistrationsRepository,
	metadataRepository metadata.MetadataRepository,
	registrationIdCounter *raccoon.CounterStore,
	actor *coretypes.Actor,
	commitment *types.RegistrationsCommitment,
	cmd *types.RevealRegistrationCmd) (*types.RevealRegistrationCmdResult, error) {
	record, metadata, err := registerObjectAndMetadata(
		ctx,
		engine,
		repository,
		metadataRepository,
		commitment.PolicyId,
		commitment.Id,
		cmd.Object,
		actor,
	)
	if err != nil {
		return nil, err
	}
	return &types.RevealRegistrationCmdResult{
		Result:         types.RegistrationResultStatus_OK,
		Record:         record,
		Metadata:       metadata,
		AmendmentEvent: nil,
	}, nil
}

// registerObjectAndMetadata is an internal handler which registers an object
// on the ACP Core store and creates and stores registration metadata
func registerObjectAndMetadata(
	ctx sdk.Context,
	engine coretypes.ACPEngineServer,
	repository RegistrationsRepository,
	metadataRepository metadata.MetadataRepository,
	policyId string,
	commitmentId string,
	object *coretypes.Object,
	owner *coretypes.Actor,
) (*coretypes.RelationshipRecord, *types.RegistrationMetadata, error) {
	result, err := engine.RegisterObject(ctx, &coretypes.RegisterObjectRequest{
		PolicyId:     policyId,
		Object:       object,
		CreationTime: nil,
		Metadata:     nil, // TODO change
	})
	if err != nil {
		return nil, nil, err
	}
	ts, err := prototypes.TimestampProto(ctx.BlockTime())
	if err != nil {
		return nil, nil, err
	}

	metadata := &types.RegistrationMetadata{
		TxHash:                   utils.HashTx(ctx.TxBytes()),
		CreationHeight:           uint64(ctx.BlockHeight()),
		RegistrationCommitmentId: commitmentId,
		CreationHeightTs:         ts,
	}

	err = metadataRepository.SetRegistrationMetadata(ctx, policyId, object, metadata)
	if err != nil {
		return nil, nil, err
	}

	return result.Record, metadata, nil
	// TODO this needs to be atomic, are msgs atomic? i can't remmeber, i think so.
}
