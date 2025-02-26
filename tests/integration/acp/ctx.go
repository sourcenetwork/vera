package test

import (
	"context"
	"fmt"
	"testing"
	"time"

	prototypes "github.com/cosmos/gogoproto/types"
	"github.com/stretchr/testify/require"

	"github.com/sourcenetwork/sourcehub/x/acp/testutil"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
	"github.com/sourcenetwork/sourcehub/x/acp/utils"
)

var DefaultTs = MustDateTimeToProto("2024-01-01 00:00:00")

var _ context.Context = (*TestCtx)(nil)

type TestState struct {
	PolicyId      string
	PolicyCreator string
	// Txs is a list of bytes which contains the Txs that have been broadcast during a test
	Txs               [][]byte
	TokenIssueTs      time.Time
	TokenIssueProtoTs *prototypes.Timestamp
}

func (s *TestState) PushTx(tx []byte) {
	s.Txs = append(s.Txs, tx)
}

func (s *TestState) GetLastTx() []byte {
	if len(s.Txs) == 0 {
		return nil
	}
	return s.Txs[len(s.Txs)-1]
}

func NewTestCtxFromConfig(t *testing.T, config TestConfig) *TestCtx {
	params := types.Params{
		PolicyCommandMaxExpirationDelta: 1,
		RegistrationsCommitmentValidity: types.NewBlockCountDuration(7),
	}
	executor := NewACPClient(t, config.ExecutorStrategy, params)

	root := MustNewSourceHubActorFromName("root")
	ctx := &TestCtx{
		Ctx:       context.TODO(),
		T:         t,
		TxSigner:  root,
		Executor:  executor,
		Strategy:  config.AuthStrategy,
		ActorType: config.ActorType,
		Params:    params,
	}

	_, err := executor.GetOrCreateAccountFromActor(ctx, root)
	require.NoError(t, err)

	return ctx
}

type TestCtx struct {
	Ctx   context.Context
	T     *testing.T
	State TestState
	// Signer for Txs while running tests under Bearer or Signed Auth modes
	TxSigner      *TestActor
	Executor      ACPClient
	Strategy      AuthenticationStrategy
	AccountKeeper *testutil.AccountKeeperStub
	ActorType     ActorKeyType
	Params        types.Params
}

func NewTestCtx(t *testing.T) *TestCtx {
	initTest()
	config := MustNewTestConfigFromEnv()
	ctx := NewTestCtxFromConfig(t, config)
	return ctx
}

func (c *TestCtx) Deadline() (deadline time.Time, ok bool) { return c.Ctx.Deadline() }
func (c *TestCtx) Done() <-chan struct{}                   { return c.Ctx.Done() }
func (c *TestCtx) Err() error                              { return c.Ctx.Err() }
func (c *TestCtx) Value(key any) any                       { return c.Ctx.Value(key) }

// GetActor gets or create an account with the given alias
func (c *TestCtx) GetActor(alias string) *TestActor {
	switch c.ActorType {
	case Actor_ED25519:
		return MustNewED25519ActorFromName(alias)
	case Actor_SECP256K1:
		acc := MustNewSourceHubActorFromName(alias)
		_, err := c.Executor.GetOrCreateAccountFromActor(c, acc)
		require.NoError(c.T, err)
		return acc
	default:
		panic(fmt.Sprintf("invalid actor type: %v", c.ActorType))
	}
}

func (c *TestCtx) GetSourceHubAccount(alias string) *TestActor {
	acc := MustNewSourceHubActorFromName(alias)
	c.AccountKeeper.NewAccount(acc.PubKey)
	return acc
}

// GetRecordMetadataForActor, fetches actor from the actor registry
// and builds a RecordMetadata object for the recovered DID
func (c *TestCtx) GetRecordMetadataForActor(actor string) *types.RecordMetadata {
	return &types.RecordMetadata{
		CreationTs: c.GetBlockTs(),
		TxHash:     utils.HashTx(c.State.GetLastTx()),
		TxSigner:   c.TxSigner.SourceHubAddr,
		OwnerDid:   c.GetActor(actor).DID,
	}
}

// GetSignerRecordMetadata builds RecordMetadata for the current
// Tx Signer
func (c *TestCtx) GetSignerRecordMetadata() *types.RecordMetadata {
	return &types.RecordMetadata{
		CreationTs: c.GetBlockTs(),
		TxHash:     utils.HashTx(c.State.GetLastTx()),
		TxSigner:   c.TxSigner.SourceHubAddr,
		OwnerDid:   c.TxSigner.DID,
	}
}

func (c *TestCtx) GetParams() types.Params {
	return c.Params
}

func (c *TestCtx) Cleanup() {
	c.Executor.Cleanup()
}

// WaitBlock waits until the underlying SourceHub node advances to the next block
func (c *TestCtx) WaitBlock() {
	c.Executor.WaitBlock()
}

// WaitBlock waits until the underlying SourceHub node advances to the next block
func (c *TestCtx) WaitBlocks(n uint64) {
	for i := uint64(0); i < n; i += 1 {
		c.Executor.WaitBlock()
	}
}

// GetBlockTs returns the timestamp of the last processed block
func (c *TestCtx) GetBlockTs() *types.Timestamp {
	ts, err := c.Executor.GetLastBlockTs(c)
	require.NoError(c.T, err)
	return ts
}
