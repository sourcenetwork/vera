package signed_policy_cmd

import (
	"context"
	"testing"

	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/stretchr/testify/require"

	"github.com/sourcenetwork/vera/x/acp/types"
)

type fixedClock struct{ h uint64 }

func (c fixedClock) GetTimestampNow(ctx context.Context) (uint64, error) { return c.h, nil }

func newTestBuilder(t *testing.T) *CmdBuilder {
	t.Helper()
	return NewCmdBuilder(fixedClock{h: 100}, types.DefaultParams())
}

func TestCmdBuilder_ValidateCmd_RegisterObject_OK(t *testing.T) {
	b := newTestBuilder(t)
	b.Actor("did:key:test-actor")
	b.PolicyID("policy-1")
	b.PolicyCmd(types.NewRegisterObjectCmd(coretypes.NewObject("file", "foo")))
	payload, err := b.Build(context.Background())
	require.NoError(t, err)
	require.Equal(t, "policy-1", payload.PolicyId)
}

func TestCmdBuilder_ValidateCmd_RegisterObject_Invalid(t *testing.T) {
	b := newTestBuilder(t)
	b.Actor("did:key:test-actor")
	b.PolicyID("policy-1")
	b.PolicyCmd(types.NewRegisterObjectCmd(coretypes.NewObject("file", "")))
	_, err := b.Build(context.Background())
	require.Error(t, err)
}

func TestCmdBuilder_ValidateCmd_SetRelationship_OK(t *testing.T) {
	b := newTestBuilder(t)
	b.Actor("did:key:test-actor")
	b.PolicyID("policy-1")
	rel := coretypes.NewActorRelationship("file", "foo", "viewer", "did:key:bob")
	b.PolicyCmd(types.NewSetRelationshipCmd(rel))
	_, err := b.Build(context.Background())
	require.NoError(t, err)
}

func TestCmdBuilder_ValidateCmd_SetRelationship_InvalidRelation(t *testing.T) {
	b := newTestBuilder(t)
	b.Actor("did:key:test-actor")
	b.PolicyID("policy-1")
	rel := coretypes.NewActorRelationship("file", "foo", "", "did:key:bob")
	b.PolicyCmd(types.NewSetRelationshipCmd(rel))
	_, err := b.Build(context.Background())
	require.Error(t, err)
}
