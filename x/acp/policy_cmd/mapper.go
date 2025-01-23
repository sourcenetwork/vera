package policy_cmd

import (
	"github.com/sourcenetwork/acp_core/pkg/errors"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

func MapRelationshipRecord(rec *coretypes.RelationshipRecord) (*types.RelationshipRecord, error) {
	metadata := &types.RecordMetadata{}
	err := metadata.Unmarshal(rec.Metadata.Supplied.Blob)
	if err != nil {
		return nil, errors.Wrap("unmarshaling record metadata", err)
	}

	return &types.RelationshipRecord{
		PolicyId:     rec.PolicyId,
		Archived:     rec.Archived,
		Relationship: rec.Relationship,
		Metadata:     metadata,
	}, nil
}
