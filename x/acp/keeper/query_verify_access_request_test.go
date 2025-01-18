package keeper

import (
	"context"
	"testing"

	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/sourcenetwork/sourcehub/x/acp/did"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

type queryVerifyAccessRequestSuite struct {
	suite.Suite
}

func TestVerifyAccessRequest(t *testing.T) {
	suite.Run(t, &queryVerifyAccessRequestSuite{})
}

func setupTestVerifyAccessRequest(t *testing.T) (context.Context, Keeper, *coretypes.Policy, string) {
	ctx, keeper, accKeep := setupKeeper(t)
	msgServer := NewMsgServerImpl(keeper)

	creatorAcc := accKeep.GenAccount()
	creator := creatorAcc.GetAddress().String()
	creatorDID, _ := did.IssueDID(creatorAcc)

	obj := coretypes.NewObject("file", "1")

	policyStr := `
name: policy
resources:
  file:
    relations: 
      owner:
        types:
          - actor
      rm-root:
    permissions: 
      read: 
        expr: owner
      write: 
        expr: owner
`

	msg := types.MsgCreatePolicy{
		Creator:      creator,
		Policy:       policyStr,
		MarshalType:  coretypes.PolicyMarshalingType_SHORT_YAML,
		CreationTime: timestamp,
	}

	resp, err := msgServer.CreatePolicy(ctx, &msg)
	require.Nil(t, err)

	_, err = msgServer.DirectPolicyCmd(ctx, &types.MsgDirectPolicyCmd{
		Creator:      creator,
		PolicyId:     resp.Policy.Id,
		Cmd:          types.NewRegisterObjectCmd(obj),
		CreationTime: timestamp,
	})
	require.Nil(t, err)

	return ctx, keeper, resp.Policy, creatorDID
}

func (s *queryVerifyAccessRequestSuite) TestVerifyAccessRequest_QueryingObjectsTheActorHasAccessToReturnsTrue() {
	ctx, k, pol, creator := setupTestVerifyAccessRequest(s.T())
	querier := NewQuerier(k)

	req := &types.QueryVerifyAccessRequestRequest{
		PolicyId: pol.Id,
		AccessRequest: &coretypes.AccessRequest{
			Operations: []*coretypes.Operation{
				{
					Object:     coretypes.NewObject("file", "1"),
					Permission: "read",
				},
				{
					Object:     coretypes.NewObject("file", "1"),
					Permission: "write",
				},
			},
			Actor: &coretypes.Actor{
				Id: creator,
			},
		},
	}
	result, err := querier.VerifyAccessRequest(ctx, req)

	want := &types.QueryVerifyAccessRequestResponse{
		Valid: true,
	}
	require.Equal(s.T(), want, result)
	require.Nil(s.T(), err)
}

func (s *queryVerifyAccessRequestSuite) TestVerifyAccessRequest_QueryingOperationActorIsNotAuthorizedReturnNotValid() {
	ctx, k, pol, creator := setupTestVerifyAccessRequest(s.T())
	querier := NewQuerier(k)

	req := &types.QueryVerifyAccessRequestRequest{
		PolicyId: pol.Id,
		AccessRequest: &coretypes.AccessRequest{
			Operations: []*coretypes.Operation{
				{
					Object:     coretypes.NewObject("file", "1"),
					Permission: "rm-root",
				},
			},
			Actor: &coretypes.Actor{
				Id: creator,
			},
		},
	}
	result, err := querier.VerifyAccessRequest(ctx, req)

	want := &types.QueryVerifyAccessRequestResponse{
		Valid: false,
	}
	require.Equal(s.T(), want, result)
	require.Nil(s.T(), err)
}

func (s *queryVerifyAccessRequestSuite) TestVerifyAccessRequest_QueryingObjectThatDoesNotExistReturnValidFalse() {
	ctx, k, pol, creator := setupTestVerifyAccessRequest(s.T())
	querier := NewQuerier(k)

	req := &types.QueryVerifyAccessRequestRequest{
		PolicyId: pol.Id,
		AccessRequest: &coretypes.AccessRequest{
			Operations: []*coretypes.Operation{
				{
					Object:     coretypes.NewObject("file", "file-that-is-not-registered"),
					Permission: "read",
				},
			},
			Actor: &coretypes.Actor{
				Id: creator,
			},
		},
	}
	result, err := querier.VerifyAccessRequest(ctx, req)

	want := &types.QueryVerifyAccessRequestResponse{
		Valid: false,
	}
	require.Equal(s.T(), want, result)
	require.Nil(s.T(), err)
}
