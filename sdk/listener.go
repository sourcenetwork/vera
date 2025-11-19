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
	bulletintypes "github.com/sourcenetwork/sourcehub/x/bulletin/types"
	tiertypes "github.com/sourcenetwork/sourcehub/x/tier/types"
)

const mapperBuffSize int = 100

// TxListener is a client which subscribes to Tx events in SourceHub's cometbft socket
// and parses the received events into version with unmarshaled Msg Responses.
type TxListener struct {
	rpc       cometclient.Client
	cleanupFn func()
}

// NewTxListener creates a new listenver from a comet client
func NewTxListener(client cometclient.Client) TxListener {
	return TxListener{
		rpc: client,
	}
}

// Event models a Cometbft Tx event with unmarsheled Msg responses
type Event struct {
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

// ListenTxs spawns a go routine which continously listens for Tx events from a cometbft connection.
// The received events are returned into the Events channel, all errors are sent to the errors channel.
//
// If ListenTxs fails to connect to the comet client, returns an error and nil channels.
func (l *TxListener) ListenTxs(ctx context.Context) (<-chan Event, <-chan error, error) {
	ch, err := l.rpc.Subscribe(ctx, "", "tm.event='Tx'")
	if err != nil {
		return nil, nil, fmt.Errorf("TxListener: subscribing to Tx event: %w", err)
	}

	registry := cdctypes.NewInterfaceRegistry()
	acptypes.RegisterInterfaces(registry)
	bulletintypes.RegisterInterfaces(registry)
	tiertypes.RegisterInterfaces(registry)

	mapper := func(in rpctypes.ResultEvent) (Event, error) {
		resultBytes, err := json.Marshal(in.Data)
		if err != nil {
			return Event{}, fmt.Errorf("marshaling result data to json: %v", err)
		}

		txResult := &abcitypes.TxResult{}
		err = json.Unmarshal(resultBytes, txResult)
		if err != nil {
			return Event{}, fmt.Errorf("unmarshaling into TxResult: %w", err)
		}

		msgData := sdk.TxMsgData{}
		err = msgData.Unmarshal(txResult.Result.Data)
		if err != nil {
			return Event{}, fmt.Errorf("unmarshaling TxResult.ExecResultTx.Data into TxMsgData: %v", err)
		}

		//unmarshals the msg results into their
		// actual response values using the registered sourcehub types
		responses := make([]sdk.Msg, 0, len(msgData.MsgResponses))
		for i, resp := range msgData.MsgResponses {
			var msg sdk.Msg
			err := registry.UnpackAny(resp, &msg)
			if err != nil {
				return Event{}, fmt.Errorf("unmarshaling response %v: %w", i, err)
			}
			responses = append(responses, msg)
		}

		return Event{
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

// ListenAsync spawns a go routine and listens for txs asyncrhonously,
// until the comet client closes the connection, the context is cancelled.
// or the listener is closed.
// Callback is called each time an event or an error is received
// Returns an error if connection to commet fails
func (l *TxListener) ListenAsync(ctx context.Context, cb func(*Event, error)) error {
	evs, errs, err := l.ListenTxs(ctx)
	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case result := <-evs:
				cb(&result, nil)
			case err := <-errs:
				cb(nil, err)
			case <-l.Done():
				log.Printf("Listener closed: canceling loop")
				break
			case <-ctx.Done():
				break
			}
		}
	}()
	return nil
}

// Done returns a channel which will be closed when the connection fails
func (l *TxListener) Done() <-chan struct{} {
	return l.rpc.Quit()
}

// Close stops listening for events and cleans up
func (l *TxListener) Close() {
	l.rpc.Stop()
	l.cleanupFn()
}

// channelMapper wraps a channel and applies a failable mapper to all incoming items.
// Returns a value channel, an error channel and a callback to terminate the channel
func channelMapper[T, U any](ch <-chan T, mapper func(T) (U, error)) (values <-chan U, errors <-chan error, closeFn func()) {
	errCh := make(chan error, mapperBuffSize)
	valCh := make(chan U, mapperBuffSize)
	closeFn = func() {
		close(errCh)
		close(valCh)
	}
	go func() {
		for {
			select {
			case result, ok := <-ch:
				if !ok {
					close(errCh)
					close(valCh)
					return
				}

				u, err := mapper(result)
				if err != nil {
					errCh <- err
				} else {
					valCh <- u
				}
			}
		}
	}()
	return valCh, errCh, closeFn
}
