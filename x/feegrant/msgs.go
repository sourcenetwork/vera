package feegrant

import (
	"github.com/cosmos/gogoproto/proto"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var (
	_, _, _, _, _, _ sdk.Msg = &MsgGrantAllowance{}, &MsgRevokeAllowance{}, &MsgPruneAllowances{},
		&MsgGrantDIDAllowance{}, &MsgExpireDIDAllowance{}, &MsgPruneDIDAllowances{}
	_, _ types.UnpackInterfacesMessage = &MsgGrantAllowance{}, &MsgGrantDIDAllowance{}
)

// NewMsgGrantAllowance creates a new MsgGrantAllowance.
func NewMsgGrantAllowance(feeAllowance FeeAllowanceI, granter, grantee sdk.AccAddress) (*MsgGrantAllowance, error) {
	msg, ok := feeAllowance.(proto.Message)
	if !ok {
		return nil, errorsmod.Wrapf(sdkerrors.ErrPackAny, "cannot proto marshal %T", msg)
	}
	any, err := types.NewAnyWithValue(msg)
	if err != nil {
		return nil, err
	}

	return &MsgGrantAllowance{
		Granter:   granter.String(),
		Grantee:   grantee.String(),
		Allowance: any,
	}, nil
}

// GetFeeAllowanceI returns unpacked FeeAllowance.
func (msg MsgGrantAllowance) GetFeeAllowanceI() (FeeAllowanceI, error) {
	allowance, ok := msg.Allowance.GetCachedValue().(FeeAllowanceI)
	if !ok {
		return nil, errorsmod.Wrap(ErrNoAllowance, "failed to get allowance")
	}

	return allowance, nil
}

// UnpackInterfaces implements UnpackInterfacesMessage.UnpackInterfaces
func (msg MsgGrantAllowance) UnpackInterfaces(unpacker types.AnyUnpacker) error {
	var allowance FeeAllowanceI
	return unpacker.UnpackAny(msg.Allowance, &allowance)
}

// NewMsgRevokeAllowance returns a message to revoke a fee allowance for a given granter and grantee.
func NewMsgRevokeAllowance(granter, grantee sdk.AccAddress) MsgRevokeAllowance {
	return MsgRevokeAllowance{Granter: granter.String(), Grantee: grantee.String()}
}

// NewMsgGrantDIDAllowance creates a new MsgGrantDIDAllowance.
func NewMsgGrantDIDAllowance(feeAllowance FeeAllowanceI, granter sdk.AccAddress, granteeDID string) (*MsgGrantDIDAllowance, error) {
	msg, ok := feeAllowance.(proto.Message)
	if !ok {
		return nil, errorsmod.Wrapf(sdkerrors.ErrPackAny, "cannot proto marshal %T", msg)
	}
	any, err := types.NewAnyWithValue(msg)
	if err != nil {
		return nil, err
	}

	return &MsgGrantDIDAllowance{
		Granter:    granter.String(),
		GranteeDid: granteeDID,
		Allowance:  any,
	}, nil
}

// GetFeeAllowanceI returns unpacked FeeAllowance for DID grants.
func (msg MsgGrantDIDAllowance) GetFeeAllowanceI() (FeeAllowanceI, error) {
	allowance, ok := msg.Allowance.GetCachedValue().(FeeAllowanceI)
	if !ok {
		return nil, errorsmod.Wrap(ErrNoAllowance, "failed to get allowance")
	}

	return allowance, nil
}

// UnpackInterfaces implements UnpackInterfacesMessage.UnpackInterfaces for DID grants.
func (msg MsgGrantDIDAllowance) UnpackInterfaces(unpacker types.AnyUnpacker) error {
	var allowance FeeAllowanceI
	return unpacker.UnpackAny(msg.Allowance, &allowance)
}

func (msg MsgGrantDIDAllowance) ValidateBasic() error {
	if msg.Granter == "" {
		return errorsmod.Wrap(sdkerrors.ErrInvalidAddress, "missing granter address")
	}
	if len(msg.GranteeDid) <= 4 || msg.GranteeDid[:4] != "did:" {
		return ErrInvalidDID
	}
	if msg.Allowance == nil {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "missing allowance")
	}

	allowance, err := msg.GetFeeAllowanceI()
	if err != nil {
		return err
	}

	return allowance.ValidateBasic()
}

// NewMsgExpireDIDAllowance returns a message to expire a fee allowance for a given granter and DID.
func NewMsgExpireDIDAllowance(granter sdk.AccAddress, granteeDID string) MsgExpireDIDAllowance {
	return MsgExpireDIDAllowance{Granter: granter.String(), GranteeDid: granteeDID}
}

// NewMsgPruneDIDAllowances returns a message to prune expired DID allowances.
func NewMsgPruneDIDAllowances(pruner sdk.AccAddress) MsgPruneDIDAllowances {
	return MsgPruneDIDAllowances{Pruner: pruner.String()}
}

// ValidateBasic implements sdk.Msg
func (msg MsgPruneDIDAllowances) ValidateBasic() error {
	if msg.Pruner == "" {
		return errorsmod.Wrap(sdkerrors.ErrInvalidAddress, "missing pruner address")
	}
	return nil
}

func (msg MsgExpireDIDAllowance) ValidateBasic() error {
	if msg.Granter == "" {
		return errorsmod.Wrap(sdkerrors.ErrInvalidAddress, "missing granter address")
	}
	if len(msg.GranteeDid) <= 4 || msg.GranteeDid[:4] != "did:" {
		return ErrInvalidDID
	}
	return nil
}
