package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/sourcenetwork/sourcehub/testutil/keeper"
)

func TestGetParams(t *testing.T) {
	k, ctx := keepertest.TierKeeper(t)
	params := k.GetParams(ctx)

	require.NoError(t, k.SetParams(ctx, params))
	require.EqualValues(t, params, k.GetParams(ctx))
}
