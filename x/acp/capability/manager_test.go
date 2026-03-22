package capability

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
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
	"github.com/stretchr/testify/require"
)

func setupCapKeeper(t *testing.T) (sdk.Context, *capabilitykeeper.ScopedKeeper, *capabilitykeeper.ScopedKeeper) {
	capStoreKey := storetypes.NewKVStoreKey("capkeeper")
	capMemStoreKey := storetypes.NewMemoryStoreKey("capkeepermem")

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(capStoreKey, storetypes.StoreTypeDB, db)
	stateStore.MountStoreWithDB(capMemStoreKey, storetypes.StoreTypeMemory, nil)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	capKeeper := capabilitykeeper.NewKeeper(cdc, capStoreKey, capMemStoreKey)

	acpScoped := capKeeper.ScopeToModule("acp")
	otherScoped := capKeeper.ScopeToModule("other")

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	capKeeper.Seal()

	return ctx, &acpScoped, &otherScoped
}

func TestNewPolicyCapabilityManager(t *testing.T) {
	_, acpScoped, _ := setupCapKeeper(t)
	mgr := NewPolicyCapabilityManager(acpScoped)
	require.NotNil(t, mgr)
}

func TestIssueAndFetch(t *testing.T) {
	ctx, acpScoped, _ := setupCapKeeper(t)
	mgr := NewPolicyCapabilityManager(acpScoped)

	cap, err := mgr.Issue(ctx, "policy-1")
	require.NoError(t, err)
	require.NotNil(t, cap)
	require.Equal(t, "policy-1", cap.GetPolicyId())
	require.NotNil(t, cap.GetCosmosCapability())

	fetched, err := mgr.Fetch(ctx, "policy-1")
	require.NoError(t, err)
	require.NotNil(t, fetched)
	require.Equal(t, "policy-1", fetched.GetPolicyId())
}

func TestFetchNonExistent(t *testing.T) {
	ctx, acpScoped, _ := setupCapKeeper(t)
	mgr := NewPolicyCapabilityManager(acpScoped)

	_, err := mgr.Fetch(ctx, "nonexistent")
	require.Error(t, err)
}

func TestClaimCapability(t *testing.T) {
	ctx, acpScoped, otherScoped := setupCapKeeper(t)
	acpMgr := NewPolicyCapabilityManager(acpScoped)
	otherMgr := NewPolicyCapabilityManager(otherScoped)

	// acp issues
	cap, err := acpMgr.Issue(ctx, "policy-1")
	require.NoError(t, err)

	// other module claims
	err = otherMgr.Claim(ctx, cap)
	require.NoError(t, err)

	// other module can now fetch
	fetched, err := otherMgr.Fetch(ctx, "policy-1")
	require.NoError(t, err)
	require.NotNil(t, fetched)
}

func TestValidateCapability(t *testing.T) {
	ctx, acpScoped, otherScoped := setupCapKeeper(t)
	acpMgr := NewPolicyCapabilityManager(acpScoped)
	otherMgr := NewPolicyCapabilityManager(otherScoped)

	cap, err := acpMgr.Issue(ctx, "policy-1")
	require.NoError(t, err)

	err = otherMgr.Claim(ctx, cap)
	require.NoError(t, err)

	// validate from the other module's perspective — acp is an owner, so it passes
	err = otherMgr.Validate(ctx, cap)
	require.NoError(t, err)
}

func TestGetOwnerModule(t *testing.T) {
	ctx, acpScoped, otherScoped := setupCapKeeper(t)
	acpMgr := NewPolicyCapabilityManager(acpScoped)
	otherMgr := NewPolicyCapabilityManager(otherScoped)

	cap, err := acpMgr.Issue(ctx, "policy-1")
	require.NoError(t, err)

	err = otherMgr.Claim(ctx, cap)
	require.NoError(t, err)

	owner, err := otherMgr.GetOwnerModule(ctx, cap)
	require.NoError(t, err)
	require.Equal(t, "other", owner)
}

func TestGetOwnerModuleNoClaimer(t *testing.T) {
	ctx, acpScoped, _ := setupCapKeeper(t)
	acpMgr := NewPolicyCapabilityManager(acpScoped)

	cap, err := acpMgr.Issue(ctx, "policy-1")
	require.NoError(t, err)

	// only acp owns it, after filtering acp out, mods is empty
	_, err = acpMgr.GetOwnerModule(ctx, cap)
	require.Error(t, err)
}

func TestValidateAcpOnlyOwner(t *testing.T) {
	ctx, acpScoped, _ := setupCapKeeper(t)
	acpMgr := NewPolicyCapabilityManager(acpScoped)

	cap, err := acpMgr.Issue(ctx, "policy-1")
	require.NoError(t, err)

	// Validate should pass since acp issued it
	err = acpMgr.Validate(ctx, cap)
	require.NoError(t, err)
}
