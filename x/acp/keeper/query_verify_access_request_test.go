package keeper

import (
	"context"
	"testing"

	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/stretchr/testify/require"

	"github.com/sourcenetwork/sourcehub/x/acp/did"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

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
		Creator:     creator,
		Policy:      policyStr,
		MarshalType: coretypes.PolicyMarshalingType_SHORT_YAML,
	}

	resp, err := msgServer.CreatePolicy(ctx, &msg)
	require.Nil(t, err)

	_, err = msgServer.DirectPolicyCmd(ctx, &types.MsgDirectPolicyCmd{
		Creator:  creator,
		PolicyId: resp.Record.Policy.Id,
		Cmd:      types.NewRegisterObjectCmd(obj),
	})
	require.Nil(t, err)

	return ctx, keeper, resp.Record.Policy, creatorDID
}

func TestVerifyAccessRequest_QueryingObjectsTheActorHasAccessToReturnsTrue(t *testing.T) {
	ctx, keeper, pol, creator := setupTestVerifyAccessRequest(t)

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
	result, err := keeper.VerifyAccessRequest(ctx, req)

	want := &types.QueryVerifyAccessRequestResponse{
		Valid: true,
	}
	require.Equal(t, want, result)
	require.Nil(t, err)
}

func TestVerifyAccessRequest_QueryingOperationActorIsNotAuthorizedReturnNotValid(t *testing.T) {
	ctx, keeper, pol, creator := setupTestVerifyAccessRequest(t)

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
	result, err := keeper.VerifyAccessRequest(ctx, req)

	want := &types.QueryVerifyAccessRequestResponse{
		Valid: false,
	}
	require.Equal(t, want, result)
	require.Nil(t, err)
}

func TestVerifyAccessRequest_QueryingObjectThatDoesNotExistReturnValidFalse(t *testing.T) {
	ctx, keeper, pol, creator := setupTestVerifyAccessRequest(t)

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
	result, err := keeper.VerifyAccessRequest(ctx, req)

	want := &types.QueryVerifyAccessRequestResponse{
		Valid: false,
	}
	require.Equal(t, want, result)
	require.Nil(t, err)
}
