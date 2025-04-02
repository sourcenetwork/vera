package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sourcenetwork/acp_core/pkg/errors"
	hubtypes "github.com/sourcenetwork/sourcehub/types"
	"github.com/sourcenetwork/sourcehub/x/acp/did"

	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

func (k *Keeper) issueDIDFromAccountAddr(ctx sdk.Context, addr string) (string, error) {
	sdkAddr, err := hubtypes.AccAddressFromBech32(addr)
	if err != nil {
		return "", types.NewErrInvalidAccAddrErr(err, addr)
	}

	acc := k.accountKeeper.GetAccount(ctx, sdkAddr)
	if acc == nil {
		return "", types.NewAccNotFoundErr(addr)
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
