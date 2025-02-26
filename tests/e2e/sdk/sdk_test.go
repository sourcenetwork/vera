package sdk

import (
	"context"
	"testing"

	coretypes "github.com/sourcenetwork/acp_core/pkg/types"

	"github.com/sourcenetwork/sourcehub/sdk"
	"github.com/sourcenetwork/sourcehub/testutil/e2e"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
	"github.com/stretchr/testify/require"
)

func TestSDKBasic(t *testing.T) {
	network := e2e.TestNetwork{}

	network.Setup(t)

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
