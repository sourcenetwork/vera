package sdk

import (
	"context"
	"testing"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"

	"github.com/sourcenetwork/sourcehub/sdk"
	testutil "github.com/sourcenetwork/sourcehub/testutil"
	"github.com/sourcenetwork/sourcehub/testutil/e2e"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
	"github.com/stretchr/testify/require"
)

func TestSDKBasic(t *testing.T) {
	network := e2e.TestNetwork{}

	network.Setup(t)
	t.Cleanup(network.TearDown)

	client := network.GetSDKClient()

	builder, err := sdk.NewTxBuilder(
		sdk.WithSDKClient(client),
		sdk.WithChainID(network.GetChainID()),
	)
	require.NoError(t, err)

	policy := `
name: test policy
`
	msgSet := sdk.MsgSet{}
	mapper := msgSet.WithCreatePolicy(
		types.NewMsgCreatePolicy(
			network.GetValidatorAddr(),
			policy,
			coretypes.PolicyMarshalingType_SHORT_YAML,
		),
	)

	signer := sdk.TxSignerFromCosmosKey(network.GetValidatorKey())

	ctx := context.TODO()
	tx, err := builder.Build(ctx, signer, &msgSet)
	require.NoError(t, err)

	response, err := client.BroadcastTx(ctx, tx)
	require.NoError(t, err)

	network.Network.WaitForNextBlock()

	result, err := network.Client.GetTx(ctx, response.TxHash)
	require.NoError(t, err)
	require.NoError(t, result.Error())

	_, err = mapper.Map(result.TxPayload())
	require.NoError(t, err)
}

func TestSDKWithBearerToken(t *testing.T) {
	network := e2e.TestNetwork{}

	network.Setup(t)
	t.Cleanup(network.TearDown)

	client := network.GetSDKClient()

	validatorAddr := network.GetValidatorAddr()
	bearerToken, _ := testutil.GenerateSignedJWSWithMatchingDID(t, validatorAddr)

	builder, err := sdk.NewTxBuilder(
		sdk.WithSDKClient(client),
		sdk.WithChainID(network.GetChainID()),
		sdk.WithBearerToken(bearerToken),
	)
	require.NoError(t, err)

	policy := `
name: test policy
`
	msgSet := sdk.MsgSet{}
	mapper := msgSet.WithCreatePolicy(
		types.NewMsgCreatePolicy(
			validatorAddr,
			policy,
			coretypes.PolicyMarshalingType_SHORT_YAML,
		),
	)

	signer := sdk.TxSignerFromCosmosKey(network.GetValidatorKey())

	ctx := context.TODO()
	tx, err := builder.Build(ctx, signer, &msgSet)
	require.NoError(t, err)

	extOptsTx, ok := tx.(interface {
		GetExtensionOptions() []*codectypes.Any
	})
	require.True(t, ok, "transaction should support extension options")
	extOpts := extOptsTx.GetExtensionOptions()
	require.Len(t, extOpts, 1, "should have exactly one extension option")

	response, err := client.BroadcastTx(ctx, tx)
	require.NoError(t, err)

	network.Network.WaitForNextBlock()

	result, err := network.Client.GetTx(ctx, response.TxHash)
	require.NoError(t, err)
	require.NoError(t, result.Error())

	_, err = mapper.Map(result.TxPayload())
	require.NoError(t, err)
}
