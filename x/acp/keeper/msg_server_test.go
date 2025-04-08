package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/sourcenetwork/sourcehub/testutil/keeper"
)

func TestMsgServer(t *testing.T) {
	k, ctx := keepertest.AcpKeeper(t)
	require.NotNil(t, ctx)
	require.NotEmpty(t, k)
}
