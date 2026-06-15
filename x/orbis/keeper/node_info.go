package keeper

import (
	"encoding/hex"
	"slices"
	"strings"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/sourcenetwork/sourcehub/x/orbis/types"
)

const compressedPubKeyLen = 33

func nodeInfoAllowsRing(nodeInfo *types.NodeInfo, ring *types.Ring) bool {
	return slices.Contains(nodeInfo.WhitelistedPolicyIds, ring.PolicyId) ||
		slices.Contains(nodeInfo.WhitelistedRingIds, ring.Id)
}

func validateNodeInfo(nodeInfo *types.NodeInfo) error {
	switch {
	case nodeInfo.PeerId == "":
		return errorsmod.Wrap(types.ErrInvalidNodeInfo, "missing peer_id")
	case nodeInfo.ControllerKey == "":
		return errorsmod.Wrap(types.ErrInvalidNodeInfo, "missing controller_key")
	}
	if err := validateUniqueNonEmptyStrings("whitelisted_policy_ids", nodeInfo.WhitelistedPolicyIds); err != nil {
		return err
	}
	if err := validateUniqueNonEmptyStrings("whitelisted_ring_ids", nodeInfo.WhitelistedRingIds); err != nil {
		return err
	}
	keyHex := strings.ToLower(strings.TrimPrefix(nodeInfo.ControllerKey, "0x"))
	decoded, err := hex.DecodeString(keyHex)
	if err != nil || len(decoded) != compressedPubKeyLen {
		return errorsmod.Wrap(types.ErrInvalidNodeInfo, "invalid controller_key encoding")
	}
	nodeInfo.ControllerKey = keyHex
	return nil
}

func validateUniqueNonEmptyStrings(field string, values []string) error {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value == "" {
			return errorsmod.Wrapf(types.ErrInvalidNodeInfo, "%s contains an empty value", field)
		}
		if _, exists := seen[value]; exists {
			return errorsmod.Wrapf(types.ErrInvalidNodeInfo, "%s contains duplicate value %q", field, value)
		}
		seen[value] = struct{}{}
	}
	return nil
}

// signerPublicKeyHex returns the hex-encoded compressed public key for a bech32 address.
// The ante handler populates the account's public key before message handlers run,
// so it is guaranteed non-nil for the transaction signer.
func signerPublicKeyHex(ctx sdk.Context, k *Keeper, address string) (string, error) {
	addr, err := sdk.AccAddressFromBech32(address)
	if err != nil {
		return "", errorsmod.Wrapf(types.ErrInvalidNodeInfo, "invalid signer address: %s", err)
	}

	account := k.accountKeeper.GetAccount(ctx, addr)
	if account == nil {
		return "", errorsmod.Wrapf(types.ErrInvalidNodeInfo, "account not found for address %s", address)
	}

	pubKey := account.GetPubKey()
	if pubKey == nil {
		return "", errorsmod.Wrapf(types.ErrInvalidNodeInfo, "public key not set for account %s", address)
	}

	return hex.EncodeToString(pubKey.Bytes()), nil
}
