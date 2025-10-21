package keeper

import (
	"github.com/sourcenetwork/sourcehub/x/hub/types"
)

var _ types.MsgServer = &Keeper{}
