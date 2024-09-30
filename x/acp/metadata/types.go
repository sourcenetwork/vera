package metadata

import (
	"context"

	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

type MetadataRepository interface {
	SetRegistrationMetadata(context.Context, string, *coretypes.Object, *types.RegistrationMetadata) error
	GetRegistrationMetadata(context.Context, string, *coretypes.Object) (*types.RegistrationMetadata, error)
}
