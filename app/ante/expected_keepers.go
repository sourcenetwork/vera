package ante

import (
	"context"
	"time"

	coretypes "github.com/sourcenetwork/vera/x/core/types"
)

// CoreKeeper is an interface for the x/core module keeper.
type CoreKeeper interface {
	GetChainConfig(context.Context) coretypes.ChainConfig
	GetParams(context.Context) coretypes.Params

	// JWS token management
	StoreOrUpdateJWSToken(
		ctx context.Context,
		bearerToken string,
		issuerDid string,
		authorizedAccount string,
		issuedAt time.Time,
		expiresAt time.Time,
	) error
}
