// package registration exposes types for the object registration protocol
package registration

import (
	"context"

	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

type RegistrationsRepository interface {
	Set(ctx context.Context, reg *types.RegistrationsCommitment) error

	GetById(ctx context.Context, id string) (*types.RegistrationsCommitment, error)

	FilterByCommitment(ctx context.Context, commitment []byte) ([]*types.RegistrationsCommitment, error)
}

type RegistrationCounter interface {
	GetNextId(context.Context) (uint64, error)
}
