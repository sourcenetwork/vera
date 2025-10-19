package keeper

import (
	"github.com/sourcenetwork/sourcehub/x/ica/types"
)

var _ types.MsgServer = &Keeper{}
