package keeper

import (
	"github.com/sourcenetwork/vera/x/core/types"
)

var _ types.MsgServer = &Keeper{}
