package ante

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// HandlePanicDecorator catches and wraps panics in the transaction that caused them.
type HandlePanicDecorator struct{}

func NewHandlePanicDecorator() HandlePanicDecorator {
	return HandlePanicDecorator{}
}

func (d HandlePanicDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	defer func() {
		if r := recover(); r != nil {
			panic(fmt.Sprint(r, FormatTx(tx)))
		}
	}()

	return next(ctx, tx, simulate)
}

func FormatTx(tx sdk.Tx) string {
	output := "\ntransaction failed:\n"
	for _, msg := range tx.GetMsgs() {
		output += fmt.Sprintf("%T{%s}\n", msg, msg)
	}

	return output
}
