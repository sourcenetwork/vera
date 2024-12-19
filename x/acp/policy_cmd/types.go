package policy_cmd

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	prototypes "github.com/cosmos/gogoproto/types"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

type PolicyCmdCtx struct {
	PolicyId      string
	PrincipalDID  string
	Now           *prototypes.Timestamp
	SDKCtx        sdk.Context
	EngineContext context.Context
	Params        types.Params
	Signer        string
}
