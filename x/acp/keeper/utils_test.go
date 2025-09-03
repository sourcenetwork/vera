package keeper

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestIssueDIDFromAccountAddr(t *testing.T) {
	ctx, k, accountKeeper := setupKeeper(t)

	t.Run("empty address", func(t *testing.T) {
		did, err := k.IssueDIDFromAccountAddr(ctx, "")
		require.Error(t, err)
		require.Empty(t, did)
	})

	t.Run("valid regular account address", func(t *testing.T) {
		account := accountKeeper.GenAccount()
		addr := account.GetAddress().String()

		did, err := k.IssueDIDFromAccountAddr(ctx, addr)
		require.NoError(t, err)
		require.NotEmpty(t, did)
		require.Contains(t, did, "did:key:")
	})

	t.Run("invalid address format", func(t *testing.T) {
		invalidAddr := "invalid-address"

		did, err := k.IssueDIDFromAccountAddr(ctx, invalidAddr)
		require.Error(t, err)
		require.Empty(t, did)
		require.Contains(t, err.Error(), "IssueDIDFromAccountAddr")
	})

	t.Run("valid address but account not found", func(t *testing.T) {
		nonExistentAddr := "source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et"

		did, err := k.IssueDIDFromAccountAddr(ctx, nonExistentAddr)
		require.Error(t, err)
		require.Empty(t, did)
		require.Contains(t, err.Error(), "account not found")
	})

	t.Run("ICA address - should use IssueInterchainAccountDID", func(t *testing.T) {
		icaAccount := accountKeeper.GenAccount()
		icaAddress := icaAccount.GetAddress().String()
		controllerAddress := "shinzo1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9"
		controllerChainID := "shinzo-1"
		connectionID := "connection-0"

		err := k.SetICAConnection(sdk.UnwrapSDKContext(ctx), icaAddress, controllerAddress, controllerChainID, connectionID)
		require.NoError(t, err)

		did, err := k.IssueDIDFromAccountAddr(ctx, icaAddress)
		require.NoError(t, err)
		require.NotEmpty(t, did)
		require.Contains(t, did, "did:ica:")
	})

	t.Run("ICA address without account should still fail", func(t *testing.T) {
		icaAddress := "source1n34fvpteuanu2nx2a4hql4jvcrcnal3gsrjppy"
		controllerAddress := "shinzo1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9"
		controllerChainID := "shinzo-1"
		connectionID := "connection-1"

		err := k.SetICAConnection(sdk.UnwrapSDKContext(ctx), icaAddress, controllerAddress, controllerChainID, connectionID)
		require.NoError(t, err)

		did, err := k.IssueDIDFromAccountAddr(ctx, icaAddress)
		require.Error(t, err)
		require.Empty(t, did)
		require.Contains(t, err.Error(), "account not found")
	})

	t.Run("ICA address with account exists", func(t *testing.T) {
		icaAccount := accountKeeper.GenAccount()
		icaAddress := icaAccount.GetAddress().String()
		controllerAddress := "shinzo1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9"
		controllerChainID := "shinzo-1"
		connectionID := "connection-full"

		err := k.SetICAConnection(sdk.UnwrapSDKContext(ctx), icaAddress, controllerAddress, controllerChainID, connectionID)
		require.NoError(t, err)

		did, err := k.IssueDIDFromAccountAddr(ctx, icaAddress)
		require.NoError(t, err)
		require.NotEmpty(t, did)
		require.Contains(t, did, "did:ica:")

		connection, found := k.GetICAConnection(sdk.UnwrapSDKContext(ctx), icaAddress)
		require.True(t, found)
		require.Equal(t, controllerAddress, connection.ControllerAddress)
		require.Equal(t, controllerChainID, connection.ControllerChainId)
		require.Equal(t, connectionID, connection.ConnectionId)
	})
}
