package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sourcenetwork/acp_core/pkg/errors"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/sourcenetwork/sourcehub/x/acp/utils"
)

func BuildRecordMetadata(ctx sdk.Context, actorDID string, msgCreator string) (*RecordMetadata, error) {
	ts, err := TimestampFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	dt := &RecordMetadata{
		CreationTs: ts,
		TxHash:     utils.HashTx(ctx.TxBytes()),
		OwnerDid:   actorDID,
		TxSigner:   msgCreator,
	}
	return dt, nil
}

func BuildACPSuppliedMetadata(ctx sdk.Context, actorDID string, msgCreator string) (*coretypes.SuppliedMetadata, error) {
	rm, err := BuildRecordMetadata(ctx, actorDID, msgCreator)
	if err != nil {
		return nil, err
	}
	bytes, err := rm.Marshal()
	if err != nil {
		return nil, err
	}
	return &coretypes.SuppliedMetadata{
		Blob: bytes,
	}, nil
}

func UmarshalRecordMetadata(md *coretypes.RecordMetadata) (*RecordMetadata, error) {
	metadata := &RecordMetadata{}
	err := metadata.Unmarshal(md.Supplied.Blob)
	if err != nil {
		return nil, errors.Wrap("unmarshaling record metadata", err)
	}
	return metadata, nil
}

func MapRelationshipRecord(rec *coretypes.RelationshipRecord) (*RelationshipRecord, error) {
	metadata := &RecordMetadata{}
	err := metadata.Unmarshal(rec.Metadata.Supplied.Blob)
	if err != nil {
		return nil, errors.Wrap("unmarshaling record metadata", err)
	}

	return &RelationshipRecord{
		PolicyId:     rec.PolicyId,
		Archived:     rec.Archived,
		Relationship: rec.Relationship,
		Metadata:     metadata,
	}, nil
}

func MapPolicy(rec *coretypes.PolicyRecord) (*PolicyRecord, error) {
	metadata := &RecordMetadata{}
	err := metadata.Unmarshal(rec.Metadata.Supplied.Blob)
	if err != nil {
		return nil, errors.Wrap("unmarshaling record metadata", err)
	}

	return &PolicyRecord{
		Policy:      rec.Policy,
		Metadata:    metadata,
		MarshalType: rec.MarshalType,
		RawPolicy:   rec.PolicyDefinition,
	}, nil
}
