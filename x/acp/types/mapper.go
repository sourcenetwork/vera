package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sourcenetwork/acp_core/pkg/errors"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/sourcenetwork/sourcehub/x/acp/utils"
)

// BuildRecordMetadata returns a RecordMetadata from an sdk Context
// and authenticated actor / signer data.
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

// BuildACPSuppliedMetadata returns an acp SuppliedMetadata
// object which embeds a SourceHub RecordMetadata
// built from ctx, actorDID and msgCreator
func BuildACPSuppliedMetadata(ctx sdk.Context, actorDID string, msgCreator string) (*coretypes.SuppliedMetadata, error) {
	metadata, err := BuildRecordMetadata(ctx, actorDID, msgCreator)
	if err != nil {
		return nil, err
	}
	bytes, err := metadata.Marshal()
	if err != nil {
		return nil, err
	}
	return &coretypes.SuppliedMetadata{
		Blob: bytes,
	}, nil
}

// BuildACPSuppliedMetadata returns an acp SuppliedMetadata
// object which embeds a SourceHub RecordMetadata
// built from ctx, actorDID and msgCreator
func BuildACPSuppliedMetadataWithTime(ctx sdk.Context, ts *Timestamp, actorDID string, msgCreator string) (*coretypes.SuppliedMetadata, error) {
	metadata := &RecordMetadata{
		CreationTs: ts,
		TxHash:     utils.HashTx(ctx.TxBytes()),
		OwnerDid:   actorDID,
		TxSigner:   msgCreator,
	}
	bytes, err := metadata.Marshal()
	if err != nil {
		return nil, err
	}
	return &coretypes.SuppliedMetadata{
		Blob: bytes,
	}, nil
}

// ExtractRecordMetadata extracts and unmarshals
// a RecordMetadata from the blob field in acp_core's metadata
func ExtractRecordMetadata(md *coretypes.RecordMetadata) (*RecordMetadata, error) {
	metadata := &RecordMetadata{}
	err := metadata.Unmarshal(md.Supplied.Blob)
	if err != nil {
		return nil, errors.Wrap("unmarshaling record metadata", err)
	}
	return metadata, nil
}

// MapRelationshipRecord maps an acp_core RelationshipRecord
// into a SourceHub RelationshpRecord
func MapRelationshipRecord(rec *coretypes.RelationshipRecord) (*RelationshipRecord, error) {
	metadata, err := ExtractRecordMetadata(rec.Metadata)
	if err != nil {
		return nil, err
	}

	return &RelationshipRecord{
		PolicyId:     rec.PolicyId,
		Archived:     rec.Archived,
		Relationship: rec.Relationship,
		Metadata:     metadata,
	}, nil
}

// MapPolicy maps an acp core PolicyRecord into a SourceHub
// PolicyRecord
func MapPolicy(rec *coretypes.PolicyRecord) (*PolicyRecord, error) {
	metadata, err := ExtractRecordMetadata(rec.Metadata)
	if err != nil {
		return nil, err
	}

	return &PolicyRecord{
		Policy:      rec.Policy,
		Metadata:    metadata,
		MarshalType: rec.MarshalType,
		RawPolicy:   rec.PolicyDefinition,
	}, nil
}
