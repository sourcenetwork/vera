package keeper_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/sourcenetwork/vera/testutil/keeper"
	"github.com/sourcenetwork/vera/x/tier/keeper"
	"github.com/sourcenetwork/vera/x/tier/types"
)

func setupMsgServer(t testing.TB) (keeper.Keeper, types.MsgServer, context.Context) {
	k, ctx := keepertest.TierKeeper(t)
	return k, keeper.NewMsgServerImpl(&k), ctx
}

func TestMsgServer(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	require.NotNil(t, ms)
	require.NotNil(t, ctx)
	require.NotEmpty(t, k)
}

func TestUpdateParams(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	params := k.GetParams(ctx)

	tests := []struct {
		name      string
		devFee    int64
		insFee    int64
		expErr    bool
		expErrMsg string
	}{
		{
			name:      "combined fees exceed 100 is rejected",
			devFee:    60,
			insFee:    41,
			expErr:    true,
			expErrMsg: "invalid params",
		},
		{
			name:   "combined fees equal 100 is accepted",
			devFee: 60,
			insFee: 40,
			expErr: false,
		},
		{
			name:   "default fees are accepted",
			devFee: params.DeveloperPoolFee,
			insFee: params.InsurancePoolFee,
			expErr: false,
		},
		{
			name:      "int64 overflow combined fees are rejected",
			devFee:    9223372036854775807,
			insFee:    1,
			expErr:    true,
			expErrMsg: "invalid params",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := params
			p.DeveloperPoolFee = tt.devFee
			p.InsurancePoolFee = tt.insFee
			_, err := ms.UpdateParams(ctx, &types.MsgUpdateParams{
				Authority: k.GetAuthority(),
				Params:    p,
			})
			if tt.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expErrMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
