package keeper

import (
	"context"
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	sourcehubtypes "github.com/sourcenetwork/sourcehub/types"
	"github.com/sourcenetwork/sourcehub/x/acp/capability"
	"github.com/sourcenetwork/sourcehub/x/acp/did"
	acptypes "github.com/sourcenetwork/sourcehub/x/acp/types"
	"github.com/sourcenetwork/sourcehub/x/bulletin/types"
)

// createActorDID issues a DID based on the specified signer address string.
func createActorDID(ctx context.Context, accountKeeper types.AccountKeeper, signer string) (string, error) {
	addr, err := sourcehubtypes.AccAddressFromBech32(signer)
	if err != nil {
		return "", fmt.Errorf("createActorDID: %v: %w", err, acptypes.NewErrInvalidAccAddrErr(err, signer))
	}

	acc := accountKeeper.GetAccount(ctx, addr)
	if acc == nil {
		return "", fmt.Errorf("createActorDID: %w", acptypes.NewAccNotFoundErr(signer))
	}

	actorDID, err := did.IssueDID(acc)
	if err != nil {
		return "", err
	}

	return actorDID, nil
}

// getNamespaceId adds a prefix to the namespace and returns final namespace id.
func getNamespaceId(namespace string) string {
	if strings.HasPrefix(namespace, types.NamespaceIdPrefix) {
		return namespace // Already prefixed, return as is
	}
	return types.NamespaceIdPrefix + namespace
}

// AddCollaborator adds new namespace collaborator.
func AddCollaborator(ctx context.Context, k *Keeper, namespaceId, collaboratorDID, ownerDID, signer string) error {
	rel := coretypes.NewActorRelationship(types.NamespaceResource, namespaceId, types.CollaboratorRelation, collaboratorDID)
	return addRelationship(ctx, k, rel, namespaceId, ownerDID, signer)
}

// deleteCollaborator deletes existing namespace collaborator.
func deleteCollaborator(ctx context.Context, k *Keeper, namespaceId, collaboratorDID, ownerDID, signer string) error {
	rel := coretypes.NewActorRelationship(types.NamespaceResource, namespaceId, types.CollaboratorRelation, collaboratorDID)
	return deleteRelationship(ctx, k, rel, namespaceId, ownerDID, signer)
}

// addRelationship adds new actor relationship for the specified namespace object.
func addRelationship(goCtx context.Context, k *Keeper, relation *coretypes.Relationship, namespaceId, ownerDID, signer string) error {
	policyId := k.GetPolicyId(goCtx)
	if policyId == "" {
		return types.ErrInvalidPolicyId
	}

	policyCmd := acptypes.NewSetRelationshipCmd(relation)

	manager := capability.NewPolicyCapabilityManager(k.GetScopedKeeper())

	ctx := sdk.UnwrapSDKContext(goCtx)

	polCap, err := manager.Fetch(ctx, policyId)
	if err != nil {
		return err
	}

	_, err = k.GetAcpKeeper().ModulePolicyCmdForActorDID(ctx, polCap, policyCmd, ownerDID, signer)

	return err
}

// deleteRelationship deletes existing actor relationship for the specified namespace object.
func deleteRelationship(goCtx context.Context, k *Keeper, relation *coretypes.Relationship, namespaceId, ownerDID, signer string) error {
	policyId := k.GetPolicyId(goCtx)
	if policyId == "" {
		return types.ErrInvalidPolicyId
	}

	policyCmd := acptypes.NewDeleteRelationshipCmd(relation)

	manager := capability.NewPolicyCapabilityManager(k.GetScopedKeeper())

	ctx := sdk.UnwrapSDKContext(goCtx)

	polCap, err := manager.Fetch(ctx, policyId)
	if err != nil {
		return err
	}

	_, err = k.GetAcpKeeper().ModulePolicyCmdForActorDID(ctx, polCap, policyCmd, ownerDID, signer)

	return err
}

// RegisterNamespace registers a new namespace object under the namespace resource.
func RegisterNamespace(ctx sdk.Context, k *Keeper, namespaceId, ownerDID, signer string) error {
	policyId := k.GetPolicyId(ctx)
	if policyId == "" {
		return types.ErrInvalidPolicyId
	}

	policyCmd := acptypes.NewRegisterObjectCmd(coretypes.NewObject(types.NamespaceResource, namespaceId))

	manager := capability.NewPolicyCapabilityManager(k.GetScopedKeeper())

	polCap, err := manager.Fetch(ctx, policyId)
	if err != nil {
		return err
	}

	_, err = k.GetAcpKeeper().ModulePolicyCmdForActorDID(ctx, polCap, policyCmd, ownerDID, signer)
	return err
}

// hasPermission checks if an actor has required permission for the specified namespace object.
func hasPermission(goCtx context.Context, k *Keeper, namespaceId, permission, actorDID, signer string) (bool, error) {
	policyId := k.GetPolicyId(goCtx)
	if policyId == "" {
		return false, types.ErrInvalidPolicyId
	}

	req := &acptypes.QueryVerifyAccessRequestRequest{
		PolicyId: policyId,
		AccessRequest: &coretypes.AccessRequest{
			Operations: []*coretypes.Operation{
				{
					Object:     coretypes.NewObject(types.NamespaceResource, namespaceId),
					Permission: permission,
				},
			},
			Actor: &coretypes.Actor{
				Id: actorDID,
			},
		},
	}
	result, err := k.GetAcpKeeper().VerifyAccessRequest(goCtx, req)
	if err != nil {
		return false, err
	}

	return result.Valid, nil

}
