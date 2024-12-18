package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	appparams "github.com/sourcenetwork/sourcehub/app/params"
)

var (
	_ sdk.Msg = &MsgLock{}
	_ sdk.Msg = &MsgUnlock{}
	_ sdk.Msg = &MsgCancelUnlocking{}
	_ sdk.Msg = &MsgRedelegate{}
)

// MsgLock
func NewMsgLock(delAddr, valAddr string, stake sdk.Coin) *MsgLock {
	return &MsgLock{
		DelegatorAddress: delAddr,
		ValidatorAddress: valAddr,
		Stake:            stake,
	}
}

func (msg *MsgLock) ValidateBasic() error {
	if err := validateAccAddr(msg.DelegatorAddress); err != nil {
		return err
	}
	if err := validateValAddr(msg.ValidatorAddress); err != nil {
		return err
	}
	if err := validateDenom(msg.Stake); err != nil {
		return err
	}
	return nil
}

// MsgUnlock
func NewMsgUnlock(delAddr, valAddr string, stake sdk.Coin) *MsgUnlock {
	return &MsgUnlock{
		DelegatorAddress: delAddr,
		ValidatorAddress: valAddr,
		Stake:            stake,
	}
}

func (msg *MsgUnlock) ValidateBasic() error {
	if err := validateAccAddr(msg.DelegatorAddress); err != nil {
		return err
	}
	if err := validateValAddr(msg.ValidatorAddress); err != nil {
		return err
	}
	if err := validateDenom(msg.Stake); err != nil {
		return err
	}
	return nil
}

// MsgCancelUnlocking
func NewMsgCancelUnlocking(delAddr, valAddr string, stake sdk.Coin, creationHeight int64) *MsgCancelUnlocking {
	return &MsgCancelUnlocking{
		DelegatorAddress: delAddr,
		ValidatorAddress: valAddr,
		Stake:            stake,
		CreationHeight:   creationHeight,
	}
}

func (msg *MsgCancelUnlocking) ValidateBasic() error {
	if err := validateAccAddr(msg.DelegatorAddress); err != nil {
		return err
	}
	if err := validateValAddr(msg.ValidatorAddress); err != nil {
		return err
	}
	if err := validateDenom(msg.Stake); err != nil {
		return err
	}
	return nil
}

// MsgRedelegate
func NewMsgRedelegate(delAddress, srcValAddr, dstValAddr string, stake sdk.Coin) *MsgRedelegate {
	return &MsgRedelegate{
		DelegatorAddress:    delAddress,
		SrcValidatorAddress: srcValAddr,
		DstValidatorAddress: dstValAddr,
		Stake:               stake,
	}
}

func (msg *MsgRedelegate) ValidateBasic() error {
	if msg.SrcValidatorAddress == msg.DstValidatorAddress {
		return ErrInvalidAddress.Wrapf("src and dst validator addresses are the same")
	}
	if err := validateAccAddr(msg.DelegatorAddress); err != nil {
		return err
	}
	if err := validateValAddr(msg.SrcValidatorAddress); err != nil {
		return err
	}
	if err := validateValAddr(msg.DstValidatorAddress); err != nil {
		return err
	}
	if err := validateDenom(msg.Stake); err != nil {
		return err
	}
	return nil
}

func validateAccAddr(address string) error {
	_, err := sdk.AccAddressFromBech32(address)
	if err != nil {
		return ErrInvalidAddress.Wrapf("delegator address %s:%s", address, err)
	}
	return nil
}

func validateValAddr(address string) error {
	_, err := sdk.ValAddressFromBech32(address)
	if err != nil {
		return ErrInvalidAddress.Wrapf("validator address %s:%s", address, err)
	}
	return nil
}

func validateDenom(stake sdk.Coin) error {
	if !stake.IsValid() || !stake.Amount.IsPositive() || !stake.Amount.IsInt64() {
		return ErrInvalidDenom.Wrapf("invalid amount %s", stake)
	}

	if stake.Denom != appparams.DefaultBondDenom {
		return ErrInvalidDenom.Wrapf("got %s, expected %s", stake.Denom, appparams.DefaultBondDenom)
	}
	return nil
}

// Must variant which panics on error
func MustValAddressFromBech32(address string) sdk.ValAddress {
	valAddr, err := sdk.ValAddressFromBech32(address)
	if err != nil {
		panic(err)
	}
	return valAddr
}
