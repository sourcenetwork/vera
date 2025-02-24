package keeper

import (
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

var _ types.QueryServer = Querier{}

// Querier defines a wrapper around the x/acp keeper providing gRPC method handlers.
type Querier struct {
	Keeper
}

// NewQuerier initializes new querier.
func NewQuerier(k Keeper) Querier {
	return Querier{Keeper: k}
}
