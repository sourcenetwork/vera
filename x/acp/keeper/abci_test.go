package keeper

import (
	"testing"

	prototypes "github.com/cosmos/gogoproto/types"
	"github.com/stretchr/testify/require"

	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

func TestEndBlocker(t *testing.T) {
	// TODO
	ctx, k, _ := setupKeeper(t)

	repo := k.GetRegistrationsCommitmentRepository(ctx)
	err := repo.Set(ctx, &types.RegistrationsCommitment{
		Id:         "1",
		PolicyId:   "abc",
		Actor:      coretypes.NewActor("abc"),
		Commitment: []byte("0"),
		Expired:    false,
		TxHash:     []byte("00"),
		Validity: &types.Duration{
			Duration: &types.Duration_ProtoDuration{
				ProtoDuration: &prototypes.Duration{
					Nanos: 1,
				},
			},
		},
		CreationTs: &types.Timestamp{
			ProtoTs: &prototypes.Timestamp{
				Seconds: 0,
				Nanos:   0,
			},
			BlockHeight: 1,
		},
	})
	require.NoError(t, err)

	_, err = k.EndBlocker(ctx)
	require.NoError(t, err)
}
