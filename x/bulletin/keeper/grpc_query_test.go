package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/sourcenetwork/sourcehub/testutil/keeper"
	"github.com/sourcenetwork/sourcehub/x/bulletin/types"
)

func TestParamsQuery(t *testing.T) {
	k, ctx := keepertest.BulletinKeeper(t)
	params := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, params))

	response, err := k.Params(ctx, &types.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, &types.QueryParamsResponse{Params: params}, response)
}
