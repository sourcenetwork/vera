package sdk

import (
	"context"
	"fmt"

	"github.com/sourcenetwork/sourcehub/x/acp/signed_policy_cmd"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

// NewCmdBuilder returns a builder for PolicyCmd objects from a client.
//
// The client is used to fetch the latest ACP module params from SourceHub
// and as a block height fetcher.
func NewCmdBuilder(ctx context.Context, client *Client) (*signed_policy_cmd.CmdBuilder, error) {
	acpClient := client.ACPQueryClient()
	paramsResp, err := acpClient.Params(ctx, &types.QueryParamsRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch params for cmd builder: %v", err)
	}
	params := paramsResp.Params

	clock := signed_policy_cmd.LogicalClockFromCometClient(client.cometClient)

	return signed_policy_cmd.NewCmdBuilder(clock, params), nil
}
