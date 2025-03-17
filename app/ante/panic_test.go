package ante_test

import (
	"fmt"
	"testing"

	"github.com/sourcenetwork/sourcehub/app/ante"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	appparams "github.com/sourcenetwork/sourcehub/app/params"
	test "github.com/sourcenetwork/sourcehub/testutil"
	"github.com/sourcenetwork/sourcehub/testutil/sample"
	"github.com/stretchr/testify/require"
)

type mockPanicDecorator struct{}

func (d mockPanicDecorator) AnteHandle(_ sdk.Context, _ sdk.Tx, _ bool, _ sdk.AnteHandler) (newCtx sdk.Context, err error) {
	panic("panic")
}

func TestPanicHandlerDecorator(t *testing.T) {
	decorator := ante.NewHandlePanicDecorator()
	anteHandler := sdk.ChainAnteDecorators(decorator, mockPanicDecorator{})
	encCfg := test.CreateTestEncodingConfig()
	builder := encCfg.TxConfig.NewTxBuilder()

	err := builder.SetMsgs(banktypes.NewMsgSend(sample.RandomAccAddress(), sample.RandomAccAddress(), sdk.NewCoins(sdk.NewInt64Coin(appparams.DefaultBondDenom, 100))))
	require.NoError(t, err)
	tx := builder.GetTx()

	defer func() {
		r := recover()
		require.NotNil(t, r)
		require.Equal(t, fmt.Sprint("panic", ante.FormatTx(tx)), r)
	}()

	_, _ = anteHandler(sdk.Context{}, tx, false)
}
