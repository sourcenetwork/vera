package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgInvalidateJWS{}

// ValidateBasic does a sanity check on the provided data.
func (m *MsgInvalidateJWS) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Creator); err != nil {
		return errorsmod.Wrap(err, "invalid creator address")
	}

	if m.TokenHash == "" {
		return ErrInvalidInput.Wrapf("invalid token hash")
	}

	return nil
}
