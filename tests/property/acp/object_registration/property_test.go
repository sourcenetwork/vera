package object_registration

import (
	"fmt"
	"testing"

	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	test "github.com/sourcenetwork/sourcehub/tests/integration/acp"
	"github.com/sourcenetwork/sourcehub/tests/property"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
	"github.com/stretchr/testify/require"
)

func TestObjectRegistrationProperties(t *testing.T) {
	for i := 0; i < TestCount; i++ {
		t.Run(fmt.Sprintf("%v", i), func(t *testing.T) {
			runPropTest(t)
		})
	}
}

func runPropTest(t *testing.T) {
	ctx := test.NewTestCtx(t)
	ctx.Strategy = test.Direct
	ctx.ActorType = test.Actor_SECP256K1

	resp, err := ctx.Executor.CreatePolicy(ctx, &types.MsgCreatePolicy{
		Policy:      Policy,
		MarshalType: coretypes.PolicyMarshalingType_SHORT_YAML,
		Creator:     ctx.TxSigner.SourceHubAddr,
	})
	require.NoError(t, err)

	state := InitialState(ctx, resp.Record.Policy.Id)

	ops := make([]Operation, 0, OperationsPerTest)
	for count, max := 0, 0; count < OperationsPerTest || max < MaxTriesPerTest; max++ {
		kind := property.PickAny(ListOperationKinds())
		ok := Precoditions(state, kind)
		if !ok {
			continue
		}

		op := GenerateOperation(t, state, kind)
		ops = append(ops, op)
		count++

		result, err := ctx.Executor.DirectPolicyCmd(ctx, &types.MsgDirectPolicyCmd{
			Creator:  op.Actor.SourceHubAddr,
			PolicyId: state.PolicyId,
			Cmd:      &op.Request,
		})

		op.ResultErr = err
		if result != nil {
			op.Result = *result.Result
		}

		state = NextState(t, state, op)

		if state.Registered {
			ownerRecord, err := ctx.Executor.ObjectOwner(ctx, &types.QueryObjectOwnerRequest{
				PolicyId: state.PolicyId,
				Object:   &state.Object,
			})
			require.NoError(t, err, "ObjectOwner call failed")
			op.ResultRecord = *ownerRecord.Record
		}

		err = Post(state, op)

		if err != nil {
			t.Logf("Post condition failed: %v", err)
			t.Logf("State: %v", state)
			t.FailNow()
		}

		ts, err := ctx.Executor.GetLastBlockTs(ctx)
		require.NoError(t, err)
		state.LastTs = *ts
	}
	t.Logf("test concluded")
	t.Logf("model: %v", state.Model)
}
