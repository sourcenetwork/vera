package keeper

import (
	"testing"

	prototypes "github.com/cosmos/gogoproto/types"
	"github.com/stretchr/testify/require"

	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

func TestEndBlocker(t *testing.T) {
	ctx, k, _ := setupKeeper(t)

	repo := k.GetRegistrationsCommitmentRepository(ctx)
	_ = repo
	comm := &types.RegistrationsCommitment{
		Id:       1,
		PolicyId: "abc",
		Metadata: &types.RecordMetadata{
			OwnerDid: "abc",
			TxHash:   []byte("00"),
			TxSigner: "source1235",
			CreationTs: &types.Timestamp{
				ProtoTs: &prototypes.Timestamp{
					Seconds: 0,
					Nanos:   0,
				},
				BlockHeight: 1,
			},
		},
		Commitment: []byte("0"),
		Expired:    false,
		Validity: &types.Duration{
			Duration: &types.Duration_ProtoDuration{
				ProtoDuration: &prototypes.Duration{
					Nanos: 1,
				},
			},
		},
	}

	// FIXME
	//err := repo.Set(ctx, comm)
	//require.NoError(t, err)

	expired, err := k.EndBlocker(ctx)
	require.NoError(t, err)
	require.Len(t, expired, 1)
	require.Equal(t, comm, expired[0])
}
