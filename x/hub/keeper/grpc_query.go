package keeper

import (
	"github.com/sourcenetwork/sourcehub/x/hub/types"
)

var _ types.QueryServer = &Keeper{}
