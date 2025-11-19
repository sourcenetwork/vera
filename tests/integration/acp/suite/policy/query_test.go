package policy

import (
	"testing"

	coretypes "github.com/sourcenetwork/acp_core/pkg/types"

	test "github.com/sourcenetwork/sourcehub/tests/integration/acp"
	acptypes "github.com/sourcenetwork/sourcehub/x/acp/types"
)

func TestQuery_Policy_ReturnsPolicyAndRaw(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	policyStr := `
name: policy
`

	a1 := test.CreatePolicyAction{
		Policy:  policyStr,
		Creator: ctx.TxSigner,
	}
	pol := a1.Run(ctx)
	pol.Resources = nil

	metadata := ctx.GetSignerRecordMetadata()
	metadata.CreationTs.BlockHeight++
	action := test.GetPolicyAction{
		Id: pol.Id,
		Expected: &acptypes.PolicyRecord{
			Policy:      pol,
			MarshalType: coretypes.PolicyMarshalingType_SHORT_YAML,
			RawPolicy:   policyStr,
			Metadata:    metadata,
		},
	}
	action.Run(ctx)
}

func TestQuery_Policy_UnknownPolicyReturnsPolicyNotFoundErr(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	action := test.GetPolicyAction{
		Id:          "blahblahblah",
		ExpectedErr: acptypes.ErrorType_NOT_FOUND,
	}
	action.Run(ctx)
}
