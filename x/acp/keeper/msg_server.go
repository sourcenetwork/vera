package keeper

import (
	"github.com/sourcenetwork/vera/x/acp/types"
)

var _ types.MsgServer = &Keeper{}
