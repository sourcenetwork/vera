package lanes

import (
	"fmt"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	appparams "github.com/sourcenetwork/sourcehub/app/params"
	acptypes "github.com/sourcenetwork/sourcehub/x/acp/types"
	bulletintypes "github.com/sourcenetwork/sourcehub/x/bulletin/types"
	tiertypes "github.com/sourcenetwork/sourcehub/x/tier/types"
)

type mockTx struct {
	msgs     []sdk.Msg
	fee      sdk.Coins
	gasLimit uint64
	nonce    uint64
}

func (tx mockTx) GetMsgs() []sdk.Msg { return tx.msgs }
func (tx mockTx) GetMsgsV2() ([]proto.Message, error) {
	protoMsgs := make([]proto.Message, 0, len(tx.msgs))
	for _, msg := range tx.msgs {
		pMsg, ok := msg.(proto.Message)
		if !ok {
			return nil, fmt.Errorf("message %T does not implement proto.Message", msg)
		}
		protoMsgs = append(protoMsgs, pMsg)
	}
	return protoMsgs, nil
}
func (tx mockTx) ValidateBasic() error { return nil }
func (tx mockTx) GetGas() uint64       { return tx.gasLimit }
func (tx mockTx) GetFee() sdk.Coins    { return tx.fee }
func (tx mockTx) FeePayer() []byte     { return []byte("") }
func (tx mockTx) FeeGranter() []byte   { return []byte("") }

var _ sdk.FeeTx = mockTx{}

func setupTest(t testing.TB) sdk.Context {
	storeKey := storetypes.NewKVStoreKey(acptypes.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeDB, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())

	return ctx
}

func TestTxPriority(t *testing.T) {
	ctx := setupTest(t)

	acpMsg := &acptypes.MsgBearerPolicyCmd{}
	tx1 := mockTx{
		msgs:     []sdk.Msg{acpMsg},
		fee:      sdk.NewCoins(sdk.NewInt64Coin(appparams.DefaultBondDenom, 5000)),
		gasLimit: 10000,
	}

	tierMsg := &tiertypes.MsgLock{}
	tx2 := mockTx{
		msgs:     []sdk.Msg{tierMsg},
		fee:      sdk.NewCoins(sdk.NewInt64Coin(appparams.DefaultBondDenom, 10000)),
		gasLimit: 10000,
	}

	bulletinMsg := &bulletintypes.MsgCreatePost{}
	tx3 := mockTx{
		msgs:     []sdk.Msg{bulletinMsg},
		fee:      sdk.NewCoins(sdk.NewInt64Coin(appparams.DefaultBondDenom, 20000)),
		gasLimit: 10000,
	}

	stakingMsg := &stakingtypes.MsgDelegate{}
	tx4 := mockTx{
		msgs:     []sdk.Msg{stakingMsg},
		fee:      sdk.NewCoins(sdk.NewInt64Coin(appparams.DefaultBondDenom, 30000)),
		gasLimit: 10000,
	}

	priority1 := TxPriority().GetTxPriority(ctx, tx1)
	priority2 := TxPriority().GetTxPriority(ctx, tx2)
	priority3 := TxPriority().GetTxPriority(ctx, tx3)
	priority4 := TxPriority().GetTxPriority(ctx, tx4)

	require.True(t, priority1 > priority2, "Acp transaction should have higher priority than Tier transaction")
	require.True(t, priority2 > priority3, "Tier transaction should have higher priority than Bulletin transaction")
	require.True(t, priority3 > priority4, "Bulletin transaction should have higher priority than any other transaction")
}

func TestGasPriceSorting(t *testing.T) {
	ctx := setupTest(t)

	tx1 := mockTx{
		msgs:     []sdk.Msg{&acptypes.MsgCreatePolicy{}},
		fee:      sdk.NewCoins(sdk.NewInt64Coin(appparams.DefaultBondDenom, 50000)), // 50,000 uopen
		gasLimit: 50000,                                                             // gas price = 50,000 / 50,000 = 1.0 uopen
	}

	tx2 := mockTx{
		msgs:     []sdk.Msg{&acptypes.MsgCreatePolicy{}},
		fee:      sdk.NewCoins(sdk.NewInt64Coin(appparams.DefaultBondDenom, 30000)), // 30,000 uopen
		gasLimit: 10000,                                                             // gas price = 30,000 / 10,000 = 3.0 uopen
	}

	tx3 := mockTx{
		msgs:     []sdk.Msg{&acptypes.MsgCreatePolicy{}},
		fee:      sdk.NewCoins(sdk.NewInt64Coin(appparams.DefaultBondDenom, 10000)), // 10,000 uopen
		gasLimit: 20000,                                                             // gas price = 10,000 / 20,000 = 0.5 uopen
	}

	priority1 := TxPriority().GetTxPriority(ctx, tx1) // 1.0 uopen
	priority2 := TxPriority().GetTxPriority(ctx, tx2) // 3.0 uopen
	priority3 := TxPriority().GetTxPriority(ctx, tx3) // 0.5 uopen

	require.True(t, priority2 > priority1, "Tx with 3.0 gas price should have higher priority than tx with 1.0 gas price")
	require.True(t, priority1 > priority3, "Tx with 1.0 gas price should have higher priority than tx with 0.5 gas price")
}
