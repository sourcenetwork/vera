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
		nonExistentAddr := "vera1wjj5v5rlf57kayyeskncpu4hwev25ty697gcev"

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

		err := k.coreKeeper.SetICAConnection(sdk.UnwrapSDKContext(ctx), icaAddress, controllerAddress, controllerChainID, connectionID)
		require.NoError(t, err)

		did, err := k.IssueDIDFromAccountAddr(ctx, icaAddress)
		require.NoError(t, err)
		require.NotEmpty(t, did)
		require.Contains(t, did, "did:ica:")
	})

	t.Run("ICA address without account should still fail", func(t *testing.T) {
		icaAddress := "vera1n34fvpteuanu2nx2a4hql4jvcrcnal3gqfmnpr"
		controllerAddress := "shinzo1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9"
		controllerChainID := "shinzo-1"
		connectionID := "connection-1"

		err := k.coreKeeper.SetICAConnection(sdk.UnwrapSDKContext(ctx), icaAddress, controllerAddress, controllerChainID, connectionID)
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

		err := k.coreKeeper.SetICAConnection(sdk.UnwrapSDKContext(ctx), icaAddress, controllerAddress, controllerChainID, connectionID)
		require.NoError(t, err)

		did, err := k.IssueDIDFromAccountAddr(ctx, icaAddress)
		require.NoError(t, err)
		require.NotEmpty(t, did)
		require.Contains(t, did, "did:ica:")

		connection, found := k.coreKeeper.GetICAConnection(sdk.UnwrapSDKContext(ctx), icaAddress)
		require.True(t, found)
		require.Equal(t, controllerAddress, connection.ControllerAddress)
		require.Equal(t, controllerChainID, connection.ControllerChainId)
		require.Equal(t, connectionID, connection.ConnectionId)
	})
}

func TestGetAddressFromDID(t *testing.T) {
	ctx, k, accountKeeper := setupKeeper(t)

	t.Run("round-trip: account -> DID -> address", func(t *testing.T) {
		account := accountKeeper.GenAccount()
		originalAddr := account.GetAddress()

		did, err := k.IssueDIDFromAccountAddr(ctx, originalAddr.String())
		require.NoError(t, err)
		require.NotEmpty(t, did)
		require.Contains(t, did, "did:key:")

		derivedAddr, err := k.GetAddressFromDID(ctx, did)
		require.NoError(t, err)
		require.NotNil(t, derivedAddr)

		require.Equal(t, originalAddr.String(), derivedAddr.String(),
			"Address derived from DID should match original address")
	})

	t.Run("invalid DID format", func(t *testing.T) {
		invalidDID := "not-a-valid-did"

		addr, err := k.GetAddressFromDID(ctx, invalidDID)
		require.Error(t, err)
		require.Nil(t, addr)
		require.Contains(t, err.Error(), "invalid DID")
	})

	t.Run("empty DID", func(t *testing.T) {
		addr, err := k.GetAddressFromDID(ctx, "")
		require.Error(t, err)
		require.Nil(t, addr)
	})

	t.Run("valid did:key format with secp256k1", func(t *testing.T) {
		testDID := "did:key:zQ3shokFTS3brHcDQrn82RUDfCZESWL1ZdCEJwekUDPQiYBme"

		addr, err := k.GetAddressFromDID(ctx, testDID)
		require.NoError(t, err)
		require.NotNil(t, addr)
		require.NotEmpty(t, addr.String())

		require.True(t, len(addr) > 0)
	})

	t.Run("multiple conversions of same DID produce same address", func(t *testing.T) {
		account := accountKeeper.GenAccount()
		originalAddr := account.GetAddress()
		did, err := k.IssueDIDFromAccountAddr(ctx, originalAddr.String())
		require.NoError(t, err)

		addr1, err := k.GetAddressFromDID(ctx, did)
		require.NoError(t, err)

		addr2, err := k.GetAddressFromDID(ctx, did)
		require.NoError(t, err)

		addr3, err := k.GetAddressFromDID(ctx, did)
		require.NoError(t, err)

		require.Equal(t, addr1.String(), addr2.String())
		require.Equal(t, addr2.String(), addr3.String())
		require.Equal(t, originalAddr.String(), addr1.String())
	})
}
