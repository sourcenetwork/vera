package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sourcenetwork/acp_core/pkg/errors"
	hubtypes "github.com/sourcenetwork/sourcehub/types"
	"github.com/sourcenetwork/sourcehub/x/acp/did"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

// IssueDIDFromAccountAddr issues a DID based on the specified address string.
func (k *Keeper) IssueDIDFromAccountAddr(ctx context.Context, addr string) (string, error) {
	sdkAddr, err := hubtypes.AccAddressFromBech32(addr)
	if err != nil {
		return "", fmt.Errorf("IssueDIDFromAccountAddr: %v: %w", err, types.NewErrInvalidAccAddrErr(err, addr))
	}

	acc := k.accountKeeper.GetAccount(ctx, sdkAddr)
	if acc == nil {
		return "", fmt.Errorf("IssueDIDFromAccountAddr: %w", types.NewAccNotFoundErr(addr))
	}

	// Check if this is an ICA address
	if _, found := k.hubKeeper.GetICAConnection(sdk.UnwrapSDKContext(ctx), addr); found {
		controllerDID := did.IssueInterchainAccountDID(addr)
		return controllerDID, nil
	}

	did, err := did.IssueDID(acc)
	if err != nil {
		return "", errors.NewWithCause("could not issue did",
			err,
			errors.ErrorType_BAD_INPUT,
			errors.Pair("address", addr),
		)
	}
	return did, nil
}
