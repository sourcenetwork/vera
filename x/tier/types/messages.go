package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	appparams "github.com/sourcenetwork/sourcehub/app/params"
)

var (
	_ sdk.Msg = &MsgLock{}
	_ sdk.Msg = &MsgLockAuto{}
	_ sdk.Msg = &MsgUnlock{}
	_ sdk.Msg = &MsgCancelUnlocking{}
	_ sdk.Msg = &MsgRedelegate{}
	_ sdk.Msg = &MsgCreateDeveloper{}
	_ sdk.Msg = &MsgUpdateDeveloper{}
	_ sdk.Msg = &MsgRemoveDeveloper{}
	_ sdk.Msg = &MsgAddUserSubscription{}
	_ sdk.Msg = &MsgUpdateUserSubscription{}
	_ sdk.Msg = &MsgRemoveUserSubscription{}
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

// MsgLockAuto
func NewMsgLockAuto(delAddr string, stake sdk.Coin) *MsgLockAuto {
	return &MsgLockAuto{
		DelegatorAddress: delAddr,
		Stake:            stake,
	}
}

func (msg *MsgLockAuto) ValidateBasic() error {
	if err := validateAccAddr(msg.DelegatorAddress); err != nil {
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

// MsgCreateDeveloper
func NewMsgCreateDeveloper(developerAddr string, autoLockEnabled bool) *MsgCreateDeveloper {
	return &MsgCreateDeveloper{
		Developer:       developerAddr,
		AutoLockEnabled: autoLockEnabled,
	}
}

func (msg *MsgCreateDeveloper) ValidateBasic() error {
	if err := validateAccAddr(msg.Developer); err != nil {
		return err
	}
	return nil
}

// MsgUpdateDeveloper
func NewMsgUpdateDeveloper(developerAddr string, autoLockEnabled bool) *MsgUpdateDeveloper {
	return &MsgUpdateDeveloper{
		Developer:       developerAddr,
		AutoLockEnabled: autoLockEnabled,
	}
}

func (msg *MsgUpdateDeveloper) ValidateBasic() error {
	if err := validateAccAddr(msg.Developer); err != nil {
		return err
	}
	return nil
}

// MsgRemoveDeveloper
func NewMsgRemoveDeveloper(developerAddr string) *MsgRemoveDeveloper {
	return &MsgRemoveDeveloper{
		Developer: developerAddr,
	}
}

func (msg *MsgRemoveDeveloper) ValidateBasic() error {
	if err := validateAccAddr(msg.Developer); err != nil {
		return err
	}
	return nil
}

// MsgAddUserSubscription
func NewMsgAddUserSubscription(developerAddr, userAddr, userDid string, amount uint64, period uint64) *MsgAddUserSubscription {
	return &MsgAddUserSubscription{
		Developer: developerAddr,
		UserDid:   userDid,
		Amount:    amount,
		Period:    period,
	}
}

func (msg *MsgAddUserSubscription) ValidateBasic() error {
	if err := validateAccAddr(msg.Developer); err != nil {
		return err
	}
	if len(msg.UserDid) <= 4 || msg.UserDid[:4] != "did:" {
		return ErrInvalidDID
	}
	if msg.Amount <= 0 {
		return ErrInvalidAmount
	}
	if msg.Period <= 0 {
		return ErrInvalidSubscribtionPeriod
	}
	return nil
}

// MsgUpdateUserSubscription
func NewMsgUpdateUserSubscription(developerAddr, userAddr, userDid string, amount uint64, period uint64) *MsgUpdateUserSubscription {
	return &MsgUpdateUserSubscription{
		Developer: developerAddr,
		UserDid:   userDid,
		Amount:    amount,
		Period:    period,
	}
}

func (msg *MsgUpdateUserSubscription) ValidateBasic() error {
	if err := validateAccAddr(msg.Developer); err != nil {
		return err
	}
	if len(msg.UserDid) <= 4 || msg.UserDid[:4] != "did:" {
		return ErrInvalidDID
	}
	if msg.Amount <= 0 {
		return ErrInvalidAmount
	}
	if msg.Period <= 0 {
		return ErrInvalidSubscribtionPeriod
	}
	return nil
}

// MsgRemoveUserSubscription
func NewMsgRemoveUserSubscription(developerAddr, userAddr, userDid string) *MsgRemoveUserSubscription {
	return &MsgRemoveUserSubscription{
		Developer: developerAddr,
		UserDid:   userDid,
	}
}

func (msg *MsgRemoveUserSubscription) ValidateBasic() error {
	if err := validateAccAddr(msg.Developer); err != nil {
		return err
	}
	if len(msg.UserDid) <= 4 || msg.UserDid[:4] != "did:" {
		return ErrInvalidDID
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
