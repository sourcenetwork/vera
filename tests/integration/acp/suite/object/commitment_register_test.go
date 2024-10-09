package object

import (
	"testing"

	test "github.com/sourcenetwork/sourcehub/tests/integration/acp"
)

var policyCommitDef string = `
name: policy
resources:
  resource:
    relations:
      owner:
        types:
          - actor
`

func TestCommitRegistration_CreatingCommitmentReturnsID(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()
	t.FailNow()
}

func TestCommitRegistration_CreateAndGetCommitment(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()
	t.FailNow()
}

func TestCommitRegistration_CommitmentsGenerateDifferentIds(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()
	t.FailNow()
}
