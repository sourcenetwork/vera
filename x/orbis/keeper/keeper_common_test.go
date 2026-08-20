package keeper

import (
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authcodec "github.com/cosmos/cosmos-sdk/x/auth/codec"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
	"github.com/stretchr/testify/require"

	acpkeeper "github.com/sourcenetwork/sourcehub/x/acp/keeper"
	acptypes "github.com/sourcenetwork/sourcehub/x/acp/types"
	hubtestutil "github.com/sourcenetwork/sourcehub/x/hub/testutil"
	"github.com/sourcenetwork/sourcehub/x/orbis/types"
)

// testMintModuleName is a fictitious module account used only by tests to mint
// coins into accounts before exercising bank-moving orbis messages (e.g. DrainNodeKey).
const testMintModuleName = "orbistestmint"

func setupOrbisKeeper(t testing.TB) (Keeper, authkeeper.AccountKeeper, sdk.Context) {
	t.Helper()
	k, accountKeeper, _, ctx := setupOrbisKeeperWithBank(t)
	return k, accountKeeper, ctx
}

// setupOrbisKeeperWithBank is like setupOrbisKeeper but also wires and returns a
// real bank keeper, for tests that need to fund accounts (e.g. DrainNodeKey tests).
func setupOrbisKeeperWithBank(t testing.TB) (Keeper, authkeeper.AccountKeeper, bankkeeper.BaseKeeper, sdk.Context) {
	t.Helper()

	orbisStoreKey := storetypes.NewKVStoreKey(types.StoreKey)
	acpStoreKey := storetypes.NewKVStoreKey(acptypes.StoreKey)
	authStoreKey := storetypes.NewKVStoreKey(authtypes.StoreKey)
	bankStoreKey := storetypes.NewKVStoreKey(banktypes.StoreKey)
	capabilityStoreKey := storetypes.NewKVStoreKey("capkeeper")
	capabilityMemStoreKey := storetypes.NewKVStoreKey("capkeepermem")

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(orbisStoreKey, storetypes.StoreTypeDB, db)
	stateStore.MountStoreWithDB(acpStoreKey, storetypes.StoreTypeDB, db)
	stateStore.MountStoreWithDB(authStoreKey, storetypes.StoreTypeDB, db)
	stateStore.MountStoreWithDB(bankStoreKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(capabilityStoreKey, storetypes.StoreTypeDB, db)
	stateStore.MountStoreWithDB(capabilityMemStoreKey, storetypes.StoreTypeDB, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	authtypes.RegisterInterfaces(registry)
	banktypes.RegisterInterfaces(registry)
	cryptocodec.RegisterInterfaces(registry)

	cdc := codec.NewProtoCodec(registry)
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)
	bech32Prefix := "source"
	addressCodec := authcodec.NewBech32Codec(bech32Prefix)

	maccPerms := map[string][]string{
		authtypes.FeeCollectorName: nil,
		testMintModuleName:         {authtypes.Minter, authtypes.Burner},
	}

	accountKeeper := authkeeper.NewAccountKeeper(
		cdc,
		runtime.NewKVStoreService(authStoreKey),
		authtypes.ProtoBaseAccount,
		maccPerms,
		addressCodec,
		bech32Prefix,
		authority.String(),
	)

	blockedAddrs := map[string]bool{
		authtypes.NewModuleAddress(testMintModuleName).String(): true,
	}
	// NewBaseKeeper decodes the authority argument via accountKeeper's address codec
	// (bech32Prefix "source"), not the process-global sdk.Config prefix, so it must be
	// re-encoded with addressCodec rather than reusing authority.String().
	bankAuthority, err := addressCodec.BytesToString(authority)
	require.NoError(t, err)
	bankKeeper := bankkeeper.NewBaseKeeper(
		cdc,
		runtime.NewKVStoreService(bankStoreKey),
		accountKeeper,
		blockedAddrs,
		bankAuthority,
		log.NewNopLogger(),
	)

	capKeeper := capabilitykeeper.NewKeeper(cdc, capabilityStoreKey, capabilityMemStoreKey)
	acpCapKeeper := capKeeper.ScopeToModule(acptypes.ModuleName)

	acpKeeper := acpkeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(acpStoreKey),
		log.NewNopLogger(),
		authority.String(),
		accountKeeper,
		&acpCapKeeper,
		hubtestutil.NewHubKeeperStub(),
	)

	k := NewKeeper(
		cdc,
		runtime.NewKVStoreService(orbisStoreKey),
		log.NewNopLogger(),
		authority.String(),
		accountKeeper,
		bankKeeper,
		&acpKeeper,
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	k.SetParams(ctx, types.DefaultParams())

	// GetModuleAccount (rather than manually building+SetModuleAccount) ensures the
	// module account is assigned a proper auto-incremented account number, avoiding a
	// collision with account numbers later assigned to test accounts.
	accountKeeper.GetModuleAccount(ctx, testMintModuleName)

	return k, accountKeeper, bankKeeper, ctx
}

// fundAccount mints coins and sends them to addr, for tests that need to seed a
// balance (e.g. before exercising DrainNodeKey).
func fundAccount(t testing.TB, ctx sdk.Context, bankKeeper bankkeeper.BaseKeeper, addr sdk.AccAddress, coins sdk.Coins) {
	t.Helper()
	require.NoError(t, bankKeeper.MintCoins(ctx, testMintModuleName, coins))
	require.NoError(t, bankKeeper.SendCoinsFromModuleToAccount(ctx, testMintModuleName, addr, coins))
}
