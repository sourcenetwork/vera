package keeper

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sourcenetwork/vera/x/core/types"
)

func TestChainConfig_SetGetConfig_ReturnsSetConfig(t *testing.T) {
	ctx, k, _ := setupKeeper(t)

	cfg := types.ChainConfig{
		AllowZeroFeeTxs:  true,
		IgnoreBearerAuth: true,
	}
	err := k.SetChainConfig(ctx, cfg)
	require.NoError(t, err)

	got := k.GetChainConfig(ctx)

	require.Equal(t, cfg, got)
}

func TestChainConfig_SetWithConfigAlreadyInitialized_ReturnError(t *testing.T) {
	ctx, k, _ := setupKeeper(t)

	cfg := types.ChainConfig{
		AllowZeroFeeTxs:  true,
		IgnoreBearerAuth: true,
	}
	err := k.SetChainConfig(ctx, cfg)
	require.NoError(t, err)

	cfg2 := types.ChainConfig{
		AllowZeroFeeTxs:  false,
		IgnoreBearerAuth: true,
	}
	err = k.SetChainConfig(ctx, cfg2)
	require.ErrorIs(t, err, types.ErrConfigSet)
}

func TestChainConfig_GetWhileNotSet_ReturnsDefaultConfig(t *testing.T) {
	ctx, k, _ := setupKeeper(t)

	cfg := types.DefaultGenesis().ChainConfig
	got := k.GetChainConfig(ctx)
	require.Equal(t, cfg, got)
}
