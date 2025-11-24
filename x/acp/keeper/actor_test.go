package keeper

import (
	"testing"

	"github.com/stretchr/testify/require"

	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	appparams "github.com/sourcenetwork/sourcehub/app/params"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

func TestGetActorDID_WithExtractedDID(t *testing.T) {
	ctx, k, accK := setupKeeper(t)

	testAddr := accK.GenAccount().GetAddress().String()
	extractedDID := "did:key:z6MkhaXgBZDvotDkL5257faiztiGiC2QtKLGpbnnEGta2doK"

	// Inject extracted DID into context
	ctx = ctx.WithValue(appparams.ExtractedDIDContextKey, extractedDID)

	// GetActorDID should return the extracted DID
	actorDID, err := k.GetActorDID(ctx, testAddr)
	require.NoError(t, err)
	require.Equal(t, extractedDID, actorDID, "Should use extracted DID from context")
}

func TestGetActorDID_WithoutExtractedDID(t *testing.T) {
	ctx, k, accK := setupKeeper(t)

	testAddr := accK.GenAccount().GetAddress().String()

	// GetActorDID should issue DID from account address (no extracted DID in context)
	actorDID, err := k.GetActorDID(ctx, testAddr)
	require.NoError(t, err)
	require.NotEmpty(t, actorDID, "Should issue DID from account address")
	require.Contains(t, actorDID, "did:key:", "Should be a valid DID")
}

func TestGetActorDID_WithEmptyExtractedDID(t *testing.T) {
	ctx, k, accK := setupKeeper(t)

	testAddr := accK.GenAccount().GetAddress().String()
	ctx = ctx.WithValue(appparams.ExtractedDIDContextKey, "")

	// GetActorDID should fall back to issuing DID from account address
	actorDID, err := k.GetActorDID(ctx, testAddr)
	require.NoError(t, err)
	require.NotEmpty(t, actorDID, "Should issue DID from account address when extracted DID is empty")
	require.Contains(t, actorDID, "did:key:", "Should be a valid DID")
}

func TestCreatePolicy_WithExtractedDID(t *testing.T) {
	ctx, msgServer, accK := setupMsgServer(t)

	// Create an account and extract DID from bearer token
	testAddr := accK.GenAccount().GetAddress().String()
	extractedDID := "did:key:z6MkhaXgBZDvotDkL5257faiztiGiC2QtKLGpbnnEGta2doK"
	ctx = ctx.WithValue(appparams.ExtractedDIDContextKey, extractedDID)

	// Create a policy
	msg := &types.MsgCreatePolicy{
		Creator:     testAddr,
		Policy:      "name: test\nresources:\n  file:\n    relations:\n      owner: {}\n",
		MarshalType: coretypes.PolicyMarshalingType_YAML,
	}

	resp, err := msgServer.CreatePolicy(ctx, msg)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Record)
	require.NotNil(t, resp.Record.Metadata)

	// The policy owner should be the extracted DID
	require.Equal(t, extractedDID, resp.Record.Metadata.OwnerDid, "Policy owner DID should be the extracted DID")
}

func TestCreatePolicy_WithoutExtractedDID(t *testing.T) {
	ctx, msgServer, accK := setupMsgServer(t)

	// Create an account (no extracted DID)
	testAddr := accK.GenAccount().GetAddress().String()

	// Create a policy
	msg := &types.MsgCreatePolicy{
		Creator:     testAddr,
		Policy:      "name: test\nresources:\n  file:\n    relations:\n      owner: {}\n",
		MarshalType: coretypes.PolicyMarshalingType_YAML,
	}

	resp, err := msgServer.CreatePolicy(ctx, msg)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Record)
	require.NotNil(t, resp.Record.Metadata)

	// The policy owner should be a DID issued from the creator address
	require.Contains(t, resp.Record.Metadata.OwnerDid, "did:key:", "Policy owner DID should be a valid DID")
	require.NotEmpty(t, resp.Record.Metadata.OwnerDid, "Policy owner DID should not be empty")
}

func TestEditPolicy_WithExtractedDID(t *testing.T) {
	ctx, msgServer, accK := setupMsgServer(t)

	// Create an account and extract DID from bearer token
	testAddr := accK.GenAccount().GetAddress().String()
	extractedDID := "did:key:z6MkhaXgBZDvotDkL5257faiztiGiC2QtKLGpbnnEGta2doK"

	// Inject extracted DID before creating the policy so it becomes the policy owner
	ctx = ctx.WithValue(appparams.ExtractedDIDContextKey, extractedDID)

	createMsg := &types.MsgCreatePolicy{
		Creator: testAddr,
		Policy: `name: test
resources:
  file:
    relations:
      owner:
        types:
          - actor
`,
		MarshalType: coretypes.PolicyMarshalingType_YAML,
	}

	createResp, err := msgServer.CreatePolicy(ctx, createMsg)
	require.NoError(t, err)
	policyID := createResp.Record.Policy.Id

	// Verify the policy was created with the extracted DID as owner
	require.Equal(t, extractedDID, createResp.Record.Metadata.OwnerDid)

	// Edit the policy with the same extracted DID
	editMsg := &types.MsgEditPolicy{
		Creator:  testAddr,
		PolicyId: policyID,
		Policy: `name: test 2
resources:
  file:
    relations:
      owner:
        types:
          - actor
`,
		MarshalType: coretypes.PolicyMarshalingType_YAML,
	}

	editResp, err := msgServer.EditPolicy(ctx, editMsg)
	require.NoError(t, err)
	require.NotNil(t, editResp)
	require.NotNil(t, editResp.Record)
	require.NotNil(t, editResp.Record.Policy)

	// Verify the policy owner didn't change
	require.Equal(t, extractedDID, editResp.Record.Metadata.OwnerDid)
}

func TestDirectPolicyCmd_WithExtractedDID(t *testing.T) {
	ctx, msgServer, accK := setupMsgServer(t)

	// Setup: Create an account and extract DID from bearer token
	testAddr := accK.GenAccount().GetAddress().String()
	extractedDID := "did:key:z6MkhaXgBZDvotDkL5257faiztiGiC2QtKLGpbnnEGta2doK"

	// Inject extracted DID into context before creating policy
	ctx = ctx.WithValue(appparams.ExtractedDIDContextKey, extractedDID)

	createMsg := &types.MsgCreatePolicy{
		Creator: testAddr,
		Policy: `name: test
resources:
  file:
    relations:
      owner:
        types:
          - actor
`,
		MarshalType: coretypes.PolicyMarshalingType_YAML,
	}

	createResp, err := msgServer.CreatePolicy(ctx, createMsg)
	require.NoError(t, err)
	policyID := createResp.Record.Policy.Id

	// Verify the policy was created with the extracted DID as owner
	require.Equal(t, extractedDID, createResp.Record.Metadata.OwnerDid)

	// Execute a policy command to register an object (extracted DID still in context)
	cmdMsg := &types.MsgDirectPolicyCmd{
		Creator:  testAddr,
		PolicyId: policyID,
		Cmd: types.NewRegisterObjectCmd(&coretypes.Object{
			Resource: "file",
			Id:       "file1",
		}),
	}

	cmdResp, err := msgServer.DirectPolicyCmd(ctx, cmdMsg)
	require.NoError(t, err)
	require.NotNil(t, cmdResp)
	require.NotEmpty(t, cmdResp.Result)
}
