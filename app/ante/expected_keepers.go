package ante

import (
	"context"
	"time"
)

// HubKeeper is an interface for the x/hub module keeper.
type HubKeeper interface {
	// App-wide configuration flags
	IsZeroFeeTxsAllowed(ctx context.Context) bool
	IsBearerAuthIgnored(ctx context.Context) bool

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
