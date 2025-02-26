package test

import (
	"context"
	"crypto/rand"
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
	prototypes "github.com/cosmos/gogoproto/types"
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

type KeeperACPClient struct {
	baseCtx        sdk.Context
	k              types.MsgServer
	querier        types.QueryServer
	accountCreator *testutil.AccountKeeperStub
	ts             types.Timestamp
}

func (c *KeeperACPClient) Cleanup() {}

// WaitBlock bumps the current keeper Ts
func (c *KeeperACPClient) WaitBlock() {
	c.nextBlockTs()
}

// nextBlockTs increments the internal block count
func (c *KeeperACPClient) nextBlockTs() {
	c.ts.BlockHeight++
	c.ts.ProtoTs.Seconds++
}

// genTx generates a random byte slice to model the comet Tx bytes
//
// This is done because the keeper executor doesn't receive an actual
// cometbft signed Tx but this data is used by the code paths
func (c *KeeperACPClient) genTx(ctx *TestCtx) {
	tx := make([]byte, 50)
	_, err := rand.Read(tx)
	require.NoError(ctx.T, err)
	ctx.State.PushTx(tx)
}

// getSDKCtx returns the context which must be used before executing
// calls to the keeper.
// it increments the current timestamp, such that every function call happens with its own block time
func (c *KeeperACPClient) getSDKCtx(ctx context.Context) sdk.Context {
	c.nextBlockTs()
	time, err := prototypes.TimestampFromProto(c.ts.ProtoTs)
	if err != nil {
		panic(err)
	}
	header := cmtproto.Header{
		Time:   time.UTC(),
		Height: int64(c.ts.BlockHeight),
	}
	sdkCtx := c.baseCtx.WithContext(ctx)
	sdkCtx = c.baseCtx.WithBlockHeader(header)
	return sdkCtx
}

func (c *KeeperACPClient) BearerPolicyCmd(ctx *TestCtx, msg *types.MsgBearerPolicyCmd) (*types.MsgBearerPolicyCmdResponse, error) {
	sdkCtx := c.getSDKCtx(ctx)
	return c.k.BearerPolicyCmd(sdkCtx, msg)
}

func (c *KeeperACPClient) SignedPolicyCmd(ctx *TestCtx, msg *types.MsgSignedPolicyCmd) (*types.MsgSignedPolicyCmdResponse, error) {
	sdkCtx := c.getSDKCtx(ctx)
	return c.k.SignedPolicyCmd(sdkCtx, msg)
}

func (c *KeeperACPClient) DirectPolicyCmd(ctx *TestCtx, msg *types.MsgDirectPolicyCmd) (*types.MsgDirectPolicyCmdResponse, error) {
	sdkCtx := c.getSDKCtx(ctx)
	return c.k.DirectPolicyCmd(sdkCtx, msg)
}

func (c *KeeperACPClient) CreatePolicy(ctx *TestCtx, msg *types.MsgCreatePolicy) (*types.MsgCreatePolicyResponse, error) {
	sdkCtx := c.getSDKCtx(ctx)
	return c.k.CreatePolicy(sdkCtx, msg)
}

func (c *KeeperACPClient) GetOrCreateAccountFromActor(_ *TestCtx, actor *TestActor) (sdk.AccountI, error) {
	return c.accountCreator.NewAccount(actor.PubKey), nil
}

func (c *KeeperACPClient) Policy(ctx *TestCtx, msg *types.QueryPolicyRequest) (*types.QueryPolicyResponse, error) {
	sdkCtx := c.getSDKCtx(ctx)
	return c.querier.Policy(sdkCtx, msg)
}

func (c *KeeperACPClient) RegistrationsCommitment(ctx *TestCtx, msg *types.QueryRegistrationsCommitmentRequest) (*types.QueryRegistrationsCommitmentResponse, error) {
	sdkCtx := c.getSDKCtx(ctx)
	return c.querier.RegistrationsCommitment(sdkCtx, msg)
}

func (c *KeeperACPClient) RegistrationsCommitmentByCommitment(ctx *TestCtx, msg *types.QueryRegistrationsCommitmentByCommitmentRequest) (*types.QueryRegistrationsCommitmentByCommitmentResponse, error) {
	sdkCtx := c.getSDKCtx(ctx)
	return c.querier.RegistrationsCommitmentByCommitment(sdkCtx, msg)
}

func (c *KeeperACPClient) ObjectOwner(ctx *TestCtx, msg *types.QueryObjectOwnerRequest) (*types.QueryObjectOwnerResponse, error) {
	sdkCtx := c.getSDKCtx(ctx)
	return c.querier.ObjectOwner(sdkCtx, msg)
}

func (c *KeeperACPClient) GetLastBlockTs(ctx *TestCtx) (*types.Timestamp, error) {
	ts := c.ts
	return &ts, nil
}

func (c *KeeperACPClient) GetTimestampNow(context.Context) (uint64, error) {
	return c.ts.BlockHeight, nil
}

func NewACPClient(t *testing.T, strategy ExecutorStrategy, params types.Params) ACPClient {
	switch strategy {
	case Keeper:
		exec, err := newKeeperExecutor(params)
		require.NoError(t, err)
		return exec
	case SDK:
		network := &e2e.TestNetwork{}
		network.Setup(t)
		executor, err := newSDKExecutor(network)
		require.NoError(t, err)
		return executor
	case CLI:
		panic("sdk executor not implemented")
	default:
		panic(fmt.Sprintf("invalid executor strategy: %v", strategy))
	}
}

func newKeeperExecutor(params types.Params) (ACPClient, error) {
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	err := stateStore.LoadLatestVersion()
	if err != nil {
		return nil, fmt.Errorf("failed to create keeper executor: %v", err)
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
	k.SetParams(ctx, params)

	msgServer := keeper.NewMsgServerImpl(k)
	executor := &KeeperACPClient{
		baseCtx:        ctx,
		k:              msgServer,
		querier:        keeper.NewQuerier(k),
		accountCreator: accKeeper,
		ts: types.Timestamp{
			BlockHeight: 1,
			ProtoTs:     prototypes.TimestampNow(),
		},
	}
	return executor, nil
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

func (e *SDKClientExecutor) Policy(ctx *TestCtx, msg *types.QueryPolicyRequest) (*types.QueryPolicyResponse, error) {
	return e.Network.Client.ACPQueryClient().Policy(ctx, msg)
}

func (e *SDKClientExecutor) RegistrationsCommitment(ctx *TestCtx, msg *types.QueryRegistrationsCommitmentRequest) (*types.QueryRegistrationsCommitmentResponse, error) {
	return e.Network.Client.ACPQueryClient().RegistrationsCommitment(ctx, msg)
}

func (e *SDKClientExecutor) RegistrationsCommitmentByCommitment(ctx *TestCtx, msg *types.QueryRegistrationsCommitmentByCommitmentRequest) (*types.QueryRegistrationsCommitmentByCommitmentResponse, error) {
	return e.Network.Client.ACPQueryClient().RegistrationsCommitmentByCommitment(ctx, msg)
}

func (e *SDKClientExecutor) Cleanup() {
	e.Network.TearDown()
}

func (e *SDKClientExecutor) WaitBlock() {
	panic("not implemented")
}

func (e *SDKClientExecutor) GetLastBlockTs(ctx *TestCtx) (*types.Timestamp, error) {
	panic("not implemented")
}

func (c *SDKClientExecutor) GetTimestampNow(context.Context) (uint64, error) {
	panic("not implemented")
}

func (c *SDKClientExecutor) ObjectOwner(ctx *TestCtx, req *types.QueryObjectOwnerRequest) (*types.QueryObjectOwnerResponse, error) {
	return c.Network.Client.ACPQueryClient().ObjectOwner(ctx, req)
}
