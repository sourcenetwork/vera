package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	abcitypes "github.com/cometbft/cometbft/abci/types"
	cometclient "github.com/cometbft/cometbft/rpc/client"
	rpctypes "github.com/cometbft/cometbft/rpc/core/types"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	acptypes "github.com/sourcenetwork/sourcehub/x/acp/types"
)

type TxListener struct {
	rpc       cometclient.Client
	cleanupFn func()
}

func NewTxListener(client cometclient.Client) TxListener {
	return TxListener{
		rpc: client,
	}
}

type Thing struct {
	Height    int64     `json:"height"`
	Index     uint32    `json:"index"`
	Tx        []byte    `json:"tx"`
	Code      uint32    `json:"code"`
	Log       string    `json:"log"`
	Info      string    `json:"info"`
	GasWanted int64     `json:"gas_wanted"`
	GasUsed   int64     `json:"gas_used"`
	Codespace string    `json:"codespace"`
	Responses []sdk.Msg `json:"responses"`
}

func (l *TxListener) ListenTxs(ctx context.Context) (<-chan Thing, <-chan error, error) {
	ch, err := l.rpc.Subscribe(ctx, "", "tm.event='Tx'")
	if err != nil {
		return nil, nil, fmt.Errorf("TxListener: subscribing to Tx event: %w", err)
	}

	mapper := func(in rpctypes.ResultEvent) (Thing, error) {
		resultBytes, err := json.Marshal(in.Data)
		if err != nil {
			return Thing{}, fmt.Errorf("marshaling result data to json: %v", err)
		}

		txResult := &abcitypes.TxResult{}
		err = json.Unmarshal(resultBytes, txResult)
		if err != nil {
			return Thing{}, fmt.Errorf("unmarshaling into TxResult: %w", err)
		}

		msgData := sdk.TxMsgData{}
		err = msgData.Unmarshal(txResult.Result.Data)
		if err != nil {
			return Thing{}, fmt.Errorf("unmarshaling TxResult.ExecResultTx.Data into TxMsgData: %v", err)
		}

		registry := cdctypes.NewInterfaceRegistry()
		acptypes.RegisterInterfaces(registry)
		responses := make([]sdk.Msg, 0, len(msgData.MsgResponses))
		for i, resp := range msgData.MsgResponses {
			var msg sdk.Msg
			err := registry.UnpackAny(resp, &msg)
			if err != nil {
				return Thing{}, fmt.Errorf("unmarshaling response %v: %w", i, err)
			}
			responses = append(responses, msg)
		}
		return Thing{
			Height:    txResult.Height,
			Index:     txResult.Index,
			Tx:        txResult.Tx,
			Code:      txResult.Result.Code,
			Log:       txResult.Result.Log,
			Info:      txResult.Result.Info,
			GasWanted: txResult.Result.GasWanted,
			GasUsed:   txResult.Result.GasUsed,
			Codespace: txResult.Result.Codespace,
			Responses: responses,
		}, nil
	}

	resultCh, errChn, closeFn := channelMapper(ch, mapper)
	l.cleanupFn = closeFn
	return resultCh, errChn, err
}

func (l *TxListener) Done() <-chan struct{} {
	return l.rpc.Quit()
}

func (l *TxListener) Stop() {
	l.rpc.Stop()
	l.cleanupFn()
}

func channelMapper[T, U any](ch <-chan T, mapper func(T) (U, error)) (<-chan U, <-chan error, func()) {
	errCh := make(chan error, 100)
	newCh := make(chan U, 100)
	closeFn := func() {
		close(errCh)
		close(newCh)
	}
	go func() {
		for {
			select {
			case result, ok := <-ch:
				log.Printf("received result")
				if !ok {
					close(errCh)
					close(newCh)
					return
				}

				u, err := mapper(result)
				if err != nil {
					errCh <- err
				} else {
					newCh <- u
				}
			}
		}
	}()
	return newCh, errCh, closeFn
}
