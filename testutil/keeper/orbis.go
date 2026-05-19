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
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
	"github.com/stretchr/testify/require"

	acpkeeper "github.com/sourcenetwork/sourcehub/x/acp/keeper"
	acptypes "github.com/sourcenetwork/sourcehub/x/acp/types"
	hubtestutil "github.com/sourcenetwork/sourcehub/x/hub/testutil"
	"github.com/sourcenetwork/sourcehub/x/orbis/keeper"
	"github.com/sourcenetwork/sourcehub/x/orbis/types"
)

func OrbisKeeper(t testing.TB) (keeper.Keeper, sdk.Context) {
	orbisStoreKey := storetypes.NewKVStoreKey(types.StoreKey)
	acpStoreKey := storetypes.NewKVStoreKey(acptypes.StoreKey)
	authStoreKey := storetypes.NewKVStoreKey(authtypes.StoreKey)
	capabilityStoreKey := storetypes.NewKVStoreKey("capkeeper")
	capabilityMemStoreKey := storetypes.NewKVStoreKey("capkeepermem")

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(orbisStoreKey, storetypes.StoreTypeDB, db)
	stateStore.MountStoreWithDB(acpStoreKey, storetypes.StoreTypeDB, db)
	stateStore.MountStoreWithDB(authStoreKey, storetypes.StoreTypeDB, db)
	stateStore.MountStoreWithDB(capabilityStoreKey, storetypes.StoreTypeDB, db)
	stateStore.MountStoreWithDB(capabilityMemStoreKey, storetypes.StoreTypeDB, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	authtypes.RegisterInterfaces(registry)
	cryptocodec.RegisterInterfaces(registry)

	cdc := codec.NewProtoCodec(registry)
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)
	bech32Prefix := "source"
	addressCodec := authcodec.NewBech32Codec(bech32Prefix)

	maccPerms := map[string][]string{
		authtypes.FeeCollectorName: nil,
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

	k := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(orbisStoreKey),
		log.NewNopLogger(),
		authority.String(),
		accountKeeper,
		&acpKeeper,
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	k.SetParams(ctx, types.DefaultParams())

	return k, ctx
}
