package cosmos

import (
	"fmt"

	"github.com/sourcenetwork/raccoondb/v2/errors"
)

var ErrCosmosKV = errors.New("cosmossdk store")

// wrapErr wraps an error with ErrCometDb
func wrapErr(err error) error {
	return fmt.Errorf("%w: %w", ErrCosmosKV, err)
}
