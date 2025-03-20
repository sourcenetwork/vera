package keeper

import (
	"context"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sourcenetwork/acp_core/pkg/errors"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

type queryObjectOwnerSuite struct {
	suite.Suite

	obj *coretypes.Object
}

func TestObjectOwner(t *testing.T) {
	suite.Run(t, &queryObjectOwnerSuite{})
}

func (s *queryObjectOwnerSuite) setup(t *testing.T) (context.Context, Keeper, sdk.AccountI, string, string) {
	s.obj = coretypes.NewObject("file", "1")

	policyStr := `
name: policy
description: ok
resources:
  file:
    relations: 
      owner:
        doc: owner owns
        types:
          - actor-resource
      reader:
      admin:
        manages:
          - reader
    permissions: 
      own:
        expr: owner
        doc: own doc
      read: 
        expr: owner + reader
actor:
  name: actor-resource
  doc: my actor
          `

	ctx, keeper, accKeep := setupKeeper(t)
	creator := accKeep.FirstAcc().GetAddress().String()

	msg := types.MsgCreatePolicy{
		Creator:     creator,
		Policy:      policyStr,
		MarshalType: coretypes.PolicyMarshalingType_SHORT_YAML,
	}

	msgServer := NewMsgServerImpl(keeper)

	resp, err := msgServer.CreatePolicy(ctx, &msg)
	require.Nil(t, err)

	_, err = msgServer.DirectPolicyCmd(ctx, &types.MsgDirectPolicyCmd{
		Creator:  creator,
		PolicyId: resp.Record.Policy.Id,
		Cmd:      types.NewRegisterObjectCmd(s.obj),
	})
	require.Nil(t, err)

	return ctx, keeper, accKeep.FirstAcc(), creator, resp.Record.Policy.Id
}

func (s *queryObjectOwnerSuite) TestQueryReturnsObjectOwner() {
	ctx, k, _, _, policyId := s.setup(s.T())

	resp, err := k.ObjectOwner(ctx, &types.QueryObjectOwnerRequest{
		PolicyId: policyId,
		Object:   s.obj,
	})

	require.Equal(s.T(), resp, &types.QueryObjectOwnerResponse{
		IsRegistered: true,
		Record:       resp.Record,
	})
	require.Nil(s.T(), err)
}

func (s *queryObjectOwnerSuite) TestQueryingForUnregisteredObjectReturnsEmptyOwner() {
	ctx, k, _, _, policyId := s.setup(s.T())

	resp, err := k.ObjectOwner(ctx, &types.QueryObjectOwnerRequest{
		PolicyId: policyId,
		Object:   coretypes.NewObject("file", "404"),
	})

	require.Nil(s.T(), err)
	require.Equal(s.T(), resp, &types.QueryObjectOwnerResponse{
		IsRegistered: false,
		Record:       nil,
	})
}

func (s *queryObjectOwnerSuite) TestQueryingPolicyThatDoesNotExistReturnError() {
	ctx, k, _, _, _ := s.setup(s.T())

	resp, err := k.ObjectOwner(ctx, &types.QueryObjectOwnerRequest{
		PolicyId: "some-policy",
		Object:   s.obj,
	})

	require.ErrorIs(s.T(), err, errors.ErrorType_NOT_FOUND)
	require.Nil(s.T(), resp)
}

func (s *queryObjectOwnerSuite) TestQueryingForObjectInNonExistingPolicyReturnsError() {
	ctx, k, _, _, policyId := s.setup(s.T())

	resp, err := k.ObjectOwner(ctx, &types.QueryObjectOwnerRequest{
		PolicyId: policyId,
		Object:   coretypes.NewObject("missing-resource", "abc"),
	})

	require.Nil(s.T(), resp)
	require.NotNil(s.T(), err)
}
