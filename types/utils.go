package types

import (
	"errors"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
)

// IsValidSourceHubAddr verifies whether addr is a valid bech32 prefixed by
// SourceHub's prefix
func IsValidSourceHubAddr(addr string) error {
	bz, err := sdk.GetFromBech32(addr, AccountAddrPrefix)
	if err != nil {
		return err
	}
	err = sdk.VerifyAddressFormat(bz)
	if err != nil {
		return err
	}
	return nil
}

// AccAddressFromBech32 returns an AccAddress from a Bech32 SourceHub address string
func AccAddressFromBech32(address string) (addr sdk.AccAddress, err error) {
	if len(strings.TrimSpace(address)) == 0 {
		return sdk.AccAddress{}, errors.New("empty address string is not allowed")
	}

	bz, err := sdk.GetFromBech32(address, AccountAddrPrefix)
	if err != nil {
		return nil, err
	}

	err = sdk.VerifyAddressFormat(bz)
	if err != nil {
		return nil, err
	}

	return sdk.AccAddress(bz), nil
}

func AddressToAccountBech32(acc sdk.AccAddress) (string, error) {
	bech32Addr, err := bech32.ConvertAndEncode(AccountAddrPrefix, acc)
	if err != nil {
		return "", err
	}
	return bech32Addr, nil
}

func AddressToValidatorBech32(acc sdk.AccAddress) (string, error) {
	bech32Addr, err := bech32.ConvertAndEncode(AccountAddrPrefix+"valoper", acc)
	if err != nil {
		return "", err
	}
	return bech32Addr, nil
}
