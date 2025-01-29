package keeper

import (
	"testing"
	"time"

	cryptocdc "github.com/cosmos/cosmos-sdk/crypto/codec"
	appparams "github.com/sourcenetwork/sourcehub/app/params"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authcodec "github.com/cosmos/cosmos-sdk/x/auth/codec"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	epochskeeper "github.com/sourcenetwork/sourcehub/x/epochs/keeper"
	epochstypes "github.com/sourcenetwork/sourcehub/x/epochs/types"

	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/sourcenetwork/sourcehub/x/tier/keeper"
	"github.com/sourcenetwork/sourcehub/x/tier/types"
	"github.com/stretchr/testify/require"
)

func TierKeeper(t testing.TB) (keeper.Keeper, sdk.Context) {
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	authStoreKey := storetypes.NewKVStoreKey(authtypes.StoreKey)
	bankStoreKey := storetypes.NewKVStoreKey(banktypes.StoreKey)
	stakingStoreKey := storetypes.NewKVStoreKey(stakingtypes.StoreKey)
	distrStoreKey := storetypes.NewKVStoreKey(distrtypes.StoreKey)
	epochsStoreKey := storetypes.NewKVStoreKey(epochstypes.StoreKey)

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(authStoreKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(bankStoreKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(stakingStoreKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(distrStoreKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(epochsStoreKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cryptocdc.RegisterInterfaces(registry)
	authtypes.RegisterInterfaces(registry)
	banktypes.RegisterInterfaces(registry)
	stakingtypes.RegisterInterfaces(registry)
	distrtypes.RegisterInterfaces(registry)

	cdc := codec.NewProtoCodec(registry)
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)
	bech32Prefix := "source"
	addressCodec := authcodec.NewBech32Codec(bech32Prefix)
	valOperCodec := authcodec.NewBech32Codec(bech32Prefix + "valoper")
	valConsCodec := authcodec.NewBech32Codec(bech32Prefix + "valcons")

	maccPerms := map[string][]string{
		authtypes.FeeCollectorName:     nil,
		types.ModuleName:               {authtypes.Minter, authtypes.Burner, authtypes.Staking},
		stakingtypes.BondedPoolName:    {authtypes.Burner, authtypes.Staking},
		stakingtypes.NotBondedPoolName: {authtypes.Burner, authtypes.Staking},
		distrtypes.ModuleName:          {authtypes.Minter, authtypes.Burner},
	}

	authKeeper := authkeeper.NewAccountKeeper(
		cdc,
		runtime.NewKVStoreService(authStoreKey),
		authtypes.ProtoBaseAccount,
		maccPerms,
		addressCodec,
		bech32Prefix,
		authority.String(),
	)

	blockedAddrs := make(map[string]bool)
	for acc := range maccPerms {
		blockedAddrs[authtypes.NewModuleAddress(acc).String()] = true
	}
	bankKeeper := bankkeeper.NewBaseKeeper(
		cdc,
		runtime.NewKVStoreService(bankStoreKey),
		authKeeper,
		blockedAddrs,
		authority.String(),
		log.NewNopLogger(),
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	ctx = ctx.WithBlockHeight(1).WithChainID("sourcehub").WithBlockTime(time.Unix(1000000000, 0))

	appparams.RegisterDenoms(ctx, bankKeeper)

	bondedPool := authtypes.NewEmptyModuleAccount(stakingtypes.BondedPoolName, authtypes.Burner, authtypes.Staking)
	notBondedPool := authtypes.NewEmptyModuleAccount(stakingtypes.NotBondedPoolName, authtypes.Burner, authtypes.Staking)
	tierPool := authtypes.NewEmptyModuleAccount(types.ModuleName, authtypes.Minter, authtypes.Burner)
	distrPool := authtypes.NewEmptyModuleAccount(distrtypes.ModuleName)

	if authKeeper.GetModuleAccount(ctx, stakingtypes.BondedPoolName) == nil {
		authKeeper.SetModuleAccount(ctx, bondedPool)
	}
	if authKeeper.GetModuleAccount(ctx, stakingtypes.NotBondedPoolName) == nil {
		authKeeper.SetModuleAccount(ctx, notBondedPool)
	}
	if authKeeper.GetModuleAccount(ctx, types.ModuleName) == nil {
		authKeeper.SetModuleAccount(ctx, tierPool)
	}
	if authKeeper.GetModuleAccount(ctx, distrtypes.ModuleName) == nil {
		authKeeper.SetModuleAccount(ctx, distrPool)
	}

	stakingKeeper := stakingkeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(stakingStoreKey),
		authKeeper,
		bankKeeper,
		authority.String(),
		valOperCodec,
		valConsCodec,
	)

	stakingParams := stakingtypes.DefaultParams()
	stakingParams.BondDenom = appparams.DefaultBondDenom
	require.NoError(t, stakingKeeper.SetParams(ctx, stakingParams))

	distributionKeeper := distrkeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(distrStoreKey),
		authKeeper,
		bankKeeper,
		stakingKeeper,
		authority.String(),
		authority.String(),
	)

	epochsKeeper := epochskeeper.NewKeeper(
		runtime.NewKVStoreService(epochsStoreKey),
		log.NewNopLogger(),
	)

	k := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		log.NewNopLogger(),
		authority.String(),
		bankKeeper,
		stakingKeeper,
		epochsKeeper,
		distributionKeeper,
	)

	// Initialize params
	if err := k.SetParams(ctx, types.DefaultParams()); err != nil {
		panic(err)
	}

	return k, ctx
}
