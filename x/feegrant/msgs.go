package feegrant

import (
	"github.com/cosmos/gogoproto/proto"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var (
	_, _, _, _ sdk.Msg                       = &MsgGrantAllowance{}, &MsgRevokeAllowance{}, &MsgGrantDIDAllowance{}, &MsgRevokeDIDAllowance{}
	_, _       types.UnpackInterfacesMessage = &MsgGrantAllowance{}, &MsgGrantDIDAllowance{}
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

// NewMsgRevokeDIDAllowance returns a message to revoke a fee allowance for a given granter and DID.
func NewMsgRevokeDIDAllowance(granter sdk.AccAddress, granteeDID string) MsgRevokeDIDAllowance {
	return MsgRevokeDIDAllowance{Granter: granter.String(), GranteeDid: granteeDID}
}
