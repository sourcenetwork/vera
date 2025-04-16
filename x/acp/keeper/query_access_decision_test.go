package keeper

import (
	"context"
	"testing"

	prototypes "github.com/cosmos/gogoproto/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

type queryAccessDecisionSuite struct {
	suite.Suite

	testDecision *types.AccessDecision
}

func TestAccessDecision(t *testing.T) {
	suite.Run(t, &queryAccessDecisionSuite{})
}

func (s *queryAccessDecisionSuite) setup(t *testing.T) (context.Context, Keeper) {
	ctx, k, _ := setupKeeper(t)

	decision := &types.AccessDecision{
		Id:                 "decision-1",
		PolicyId:           "policy-1",
		Creator:            "creator-1",
		CreatorAccSequence: 12345,
		Operations: []*coretypes.Operation{
			{
				Object: &coretypes.Object{
					Resource: "file",
					Id:       "file-1",
				},
				Permission: "read",
			},
		},
		Actor: "collaborator",
		Params: &types.DecisionParams{
			DecisionExpirationDelta: 3600,
			ProofExpirationDelta:    7200,
			TicketExpirationDelta:   86400,
		},
		CreationTime: &types.Timestamp{
			ProtoTs:     &prototypes.Timestamp{},
			BlockHeight: 0,
		},
		IssuedHeight: 100,
	}

	repo := k.getAccessDecisionRepository(ctx)
	err := repo.Set(ctx, decision)
	require.NoError(t, err)

	s.testDecision = decision
	return ctx, k
}

func (s *queryAccessDecisionSuite) TestQueryAccessDecision_ValidRequest() {
	ctx, k := s.setup(s.T())

	resp, err := k.AccessDecision(ctx, &types.QueryAccessDecisionRequest{
		Id: s.testDecision.Id,
	})

	require.NoError(s.T(), err)
	require.NotNil(s.T(), resp)
	require.Equal(s.T(), s.testDecision, resp.Decision)
}

func (s *queryAccessDecisionSuite) TestQueryAccessDecision_InvalidRequest() {
	ctx, k := s.setup(s.T())

	resp, err := k.AccessDecision(ctx, nil)

	require.Error(s.T(), err)
	require.Nil(s.T(), resp)
	require.Equal(s.T(), codes.InvalidArgument, status.Code(err))
}

func (s *queryAccessDecisionSuite) TestQueryAccessDecision_InvalidId() {
	ctx, k := s.setup(s.T())

	resp, err := k.AccessDecision(ctx, &types.QueryAccessDecisionRequest{
		Id: "",
	})

	require.NoError(s.T(), err)
	require.Equal(s.T(), &types.QueryAccessDecisionResponse{}, resp)
	require.Equal(s.T(), codes.OK, status.Code(err))
}
