package metadata

import (
	"context"

	"github.com/sourcenetwork/sourcehub/api/sourcenetwork/acp_core"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

type MetadataRepository interface {
	SetRelationshipMetadata(context.Context, *acp_core.Relationship, *types.RelationshipMetadata) error
	GetRelationshipMetadata(context.Context, *acp_core.Relationship) (*types.RelationshipMetadata, error)
}
