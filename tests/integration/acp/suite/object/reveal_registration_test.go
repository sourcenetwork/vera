package object

import (
	"testing"

	test "github.com/sourcenetwork/sourcehub/tests/integration/acp"
)

func TestRevealRegistration_UnregisteredObjectGetsRegistered_ReturnsNewRecord(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()
	t.FailNow()
}

func TestRevealRegistration_ObjectRegistereToActor_ReturnOldRecord(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()
	t.FailNow()
}

func TestRevealRegistration_ObjectRegisteredAfterCommitment_RegistrationAmended(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()
	t.FailNow()
}

func TestRevealRegistration_ObjectRegisteredThroughNewerCommitment_RegistrationIsAmended(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()
	t.FailNow()
}

func TestRevealRegistration_ObjectOwnedByUserAfterCommitment_NoOp(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()
	t.FailNow()
}

func TestRevealRegistration_ObjectRegisteredToSomeoneElseAfterCommitment_ErrorsUnauthorized(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()
	t.FailNow()
}

func TestRevealRegistration_InvalidProof_ReturnsError(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()
	t.FailNow()
}

func TestRevealRegistration_ValidProofToExpiredCommitment_ReturnsProtocolError(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()
	t.FailNow()
}

func TestRevealRegistration_(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()
	t.FailNow()
}
