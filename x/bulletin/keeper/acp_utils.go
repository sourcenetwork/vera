package keeper

import (
	"context"
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sourcenetwork/acp_core/pkg/errors"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	sourcehubtypes "github.com/sourcenetwork/sourcehub/types"
	"github.com/sourcenetwork/sourcehub/x/acp/access_decision"
	"github.com/sourcenetwork/sourcehub/x/acp/did"
	"github.com/sourcenetwork/sourcehub/x/acp/keeper/policy_cmd"
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
func AddCollaborator(ctx context.Context, k *Keeper, namespaceId string, collaboratorDID string, ownerDID string, signer string) error {
	rel := coretypes.NewActorRelationship(types.NamespaceResource, namespaceId, types.CollaboratorRelation, collaboratorDID)
	return addRelationship(ctx, k, namespaceId, rel, ownerDID, signer)
}

// deleteCollaborator deletes existing namespace collaborator.
func deleteCollaborator(ctx context.Context, k *Keeper, namespaceId string, collaboratorDID string, ownerDID string, signer string) error {
	rel := coretypes.NewActorRelationship(types.NamespaceResource, namespaceId, types.CollaboratorRelation, collaboratorDID)
	return deleteRelationship(ctx, k, namespaceId, rel, ownerDID, signer)
}

// addRelationship adds new actor relationship for the specified namespace object.
func addRelationship(goCtx context.Context, k *Keeper, namespaceId string, relation *coretypes.Relationship, ownerDID string, signer string) error {
	policyId := k.GetPolicyId(goCtx)
	if policyId == "" {
		return types.ErrInvalidPolicyId
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	cmdCtx, err := policy_cmd.NewPolicyCmdCtx(ctx, policyId, ownerDID, signer, k.GetAcpKeeper().GetParams(ctx))
	if err != nil {
		return err
	}

	policyCmd := acptypes.NewSetRelationshipCmd(relation)
	policyCmdHandler := k.GetAcpKeeper().GetPolicyCmdHandler(ctx)

	_, err = policyCmdHandler.Dispatch(&cmdCtx, policyCmd)

	return err
}

// deleteRelationship deletes existing actor relationship for the specified namespace object.
func deleteRelationship(goCtx context.Context, k *Keeper, namespaceId string, relation *coretypes.Relationship, ownerDID string, signer string) error {
	policyId := k.GetPolicyId(goCtx)
	if policyId == "" {
		return types.ErrInvalidPolicyId
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	cmdCtx, err := policy_cmd.NewPolicyCmdCtx(ctx, policyId, ownerDID, signer, k.GetAcpKeeper().GetParams(ctx))
	if err != nil {
		return err
	}

	policyCmd := acptypes.NewDeleteRelationshipCmd(relation)
	policyCmdHandler := k.GetAcpKeeper().GetPolicyCmdHandler(ctx)

	_, err = policyCmdHandler.Dispatch(&cmdCtx, policyCmd)

	return err
}

// RegisterNamespace registers a new namespace object under the namespace resource.
func RegisterNamespace(ctx sdk.Context, k *Keeper, namespaceId string, ownerDID string, signer string) error {
	policyId := k.GetPolicyId(ctx)
	if policyId == "" {
		return types.ErrInvalidPolicyId
	}

	cmdCtx, err := policy_cmd.NewPolicyCmdCtx(ctx, policyId, ownerDID, signer, k.GetAcpKeeper().GetParams(ctx))
	if err != nil {
		return err
	}

	policyCmd := acptypes.NewRegisterObjectCmd(coretypes.NewObject(types.NamespaceResource, namespaceId))
	policyCmdHandler := k.GetAcpKeeper().GetPolicyCmdHandler(ctx)

	_, err = policyCmdHandler.Dispatch(&cmdCtx, policyCmd)
	if err != nil {
		return err
	}

	return nil
}

// hasPermission checks if an actor has required permission for the specified namespace object.
func hasPermission(goCtx context.Context, k *Keeper, namespaceId string, permission string, actorDID string, signer string) (bool, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	acpKeeper := k.GetAcpKeeper()
	repository := acpKeeper.GetAccessDecisionRepository(ctx)
	paramsRepository := access_decision.StaticParamsRepository{}
	engine := acpKeeper.GetACPEngine(ctx)

	policyId := k.GetPolicyId(goCtx)
	if policyId == "" {
		return false, types.ErrInvalidPolicyId
	}

	record, err := engine.GetPolicy(goCtx, &coretypes.GetPolicyRequest{Id: policyId})
	if err != nil {
		return false, err
	}
	if record == nil {
		return false, errors.ErrPolicyNotFound(policyId)
	}

	creatorAddr, err := sdk.AccAddressFromBech32(signer)
	if err != nil {
		return false, acptypes.NewErrInvalidAccAddrErr(err, signer)
	}

	creatorAcc := k.accountKeeper.GetAccount(goCtx, creatorAddr)
	if creatorAcc == nil {
		return false, acptypes.NewAccNotFoundErr(signer)
	}

	ts, err := acptypes.TimestampFromCtx(ctx)
	if err != nil {
		return false, err
	}

	operations := []*coretypes.Operation{
		{
			Object:     coretypes.NewObject(types.NamespaceResource, namespaceId),
			Permission: permission,
		},
	}

	cmd := access_decision.EvaluateAccessRequestsCommand{
		Policy:        record.Record.Policy,
		Operations:    operations,
		Actor:         actorDID,
		CreationTime:  ts,
		Creator:       creatorAcc,
		CurrentHeight: uint64(ctx.BlockHeight()),
	}

	decision, err := cmd.Execute(goCtx, engine, repository, &paramsRepository)
	if err != nil {
		return false, err
	}

	return decision != nil, nil
}
