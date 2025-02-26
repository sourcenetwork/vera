package commitment

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sourcenetwork/acp_core/pkg/errors"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

func NewCommitmentService(engine coretypes.ACPEngineServer, repository *CommitmentRepository) *CommitmentService {
	return &CommitmentService{
		engine:     engine,
		repository: repository,
	}
}

// CommitmentService abstracts registration commitment operations
type CommitmentService struct {
	engine     coretypes.ACPEngineServer
	repository *CommitmentRepository
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
		err := s.repository.update(ctx, commitment)
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

	if len(commitment) != commitmentBytes {
		return nil, errInvalidCommitment(policyId, commitment)
	}

	metadata, err := types.BuildRecordMetadata(ctx, actor.Id, msgCreator)
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

	err = s.repository.create(ctx, registration)
	if err != nil {
		return nil, err
	}
	return registration, nil
}

// ValidateOpening verifies whether the given opening proof is valid for the authenticated actor and
// the objects
// returns true if opening is valid
func (s *CommitmentService) ValidateOpening(ctx sdk.Context, commitmentId uint64, proof *types.RegistrationProof, actor *coretypes.Actor) (*types.RegistrationsCommitment, bool, error) {
	opt, err := s.repository.GetById(ctx, commitmentId)
	if err != nil {
		return nil, false, err
	}
	if opt.Empty() {
		return nil, false, errors.Wrap("RegistrationsCommimtnet", errors.ErrorType_NOT_FOUND,
			errors.Pair("id", commitmentId))
	}

	commitment := opt.GetValue()
	now, err := types.TimestampFromCtx(ctx)
	if err != nil {
		return commitment, false, errors.NewFromBaseError(err, errors.ErrorType_INTERNAL, "failed determining current timestamp")
	}
	after, err := commitment.Metadata.CreationTs.IsAfter(commitment.Validity, now)
	if err != nil {
		return commitment, false, errors.NewFromBaseError(err, errors.ErrorType_INTERNAL, "invalid timestmap format")
	}
	if after {
		return commitment, false, errors.Wrap("commitment expired", errors.ErrorType_OPERATION_FORBIDDEN,
			errors.Pair("commitment", commitmentId))
	}

	ok, err := VerifyProof(commitment.Commitment, commitment.PolicyId, actor, proof)
	if err != nil {
		return commitment, false, errors.Wrap("invalid registration opening", err)
	}
	return commitment, ok, nil
}
