package keeper

import (
	"context"
	"fmt"

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

	did, err := did.IssueDID(acc)
	if err != nil {
		return "", errors.NewFromCause("could not issue did",
			err,
			errors.ErrorType_BAD_INPUT,
			errors.Pair("address", addr),
		)
	}
	return did, nil
}
