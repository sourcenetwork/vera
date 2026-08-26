package keeper

import (
	"testing"
	"time"

	errorsmod "cosmossdk.io/errors"
	"github.com/stretchr/testify/require"

	"github.com/sourcenetwork/vera/x/core/types"
)

func TestStoreOrUpdateJWSTokenRejectsInvalidatedToken(t *testing.T) {
	ctx, keeper, _ := setupKeeper(t)
	ctx = ctx.WithBlockTime(time.Now())
	bearer := "header.payload.signature"
	issuedAt := ctx.BlockTime().Add(-time.Minute)
	expiresAt := ctx.BlockTime().Add(time.Minute)

	require.NoError(t, keeper.StoreOrUpdateJWSToken(
		ctx,
		bearer,
		"did:example:user",
		"source1r5v5srda7xfth3hn2s26txvrcrntldjuac798p",
		issuedAt,
		expiresAt,
	))
	hash := types.HashJWSToken(bearer)
	require.NoError(t, keeper.UpdateJWSTokenStatus(
		ctx,
		hash,
		types.JWSTokenStatus_STATUS_INVALID,
		"source1r5v5srda7xfth3hn2s26txvrcrntldjuac798p",
	))

	err := keeper.StoreOrUpdateJWSToken(
		ctx,
		bearer,
		"did:example:user",
		"source1r5v5srda7xfth3hn2s26txvrcrntldjuac798p",
		issuedAt,
		expiresAt,
	)
	require.Error(t, err)
	require.True(t, errorsmod.IsOf(err, types.ErrJWSTokenInvalid))
}
