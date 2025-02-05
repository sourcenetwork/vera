package policy_cmd

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

func NewPolicyCmdCtx(ctx sdk.Context, policyId string, actorDID string, signer string, params types.Params) (PolicyCmdCtx, error) {
	ts, err := types.TimestampFromCtx(ctx)
	if err != nil {
		return PolicyCmdCtx{}, err
	}

	return PolicyCmdCtx{
		Ctx:          ctx,
		PolicyId:     policyId,
		PrincipalDID: actorDID,
		Now:          ts,
		Params:       params,
		Signer:       signer,
	}, nil
}

// PolicyCmdCtx bundles actor and time data bound to a PolicyCmd
type PolicyCmdCtx struct {
	Ctx          sdk.Context
	PolicyId     string
	PrincipalDID string
	Now          *types.Timestamp
	Params       types.Params
	Signer       string
}
