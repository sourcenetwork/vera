package test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/sourcenetwork/sourcehub/x/acp/signed_policy_cmd"
	"github.com/sourcenetwork/sourcehub/x/acp/testutil"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

var DefaultTs = MustDateTimeToProto("2024-01-01 00:00:00")

var _ context.Context = (*TestCtx)(nil)

type TestState struct {
	PolicyId      string
	PolicyCreator string
}

func NewTestCtxFromConfig(t *testing.T, config TestConfig) *TestCtx {
	executor := NewExecutor(t, config.ExecutorStrategy)

	root := MustNewSourceHubActorFromName("root")
	ctx := &TestCtx{
		Ctx:          context.TODO(),
		T:            t,
		TxSigner:     root,
		Timestamp:    time.Now(),
		TokenIssueTs: time.Now(),
		Executor:     executor,
		Strategy:     config.AuthStrategy,
		ActorType:    config.ActorType,
		LogicalClock: &logicalClockImpl{},
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
	TxSigner *TestActor
	// Timestamp used to generate Msgs in Test
	Timestamp     time.Time
	TokenIssueTs  time.Time
	Executor      MsgExecutor
	Strategy      AuthenticationStrategy
	AccountKeeper *testutil.AccountKeeperStub
	ActorType     ActorKeyType
	LogicalClock  signed_policy_cmd.LogicalClock
	TxHash        string
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

func (c *TestCtx) GetParams() types.Params {
	return types.NewParams()
}

func (c *TestCtx) Cleanup() {
	c.Executor.Cleanup()
}

// WaitBlock waits until the underlying SourceHub node advances to the next block
func (c *TestCtx) WaitBlock() {
	c.Executor.WaitBlock()
}
