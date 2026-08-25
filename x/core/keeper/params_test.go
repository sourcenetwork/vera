package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/sourcenetwork/vera/testutil/keeper"
	"github.com/sourcenetwork/vera/x/core/types"
)

func TestGetParams(t *testing.T) {
	k, ctx := keepertest.CoreKeeper(t)
	params := types.DefaultParams()

	require.NoError(t, k.SetParams(ctx, params))
	require.EqualValues(t, params, k.GetParams(ctx))
}

func TestSetParamsRejectsInvalidTrustedRelay(t *testing.T) {
	k, ctx := keepertest.CoreKeeper(t)
	err := k.SetParams(ctx, types.Params{TrustedRelayFeeGranters: []string{"invalid"}})
	require.Error(t, err)
}
