package keeper

import (
	"testing"
	"time"

	prototypes "github.com/cosmos/gogoproto/types"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/stretchr/testify/require"

	"github.com/sourcenetwork/sourcehub/x/acp/commitment"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

func TestEndBlocker(t *testing.T) {
	ctx, k, _ := setupKeeper(t)

	params := &types.Params{
		RegistrationsCommitmentValidity: &types.Duration{
			Duration: &types.Duration_ProtoDuration{
				ProtoDuration: &prototypes.Duration{
					Nanos: 1,
				},
			},
		},
	}

	engine := k.GetACPEngine(ctx)
	resp, err := engine.CreatePolicy(ctx, &coretypes.CreatePolicyRequest{
		Policy:      `name: test`,
		MarshalType: coretypes.PolicyMarshalingType_SHORT_YAML,
	})
	require.NoError(t, err)

	repo := k.GetRegistrationsCommitmentRepository(ctx)
	service := commitment.NewCommitmentService(k.GetACPEngine(ctx), repo)
	commitment := make([]byte, 32)
	comm, err := service.SetNewCommitment(ctx, resp.Record.Policy.Id, commitment, coretypes.NewActor("test"), params, "source1234")
	require.NoError(t, err)

	// no expired commitments at this point
	expired := k.EndBlocker(ctx)
	require.Nil(t, expired)

	// set commitment to expire
	ctx = ctx.WithBlockTime(time.Now().Add(time.Nanosecond * 2))

	// should return exactly one expired commitment
	expired = k.EndBlocker(ctx)
	require.NotNil(t, expired)
	require.Len(t, expired, 1)

	want := comm
	want.Expired = true
	require.Equal(t, want, expired[0])
}
