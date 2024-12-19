package keeper

import (
	"context"
	"encoding/hex"
	"fmt"

	comettypes "github.com/cometbft/cometbft/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sourcenetwork/acp_core/pkg/auth"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"

	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

const (
	txHashMapKey  = "tx_hash"
	creatorMapKey = "creator"
)

func (k msgServer) CreatePolicy(goCtx context.Context, msg *types.MsgCreatePolicy) (*types.MsgCreatePolicyResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	engine, err := k.GetACPEngine(ctx)
	if err != nil {
		return nil, err
	}

	principal := coretypes.RootPrincipal()
	goCtx = auth.InjectPrincipal(goCtx, principal)

	tx := comettypes.Tx(ctx.TxBytes())
	txHash := hex.EncodeToString(tx.Hash())

	coreResult, err := engine.CreatePolicy(goCtx, &coretypes.CreatePolicyRequest{
		Policy:      msg.Policy,
		MarshalType: msg.MarshalType,
		Metadata: &coretypes.SuppliedMetadata{
			Attributes: map[string]string{
				txHashMapKey:  txHash,
				creatorMapKey: msg.Creator,
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("CreatePolicy: %w", err)
	}

	// TODO event

	return &types.MsgCreatePolicyResponse{
		Policy: coreResult.Record.Policy,
	}, nil
}
