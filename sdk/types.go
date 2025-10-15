package sdk

import (
	"fmt"

	rpctypes "github.com/cometbft/cometbft/rpc/core/types"
)

func newTxExecResult(result *rpctypes.ResultTx) *TxExecResult {
	var err error
	if result.TxResult.Code != 0 {
		err = fmt.Errorf("%v: %v", ErrTxFailed, result.TxResult.Log)
	}

	return &TxExecResult{
		result: result,
		err:    err,
	}
}

// TxExecResult models the outcome of a Tx evaluated by SourceHub.
// The Tx was either successfuly included or an error happened while handling one of its Msgs.
type TxExecResult struct {
	result *rpctypes.ResultTx
	err    error
}

// Error returns whether the error message if the Tx execution failed
func (r *TxExecResult) Error() error {
	return r.err
}

// TxPayload returns the payload of the executed Tx
func (r *TxExecResult) TxPayload() *rpctypes.ResultTx {
	return r.result
}

func newListenResult(result *TxExecResult, err error) *ListenResult {
	return &ListenResult{
		result: result,
		err:    err,
	}
}

// ListenResult represents the result of waiting for a Tx to be executed by SourceHub
// Can either error, meaning timeout or network error, or return the Tx object queried from the chain state
type ListenResult struct {
	result *TxExecResult
	err    error
}

// Error return the error of the WaitResult, returns nil if sucessful
func (r *ListenResult) Error() error { return r.err }

// GetTxResult returns the outcome of the executed Tx by SourceHub
func (r *ListenResult) GetTxResult() *TxExecResult { return r.result }
