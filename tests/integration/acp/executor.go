package test

import (
	"context"
	"fmt"
	"testing"

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
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	appparams "github.com/sourcenetwork/sourcehub/app/params"
	hubsdk "github.com/sourcenetwork/sourcehub/sdk"
	"github.com/sourcenetwork/sourcehub/testutil/e2e"
	"github.com/sourcenetwork/sourcehub/x/acp/keeper"
	"github.com/sourcenetwork/sourcehub/x/acp/testutil"
)

type KeeperExecutor struct {
	k              types.MsgServer
	accountCreator *testutil.AccountKeeperStub
}

func (e *KeeperExecutor) Cleanup() {}

func (e *KeeperExecutor) BearerPolicyCmd(ctx *TestCtx, msg *types.MsgBearerPolicyCmd) (*types.MsgBearerPolicyCmdResponse, error) {
	return e.k.BearerPolicyCmd(ctx, msg)
}

func (e *KeeperExecutor) SignedPolicyCmd(ctx *TestCtx, msg *types.MsgSignedPolicyCmd) (*types.MsgSignedPolicyCmdResponse, error) {
	return e.k.SignedPolicyCmd(ctx, msg)
}

func (e *KeeperExecutor) DirectPolicyCmd(ctx *TestCtx, msg *types.MsgDirectPolicyCmd) (*types.MsgDirectPolicyCmdResponse, error) {
	return e.k.DirectPolicyCmd(ctx, msg)
}

func (e *KeeperExecutor) CreatePolicy(ctx *TestCtx, msg *types.MsgCreatePolicy) (*types.MsgCreatePolicyResponse, error) {
	return e.k.CreatePolicy(ctx, msg)
}

func (e *KeeperExecutor) GetOrCreateAccountFromActor(_ *TestCtx, actor *TestActor) (sdk.AccountI, error) {
	return e.accountCreator.NewAccount(actor.PubKey), nil
}

func NewExecutor(t *testing.T, strategy ExecutorStrategy) (context.Context, MsgExecutor) {
	switch strategy {
	case Keeper:
		ctx, exec, err := newKeeperExecutor()
		require.NoError(t, err)
		return ctx, exec
	case SDK:
		network := &e2e.TestNetwork{}
		network.Setup(t)
		executor, err := newSDKExecutor(network)
		require.NoError(t, err)
		return context.Background(), executor
	case CLI:
		panic("sdk executor not implemented")
	default:
		panic(fmt.Sprintf("invalid executor strategy: %v", strategy))
	}
}

func newKeeperExecutor() (context.Context, MsgExecutor, error) {
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	err := stateStore.LoadLatestVersion()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create keeper executor: %v", err)
	}

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)

	accKeeper := &testutil.AccountKeeperStub{}
	accKeeper.GenAccount()

	kv := runtime.NewKVStoreService(storeKey)

	k := keeper.NewKeeper(
		cdc,
		kv,
		log.NewNopLogger(),
		authority.String(),
		accKeeper,
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	ctx = ctx.WithEventManager(sdk.NewEventManager())
	ctx = ctx.WithMultiStore(stateStore)

	// Initialize params
	k.SetParams(ctx, types.DefaultParams())

	msgServer := keeper.NewMsgServerImpl(k)
	executor := &KeeperExecutor{
		k:              msgServer,
		accountCreator: accKeeper,
	}
	return ctx, executor, nil
}

func newSDKExecutor(network *e2e.TestNetwork) (*SDKClientExecutor, error) {
	txBuilder, err := hubsdk.NewTxBuilder(
		hubsdk.WithSDKClient(network.Client),
		hubsdk.WithChainID(network.GetChainID()),
	)

	if err != nil {
		return nil, err
	}
	return &SDKClientExecutor{
		Network:   network,
		txBuilder: &txBuilder,
	}, nil
}

type SDKClientExecutor struct {
	Network   *e2e.TestNetwork
	txBuilder *hubsdk.TxBuilder
}

func (e *SDKClientExecutor) BearerPolicyCmd(ctx *TestCtx, msg *types.MsgBearerPolicyCmd) (*types.MsgBearerPolicyCmdResponse, error) {
	set := hubsdk.MsgSet{}
	mapper := set.WithBearerPolicyCmd(msg)
	result := e.broadcastTx(ctx, &set)
	if result.Error() != nil {
		return nil, result.Error()
	}
	response, err := mapper.Map(result.TxPayload())
	require.NoError(ctx.T, err)
	return response, nil
}

