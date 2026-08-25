package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"

	acptypes "github.com/sourcenetwork/vera/x/acp/types"
	"github.com/sourcenetwork/vera/x/orbis/types"
)

func (k *Keeper) registerRingACPObject(goCtx context.Context, creator string, policyID string, ringID string) error {
	_, err := k.GetAcpKeeper().DirectPolicyCmd(goCtx, &acptypes.MsgDirectPolicyCmd{
		Creator:  creator,
		PolicyId: policyID,
		Cmd:      acptypes.NewRegisterObjectCmd(coretypes.NewObject(types.ACPResourceRing, ringID)),
	})
	return err
}

func (k *Keeper) ensureRingCreatePermission(goCtx context.Context, policyID string, creatorDID string) error {
	if policyID == "" {
		return errorsmod.Wrap(types.ErrInvalidRing, "missing policy_id")
	}

	resp, err := k.GetAcpKeeper().VerifyAccessRequest(goCtx, &acptypes.QueryVerifyAccessRequestRequest{
		PolicyId: policyID,
		AccessRequest: &coretypes.AccessRequest{
			Operations: []*coretypes.Operation{
				{
					Object:     coretypes.NewObject(types.ACPResourceRingPolicy, policyID),
					Permission: types.ACPPermissionCreateRing,
				},
			},
			Actor: coretypes.NewActor(creatorDID),
		},
	})
	if err != nil {
		return err
	}
	if !resp.Valid {
		return errorsmod.Wrapf(types.ErrUnauthorizedRingCreate, "actor %s cannot create rings under policy %s", creatorDID, policyID)
	}
	return nil
}

func (k *Keeper) ensureRingUpdatePermission(goCtx context.Context, ring *types.Ring, updaterDID string) error {
	if ring.PolicyId == "" {
		return errorsmod.Wrap(types.ErrInvalidRing, "missing policy_id")
	}

	resp, err := k.GetAcpKeeper().VerifyAccessRequest(goCtx, &acptypes.QueryVerifyAccessRequestRequest{
		PolicyId: ring.PolicyId,
		AccessRequest: &coretypes.AccessRequest{
			Operations: []*coretypes.Operation{
				{
					Object:     coretypes.NewObject(types.ACPResourceRing, ring.Id),
					Permission: types.ACPPermissionUpdateRing,
				},
			},
			Actor: coretypes.NewActor(updaterDID),
		},
	})
	if err != nil {
		return err
	}
	if !resp.Valid {
		return errorsmod.Wrapf(types.ErrUnauthorizedRingUpdate, "actor %s cannot update ring %s", updaterDID, ring.Id)
	}
	return nil
}
