package utils

import (
	"crypto/sha256"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sourcenetwork/acp_core/pkg/auth"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"

	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

// HasTx produces a sha256 of a Tx bytes
func HashTx(txBytes []byte) []byte {
	hasher := sha256.New()
	hasher.Write(txBytes)
	return hasher.Sum(nil)
}

// InjectPrincipal injects an acp core  did principal in ctx
// and returns the new context
func InjectPrincipal(ctx sdk.Context, actorDID string) (sdk.Context, error) {
	principal, err := coretypes.NewDIDPrincipal(actorDID)
	if err != nil {
		return sdk.Context{}, err
	}
	goCtx := auth.InjectPrincipal(ctx, principal)
	ctx = ctx.WithContext(goCtx)
	return ctx, nil
}

func BuildRecordMetadata(ctx sdk.Context, actorDID string, msgCreator string) (*types.RecordMetadata, error) {
	ts, err := types.TimestampFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	dt := &types.RecordMetadata{
		CreationTs: ts,
		TxHash:     HashTx(ctx.TxBytes()),
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
