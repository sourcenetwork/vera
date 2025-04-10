package keeper

import (
	"context"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/sourcenetwork/sourcehub/x/acp/capability"
	acptypes "github.com/sourcenetwork/sourcehub/x/acp/types"
	"github.com/sourcenetwork/sourcehub/x/bulletin/types"
)

// getNamespaceId adds a prefix to the namespace and returns final namespace id.
func getNamespaceId(namespace string) string {
	if strings.HasPrefix(namespace, types.NamespaceIdPrefix) {
		return namespace // Already prefixed, return as is
	}
	return types.NamespaceIdPrefix + namespace
}

// AddCollaborator adds new namespace collaborator.
func AddCollaborator(ctx context.Context, k *Keeper, policyId, namespaceId, collaboratorDID, ownerDID, signer string) error {
	if k.getCollaborator(ctx, namespaceId, collaboratorDID) != nil {
		return types.ErrCollaboratorAlreadyExists
	}
	rel := coretypes.NewActorRelationship(types.NamespaceResource, namespaceId, types.CollaboratorRelation, collaboratorDID)
	return addRelationship(ctx, k, rel, policyId, namespaceId, ownerDID, signer)
}

// deleteCollaborator deletes existing namespace collaborator.
func deleteCollaborator(ctx context.Context, k *Keeper, policyId, namespaceId, collaboratorDID, ownerDID, signer string) error {
	if k.getCollaborator(ctx, namespaceId, collaboratorDID) == nil {
		return types.ErrCollaboratorNotFound
	}
	rel := coretypes.NewActorRelationship(types.NamespaceResource, namespaceId, types.CollaboratorRelation, collaboratorDID)
	return deleteRelationship(ctx, k, rel, policyId, namespaceId, ownerDID, signer)
}

// addRelationship adds new actor relationship for the specified namespace object.
func addRelationship(
	goCtx context.Context,
	k *Keeper,
	relation *coretypes.Relationship,
	policyId, namespaceId, ownerDID, signer string,
) error {
	ctx := sdk.UnwrapSDKContext(goCtx)

	manager := capability.NewPolicyCapabilityManager(k.GetScopedKeeper())
	polCap, err := manager.Fetch(ctx, policyId)
	if err != nil {
		return err
	}

	policyCmd := acptypes.NewSetRelationshipCmd(relation)
	_, err = k.GetAcpKeeper().ModulePolicyCmdForActorDID(ctx, polCap, policyCmd, ownerDID, signer)

	return err
}

// deleteRelationship deletes existing actor relationship for the specified namespace object.
func deleteRelationship(
	goCtx context.Context,
	k *Keeper,
	relation *coretypes.Relationship,
	policyId, namespaceId, ownerDID, signer string,
) error {
	ctx := sdk.UnwrapSDKContext(goCtx)

	manager := capability.NewPolicyCapabilityManager(k.GetScopedKeeper())
	polCap, err := manager.Fetch(ctx, policyId)
	if err != nil {
		return err
	}

	policyCmd := acptypes.NewDeleteRelationshipCmd(relation)
	_, err = k.GetAcpKeeper().ModulePolicyCmdForActorDID(ctx, polCap, policyCmd, ownerDID, signer)

	return err
}

// RegisterNamespace registers a new namespace object under the namespace resource.
func RegisterNamespace(ctx sdk.Context, k *Keeper, policyId, namespaceId, ownerDID, signer string) error {
	manager := capability.NewPolicyCapabilityManager(k.GetScopedKeeper())
	polCap, err := manager.Fetch(ctx, policyId)
	if err != nil {
		return err
	}

	policyCmd := acptypes.NewRegisterObjectCmd(coretypes.NewObject(types.NamespaceResource, namespaceId))
	_, err = k.GetAcpKeeper().ModulePolicyCmdForActorDID(ctx, polCap, policyCmd, ownerDID, signer)
	return err
}

// hasPermission checks if an actor has required permission for the specified namespace object.
func hasPermission(goCtx context.Context, k *Keeper, policyId, namespaceId, permission, actorDID, signer string) (bool, error) {
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
