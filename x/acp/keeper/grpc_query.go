package keeper

import (
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

var _ types.QueryServer = Keeper{}
