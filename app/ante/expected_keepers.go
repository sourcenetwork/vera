package ante

import (
	"context"
	"time"

	hubtypes "github.com/sourcenetwork/sourcehub/x/hub/types"
)

// HubKeeper is an interface for the x/hub module keeper.
type HubKeeper interface {
	GetChainConfig(context.Context) hubtypes.ChainConfig
	GetParams(context.Context) hubtypes.Params

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
