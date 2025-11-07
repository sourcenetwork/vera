package keeper

import (
	"context"
	"fmt"

	"github.com/TBD54566975/ssi-sdk/crypto"
	"github.com/TBD54566975/ssi-sdk/did/key"
	ed25519sdk "github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
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

// GetAddressFromDID extracts and returns the bech32 address from a DID.
func (k *Keeper) GetAddressFromDID(ctx context.Context, didStr string) (sdk.AccAddress, error) {
	if err := did.IsValidDID(didStr); err != nil {
		return nil, fmt.Errorf("invalid DID: %w", err)
	}

	didKey := key.DIDKey(didStr)
	pubBytes, _, keyType, err := didKey.Decode()
	if err != nil {
		return nil, fmt.Errorf("failed to decode DID key: %w", err)
	}

	var pubKey cryptotypes.PubKey
	switch keyType {
	case crypto.SECP256k1:
		pubKey = &secp256k1.PubKey{Key: pubBytes}
	case crypto.Ed25519:
		pubKey = &ed25519sdk.PubKey{Key: pubBytes}
	default:
		return nil, fmt.Errorf("unsupported key type: %v", keyType)
	}

	addr := sdk.AccAddress(pubKey.Address())
	return addr, nil
}
