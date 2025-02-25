package keeper

import (
	"testing"
	"time"

	cryptocdc "github.com/cosmos/cosmos-sdk/crypto/codec"
	cosmosed25519 "github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
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
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	appparams "github.com/sourcenetwork/sourcehub/app/params"
	testutil "github.com/sourcenetwork/sourcehub/testutil"
	epochskeeper "github.com/sourcenetwork/sourcehub/x/epochs/keeper"
	epochstypes "github.com/sourcenetwork/sourcehub/x/epochs/types"
	"github.com/sourcenetwork/sourcehub/x/tier/keeper"
	"github.com/sourcenetwork/sourcehub/x/tier/types"
	"github.com/stretchr/testify/require"
)

// initializeValidator creates a validator and verifies that it was set correctly.
func InitializeValidator(t *testing.T, k *stakingkeeper.Keeper, ctx sdk.Context, valAddr sdk.ValAddress, initialTokens math.Int) {
	validator := testutil.CreateTestValidator(t, ctx, k, valAddr, cosmosed25519.GenPrivKey().PubKey(), initialTokens)
	gotValidator, err := k.GetValidator(ctx, valAddr)
	require.NoError(t, err)
	require.Equal(t, validator.OperatorAddress, gotValidator.OperatorAddress)
}

// initializeDelegator initializes a delegator with balance.
func InitializeDelegator(t *testing.T, k *keeper.Keeper, ctx sdk.Context, delAddr sdk.AccAddress, initialBalance math.Int) {
	initialDelegatorBalance := sdk.NewCoins(sdk.NewCoin(appparams.DefaultBondDenom, initialBalance))
	err := k.GetBankKeeper().MintCoins(ctx, types.ModuleName, initialDelegatorBalance)
	require.NoError(t, err)
	err = k.GetBankKeeper().SendCoinsFromModuleToAccount(ctx, types.ModuleName, delAddr, initialDelegatorBalance)
	require.NoError(t, err)
}

func TierKeeper(t testing.TB) (keeper.Keeper, sdk.Context) {
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	authStoreKey := storetypes.NewKVStoreKey(authtypes.StoreKey)
	bankStoreKey := storetypes.NewKVStoreKey(banktypes.StoreKey)
	stakingStoreKey := storetypes.NewKVStoreKey(stakingtypes.StoreKey)
	distrStoreKey := storetypes.NewKVStoreKey(distrtypes.StoreKey)
	epochsStoreKey := storetypes.NewKVStoreKey(epochstypes.StoreKey)
	mintStoreKey := storetypes.NewKVStoreKey(minttypes.StoreKey)

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeDB, db)
	stateStore.MountStoreWithDB(authStoreKey, storetypes.StoreTypeDB, db)
	stateStore.MountStoreWithDB(bankStoreKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(stakingStoreKey, storetypes.StoreTypeDB, db)
	stateStore.MountStoreWithDB(distrStoreKey, storetypes.StoreTypeDB, db)
	stateStore.MountStoreWithDB(epochsStoreKey, storetypes.StoreTypeDB, db)
	stateStore.MountStoreWithDB(mintStoreKey, storetypes.StoreTypeDB, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cryptocdc.RegisterInterfaces(registry)
	authtypes.RegisterInterfaces(registry)
	banktypes.RegisterInterfaces(registry)
	stakingtypes.RegisterInterfaces(registry)
	distrtypes.RegisterInterfaces(registry)
	minttypes.RegisterInterfaces(registry)

	cdc := codec.NewProtoCodec(registry)
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)
	bech32Prefix := "source"
	addressCodec := authcodec.NewBech32Codec(bech32Prefix)
	valOperCodec := authcodec.NewBech32Codec(bech32Prefix + "valoper")
	valConsCodec := authcodec.NewBech32Codec(bech32Prefix + "valcons")

	maccPerms := map[string][]string{
		authtypes.FeeCollectorName:     nil,
		stakingtypes.BondedPoolName:    {authtypes.Burner, authtypes.Staking},
		stakingtypes.NotBondedPoolName: {authtypes.Burner, authtypes.Staking},
		distrtypes.ModuleName:          {authtypes.Minter, authtypes.Burner},
		minttypes.ModuleName:           {authtypes.Minter, authtypes.Burner},
		types.ModuleName:               {authtypes.Minter, authtypes.Burner, authtypes.Staking},
		types.InsurancePoolName:        nil,
		types.DeveloperPoolName:        nil,
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

	bondedPool := authtypes.NewEmptyModuleAccount(stakingtypes.BondedPoolName, authtypes.Burner, authtypes.Staking)
	notBondedPool := authtypes.NewEmptyModuleAccount(stakingtypes.NotBondedPoolName, authtypes.Burner, authtypes.Staking)
	distrPool := authtypes.NewEmptyModuleAccount(distrtypes.ModuleName)
	mintAcc := authtypes.NewEmptyModuleAccount(minttypes.ModuleName)
	tierPool := authtypes.NewEmptyModuleAccount(types.ModuleName, authtypes.Minter, authtypes.Burner, authtypes.Staking)
	insurancePool := authtypes.NewEmptyModuleAccount(types.InsurancePoolName)
	developerPool := authtypes.NewEmptyModuleAccount(types.DeveloperPoolName)

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
	if authKeeper.GetModuleAccount(ctx, minttypes.ModuleName) == nil {
		authKeeper.SetModuleAccount(ctx, mintAcc)
	}
	if authKeeper.GetModuleAccount(ctx, types.InsurancePoolName) == nil {
		authKeeper.SetModuleAccount(ctx, insurancePool)
	}
	if authKeeper.GetModuleAccount(ctx, types.DeveloperPoolName) == nil {
		authKeeper.SetModuleAccount(ctx, developerPool)
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
		authtypes.FeeCollectorName,
		authority.String(),
	)

	epochsKeeper := epochskeeper.NewKeeper(
		runtime.NewKVStoreService(epochsStoreKey),
		log.NewNopLogger(),
	)

	epochInfo := epochstypes.EpochInfo{
		Identifier:            types.EpochIdentifier,
		CurrentEpoch:          1,
		CurrentEpochStartTime: ctx.BlockTime(),
		Duration:              time.Hour * 24 * 30,
	}
	epochsKeeper.SetEpochInfo(ctx, epochInfo)

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