func (e *SDKClientExecutor) SignedPolicyCmd(ctx *TestCtx, msg *types.MsgSignedPolicyCmd) (*types.MsgSignedPolicyCmdResponse, error) {
	set := hubsdk.MsgSet{}
	mapper := set.WithSignedPolicyCmd(msg)
	result := e.broadcastTx(ctx, &set)
	if result.Error() != nil {
		return nil, result.Error()
	}
	response, err := mapper.Map(result.TxPayload())
	require.NoError(ctx.T, err)
	return response, nil
}

func (e *SDKClientExecutor) DirectPolicyCmd(ctx *TestCtx, msg *types.MsgDirectPolicyCmd) (*types.MsgDirectPolicyCmdResponse, error) {
	set := hubsdk.MsgSet{}
	mapper := set.WithDirectPolicyCmd(msg)
	result := e.broadcastTx(ctx, &set)
	if result.Error() != nil {
		return nil, result.Error()
	}
	response, err := mapper.Map(result.TxPayload())
	require.NoError(ctx.T, err)
	return response, nil
}

func (e *SDKClientExecutor) CreatePolicy(ctx *TestCtx, msg *types.MsgCreatePolicy) (*types.MsgCreatePolicyResponse, error) {
	set := hubsdk.MsgSet{}
	mapper := set.WithCreatePolicy(msg)
	result := e.broadcastTx(ctx, &set)
	if result.Error() != nil {
		return nil, result.Error()
	}
	response, err := mapper.Map(result.TxPayload())
	require.NoError(ctx.T, err)
	return response, nil
}

func (e *SDKClientExecutor) GetOrCreateAccountFromActor(ctx *TestCtx, actor *TestActor) (sdk.AccountI, error) {
	client := e.Network.Client
	resp, err := client.AuthQueryClient().AccountInfo(ctx, &authtypes.QueryAccountInfoRequest{
		Address: actor.SourceHubAddr,
	})
	if resp != nil {
		return resp.Info, nil
	}
	// if error was not found, means account doesnt exist
	// and we can create one
	if err != nil && status.Code(err) != codes.NotFound {
		require.NoError(ctx.T, err)
		return nil, err
	}

	var defaultSendAmt sdk.Coins = []sdk.Coin{
		{
			Denom:  appparams.DefaultBondDenom,
			Amount: math.NewInt(10000),
		},
	}

	msg := banktypes.MsgSend{
		FromAddress: e.Network.GetValidatorAddr(),
		ToAddress:   actor.SourceHubAddr,
		Amount:      defaultSendAmt,
	}
	tx, err := e.txBuilder.BuildFromMsgs(ctx,
		hubsdk.TxSignerFromCosmosKey(e.Network.GetValidatorKey()),
		&msg,
	)
	require.NoError(ctx.T, err)

	_, err = client.BroadcastTx(ctx, tx)
	require.NoError(ctx.T, err)
	e.Network.Network.WaitForNextBlock()

	resp, err = client.AuthQueryClient().AccountInfo(ctx, &authtypes.QueryAccountInfoRequest{
		Address: actor.SourceHubAddr,
	})
	require.NoError(ctx.T, err)
	return resp.Info, nil
}

func (e *SDKClientExecutor) broadcastTx(ctx *TestCtx, msgSet *hubsdk.MsgSet) *hubsdk.TxExecResult {
	_, err := e.GetOrCreateAccountFromActor(ctx, ctx.TxSigner)
	require.NoError(ctx.T, err)

	signer := hubsdk.TxSignerFromCosmosKey(ctx.TxSigner.PrivKey)

	tx, err := e.txBuilder.Build(ctx, signer, msgSet)
	require.NoError(ctx.T, err)

	response, err := e.Network.Client.BroadcastTx(ctx, tx)
	require.NoError(ctx.T, err)

	e.Network.Network.WaitForNextBlock()
	result, err := e.Network.Client.GetTx(ctx, response.TxHash)
	require.NoError(ctx.T, err)

	return result
}

func (e *SDKClientExecutor) Cleanup() {
	e.Network.TearDown()
}
