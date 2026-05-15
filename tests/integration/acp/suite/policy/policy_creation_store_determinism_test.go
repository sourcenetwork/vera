package policy

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
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
	"github.com/stretchr/testify/require"

	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/sourcenetwork/sourcehub/x/acp/keeper"
	"github.com/sourcenetwork/sourcehub/x/acp/testutil"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
	hubtestutil "github.com/sourcenetwork/sourcehub/x/hub/testutil"
)

func TestCreatePolicyCommittedRootHashIsDeterministicAcrossStores(t *testing.T) {
	policy := `
name: Source Policy
description: A policy with enough metadata to exercise map serialization.
meta:
  alpha: one
  bravo: two
  charlie: three
  delta: four
  echo: five
  foxtrot: six
resources:
  - name: file
    permissions:
      - name: read
        expr: reader
    relations:
      - name: reader
`

	hashes := make([][]byte, 0, 3)
	policyIDs := make([]string, 0, 3)

	for i := 0; i < 3; i++ {
		ctx, k, cms, creator := setupACPCommitStore(t)

		resp, err := k.CreatePolicy(ctx, &types.MsgCreatePolicy{
			Creator:     creator,
			Policy:      policy,
			MarshalType: coretypes.PolicyMarshalingType_YAML,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)

		policyIDs = append(policyIDs, resp.Record.Policy.Id)
		hashes = append(hashes, cms.Commit().Hash)
	}

	for i := 0; i < 2; i++ {
		require.Equal(t, policyIDs[i], policyIDs[i+1])
		require.Equal(t, hashes[i], hashes[i+1])
	}
}

func setupACPCommitStore(t *testing.T) (sdk.Context, keeper.Keeper, storetypes.CommitMultiStore, string) {
	t.Helper()

	acpStoreKey := storetypes.NewKVStoreKey(types.StoreKey)
	capabilityStoreKey := storetypes.NewKVStoreKey("capkeeper")
	capabilityMemStoreKey := storetypes.NewKVStoreKey("capkeepermem")

	db := dbm.NewMemDB()
	cms := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	cms.MountStoreWithDB(acpStoreKey, storetypes.StoreTypeDB, db)
	cms.MountStoreWithDB(capabilityStoreKey, storetypes.StoreTypeDB, db)
	cms.MountStoreWithDB(capabilityMemStoreKey, storetypes.StoreTypeDB, db)
	require.NoError(t, cms.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)

	capKeeper := capabilitykeeper.NewKeeper(cdc, capabilityStoreKey, capabilityMemStoreKey)
	acpCapKeeper := capKeeper.ScopeToModule(types.ModuleName)

	accKeeper := &testutil.AccountKeeperStub{}
	pubKey := secp256k1.GenPrivKeyFromSecret([]byte("acp-create-policy-store-determinism")).PubKey()
	account := accKeeper.NewAccount(pubKey)

	acpKeeper := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(acpStoreKey),
		log.NewNopLogger(),
		authority.String(),
		accKeeper,
		&acpCapKeeper,
		hubtestutil.NewHubKeeperStub(),
	)

	ctx := sdk.NewContext(cms, cmtproto.Header{}, false, log.NewNopLogger())
	ctx = ctx.WithEventManager(sdk.NewEventManager())
	acpKeeper.SetParams(ctx, types.DefaultParams())

	return ctx, acpKeeper, cms, account.GetAddress().String()
}
