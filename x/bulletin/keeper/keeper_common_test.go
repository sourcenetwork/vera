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

	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/sourcenetwork/sourcehub/x/acp/capability"
	acpkeeper "github.com/sourcenetwork/sourcehub/x/acp/keeper"
	acptypes "github.com/sourcenetwork/sourcehub/x/acp/types"
	"github.com/sourcenetwork/sourcehub/x/bulletin/types"
)

func setupKeeper(t testing.TB) (Keeper, sdk.Context) {
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	authStoreKey := storetypes.NewKVStoreKey(authtypes.StoreKey)
	capabilityStoreKey := storetypes.NewKVStoreKey("capkeeper")
	capabilityMemStoreKey := storetypes.NewKVStoreKey("capkeepermem")

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeDB, db)
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
		types.ModuleName:           nil,
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
	bulletinCapKeeper := capKeeper.ScopeToModule(types.ModuleName)

	acpKeeper := acpkeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		log.NewNopLogger(),
		authority.String(),
		accountKeeper,
		&acpCapKeeper,
	)

	k := NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		log.NewNopLogger(),
		authority.String(),
		accountKeeper,
		&acpKeeper,
		&bulletinCapKeeper,
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())

	k.SetParams(ctx, types.DefaultParams())

	return k, ctx
}

func setupTestPolicy(t *testing.T, ctx sdk.Context, k Keeper) {
	_, polCap, err := k.GetAcpKeeper().CreateModulePolicy(
		ctx,
		types.BasePolicy(),
		coretypes.PolicyMarshalingType_SHORT_YAML,
		types.ModuleName,
	)
	require.NoError(t, err)

	policyId := polCap.GetPolicyId()
	k.SetPolicyId(ctx, policyId)

	manager := capability.NewPolicyCapabilityManager(k.GetScopedKeeper())
	err = manager.Claim(ctx, polCap)
	require.NoError(t, err)
}
