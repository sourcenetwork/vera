package sdk

import (
	"fmt"

	comettypes "github.com/cometbft/cometbft/rpc/core/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
)

// Mapper processes a CometBFT Tx Event from a specific Tx and maps into a payload.
// Mapper is bound to a msgIdx within the Tx
type Mapper[T sdk.Msg] struct {
	msgIdx    int
	container T
}

func newMapper[T sdk.Msg](idx int, container T) Mapper[T] {
	return Mapper[T]{
		msgIdx:    idx,
		container: container,
	}
}

// Map receives a CometBFT Tx Event Result and maps it into a T or errors if invalid
func (e *Mapper[T]) Map(msg *comettypes.ResultTx) (T, error) {
	var zero T

	// pretty sure this is an sdk response
	data := msg.TxResult.Data
	msgData := sdk.TxMsgData{}
	err := msgData.Unmarshal(data)
	if err != nil {
		return zero, err
	}

	if len(msgData.MsgResponses) < e.msgIdx+1 {
		return zero, fmt.Errorf("invalid mapper: tx does not contain %v msgs", e.msgIdx+1)
	}

	any := msgData.MsgResponses[e.msgIdx]
	err = proto.Unmarshal(any.Value, e.container)
	if err != nil {
		return zero, err
	}

	return e.container, nil
}
