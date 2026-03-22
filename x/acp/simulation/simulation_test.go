package simulation

import (
	"math/rand"
	"testing"

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
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
	"github.com/stretchr/testify/require"

	"github.com/sourcenetwork/sourcehub/x/acp/keeper"
	"github.com/sourcenetwork/sourcehub/x/acp/testutil"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
	hubtestutil "github.com/sourcenetwork/sourcehub/x/hub/testutil"
)

func setupSimKeeper(t *testing.T) (sdk.Context, *keeper.Keeper) {
	acpStoreKey := storetypes.NewKVStoreKey(types.StoreKey)
	capStoreKey := storetypes.NewKVStoreKey("capkeeper")
	capMemStoreKey := storetypes.NewKVStoreKey("capkeepermem")

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(acpStoreKey, storetypes.StoreTypeDB, db)
	stateStore.MountStoreWithDB(capStoreKey, storetypes.StoreTypeDB, db)
	stateStore.MountStoreWithDB(capMemStoreKey, storetypes.StoreTypeDB, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)
	capKeeper := capabilitykeeper.NewKeeper(cdc, capStoreKey, capMemStoreKey)
	acpCapKeeper := capKeeper.ScopeToModule(types.ModuleName)

	accKeeper := &testutil.AccountKeeperStub{}
	k := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(acpStoreKey),
		log.NewNopLogger(),
		authority.String(),
		accKeeper,
		&acpCapKeeper,
		hubtestutil.NewHubKeeperStub(),
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	return ctx, &k
}

func makeSimAccounts() []simtypes.Account {
	r := rand.New(rand.NewSource(42))
	return simtypes.RandomAccounts(r, 3)
}

func TestFindAccount(t *testing.T) {
	accs := makeSimAccounts()
	addr := accs[0].Address.String()

	found, ok := FindAccount(accs, addr)
	require.True(t, ok)
	require.Equal(t, accs[0].Address, found.Address)
}

func TestFindAccountNotFound(t *testing.T) {
	accs := makeSimAccounts()
	// use a valid bech32 address that isn't in the list
	otherAccs := simtypes.RandomAccounts(rand.New(rand.NewSource(99)), 1)
	_, ok := FindAccount(accs, otherAccs[0].Address.String())
	require.False(t, ok)
}

func TestFindAccountPanicsOnInvalidAddress(t *testing.T) {
	accs := makeSimAccounts()
	require.Panics(t, func() {
		FindAccount(accs, "not-bech32")
	})
}

func TestSimulateMsgCreatePolicy(t *testing.T) {
	ctx, k := setupSimKeeper(t)
	r := rand.New(rand.NewSource(42))
	accs := makeSimAccounts()

	op := SimulateMsgCreatePolicy(nil, nil, k)
	msg, futures, err := op(r, nil, ctx, accs, "test-chain")
	require.NoError(t, err)
	require.Nil(t, futures)
	require.Contains(t, msg.Comment, "not implemented")
}

func TestSimulateMsgCheckAccess(t *testing.T) {
	ctx, k := setupSimKeeper(t)
	r := rand.New(rand.NewSource(42))
	accs := makeSimAccounts()

	op := SimulateMsgCheckAccess(nil, nil, k)
	msg, futures, err := op(r, nil, ctx, accs, "test-chain")
	require.NoError(t, err)
	require.Nil(t, futures)
	require.Contains(t, msg.Comment, "not implemented")
}

func TestSimulateMsgPolicyCmd(t *testing.T) {
	ctx, k := setupSimKeeper(t)
	r := rand.New(rand.NewSource(42))
	accs := makeSimAccounts()

	op := SimulateMsgPolicyCmd(nil, nil, k)
	msg, futures, err := op(r, nil, ctx, accs, "test-chain")
	require.NoError(t, err)
	require.Nil(t, futures)
	require.Contains(t, msg.Comment, "not implemented")
}

func TestSimulateMsgMsgEditPolicy(t *testing.T) {
	ctx, k := setupSimKeeper(t)
	r := rand.New(rand.NewSource(42))
	accs := makeSimAccounts()

	op := SimulateMsgMsgEditPolicy(nil, nil, k)
	msg, futures, err := op(r, nil, ctx, accs, "test-chain")
	require.NoError(t, err)
	require.Nil(t, futures)
	require.Contains(t, msg.Comment, "not implemented")
}
