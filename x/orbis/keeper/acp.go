package keeper

import (
	"context"

	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	acptypes "github.com/sourcenetwork/sourcehub/x/acp/types"
	"github.com/sourcenetwork/sourcehub/x/orbis/types"
)

func hasCreateRingPermission(ctx context.Context, k *Keeper, policyId, namespace, actorDID string) (bool, error) {
	req := &acptypes.QueryVerifyAccessRequestRequest{
		PolicyId: policyId,
		AccessRequest: &coretypes.AccessRequest{
			Operations: []*coretypes.Operation{
				{
					Object:     coretypes.NewObject(types.NamespaceResource, namespace),
					Permission: types.CreateRingPermission,
				},
			},
			Actor: &coretypes.Actor{
				Id: actorDID,
			},
		},
	}
	result, err := k.GetAcpKeeper().VerifyAccessRequest(ctx, req)
	if err != nil {
		return false, err
	}
	return result.Valid, nil
}

func hasRingUpdatePermission(ctx context.Context, k *Keeper, ring *types.Ring, actorDID string) (bool, error) {
	req := &acptypes.QueryVerifyAccessRequestRequest{
		PolicyId: ring.PolicyId,
		AccessRequest: &coretypes.AccessRequest{
			Operations: []*coretypes.Operation{
				{
					Object:     coretypes.NewObject(types.NamespaceResource, ring.Namespace),
					Permission: types.UpdateRingPermission,
				},
			},
			Actor: &coretypes.Actor{
				Id: actorDID,
			},
		},
	}
	result, err := k.GetAcpKeeper().VerifyAccessRequest(ctx, req)
	if err != nil {
		return false, err
	}

	return result.Valid, nil
}
