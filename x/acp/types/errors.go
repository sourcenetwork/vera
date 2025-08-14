package types

import (
	errorsmod "cosmossdk.io/errors"
	"github.com/sourcenetwork/acp_core/pkg/errors"
)

// x/acp module sentinel errors
var (
	ErrInvalidSigner = errorsmod.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
)

type Error = errors.Error
type ErrorType = errors.ErrorType

const (
	ErrorType_UNKNOWN             = errors.ErrorType_UNKNOWN
	ErrorType_INTERNAL            = errors.ErrorType_INTERNAL
	ErrorType_UNAUTHENTICATED     = errors.ErrorType_UNAUTHENTICATED
	ErrorType_UNAUTHORIZED        = errors.ErrorType_UNAUTHORIZED
	ErrorType_BAD_INPUT           = errors.ErrorType_BAD_INPUT
	ErrorType_OPERATION_FORBIDDEN = errors.ErrorType_OPERATION_FORBIDDEN
	ErrorType_NOT_FOUND           = errors.ErrorType_NOT_FOUND
)

var New = errors.New
var Wrap = errors.Wrap
var NewFromBaseError = errors.NewWithCause
var NewFromCause = errors.NewWithCause

func NewErrInvalidAccAddrErr(cause error, addr string) error {
	return errors.NewWithCause("invalid account address", cause, errors.ErrorType_BAD_INPUT, errors.Pair("address", addr))
}

func NewAccNotFoundErr(addr string) error {
	return errors.Wrap("account not found", errors.ErrorType_NOT_FOUND, errors.Pair("address", addr))
}
