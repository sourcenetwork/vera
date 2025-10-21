package keeper

import (
	"context"
	"strings"

	errorsmod "cosmossdk.io/errors"
	"github.com/sourcenetwork/sourcehub/x/feegrant"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the feegrant MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(k Keeper) feegrant.MsgServer {
	return &msgServer{
		Keeper: k,
	}
}

var _ feegrant.MsgServer = msgServer{}

// GrantAllowance grants an allowance from the granter's funds to be used by the grantee.
func (k msgServer) GrantAllowance(goCtx context.Context, msg *feegrant.MsgGrantAllowance) (*feegrant.MsgGrantAllowanceResponse, error) {
	if strings.EqualFold(msg.Grantee, msg.Granter) {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidAddress, "cannot self-grant fee authorization")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	grantee, err := k.authKeeper.AddressCodec().StringToBytes(msg.Grantee)
	if err != nil {
		return nil, err
	}

	granter, err := k.authKeeper.AddressCodec().StringToBytes(msg.Granter)
	if err != nil {
		return nil, err
	}

	if f, _ := k.GetAllowance(ctx, granter, grantee); f != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "fee allowance already exists")
	}

	allowance, err := msg.GetFeeAllowanceI()
	if err != nil {
		return nil, err
	}

	if err := allowance.ValidateBasic(); err != nil {
		return nil, err
	}

	err = k.Keeper.GrantAllowance(ctx, granter, grantee, allowance)
	if err != nil {
		return nil, err
	}

	return &feegrant.MsgGrantAllowanceResponse{}, nil
}

// RevokeAllowance revokes a fee allowance between a granter and grantee.
func (k msgServer) RevokeAllowance(goCtx context.Context, msg *feegrant.MsgRevokeAllowance) (*feegrant.MsgRevokeAllowanceResponse, error) {
	if msg.Grantee == msg.Granter {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidAddress, "addresses must be different")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	grantee, err := k.authKeeper.AddressCodec().StringToBytes(msg.Grantee)
	if err != nil {
		return nil, err
	}

	granter, err := k.authKeeper.AddressCodec().StringToBytes(msg.Granter)
	if err != nil {
		return nil, err
	}

	err = k.Keeper.revokeAllowance(ctx, granter, grantee)
	if err != nil {
		return nil, err
	}

	return &feegrant.MsgRevokeAllowanceResponse{}, nil
}

// PruneAllowances removes expired allowances from the store.
func (k msgServer) PruneAllowances(ctx context.Context, req *feegrant.MsgPruneAllowances) (*feegrant.MsgPruneAllowancesResponse, error) {
	// 75 is an arbitrary value, we can change it later if needed
	err := k.RemoveExpiredAllowances(ctx, 75)
	if err != nil {
		return nil, err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			feegrant.EventTypePruneFeeGrant,
			sdk.NewAttribute(feegrant.AttributeKeyPruner, req.Pruner),
		),
	)

	return &feegrant.MsgPruneAllowancesResponse{}, nil
}

// GrantDIDAllowance grants an allowance from the granter's funds to be used by a DID.
func (k msgServer) GrantDIDAllowance(
	goCtx context.Context,
	msg *feegrant.MsgGrantDIDAllowance,
) (*feegrant.MsgGrantDIDAllowanceResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	granter, err := k.authKeeper.AddressCodec().StringToBytes(msg.Granter)
	if err != nil {
		return nil, err
	}

	// Check if DID allowance already exists
	if existingAllowance, _ := k.GetDIDAllowance(ctx, granter, msg.GranteeDid); existingAllowance != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "DID allowance already exists")
	}

	allowance, err := msg.GetFeeAllowanceI()
	if err != nil {
		return nil, err
	}

	if err := allowance.ValidateBasic(); err != nil {
		return nil, err
	}

	err = k.Keeper.GrantDIDAllowance(ctx, granter, msg.GranteeDid, allowance)
	if err != nil {
		return nil, err
	}

	return &feegrant.MsgGrantDIDAllowanceResponse{}, nil
}

// ExpireDIDAllowance expires a periodic allowance by setting the expiration to current PeriodReset.
func (k msgServer) ExpireDIDAllowance(
	goCtx context.Context,
	msg *feegrant.MsgExpireDIDAllowance,
) (*feegrant.MsgExpireDIDAllowanceResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	granter, err := k.authKeeper.AddressCodec().StringToBytes(msg.Granter)
	if err != nil {
		return nil, err
	}

	err = k.Keeper.ExpireDIDAllowance(ctx, granter, msg.GranteeDid)
	if err != nil {
		return nil, err
	}

	return &feegrant.MsgExpireDIDAllowanceResponse{}, nil
}

// PruneDIDAllowances removes expired DID allowances from the store.
func (k msgServer) PruneDIDAllowances(ctx context.Context, req *feegrant.MsgPruneDIDAllowances) (*feegrant.MsgPruneDIDAllowancesResponse, error) {
	// 75 is an arbitrary value, we can change it later if needed
	err := k.RemoveExpiredDIDAllowances(ctx, 75)
	if err != nil {
		return nil, err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			feegrant.EventTypePruneDIDFeeGrant,
			sdk.NewAttribute(feegrant.AttributeKeyPruner, req.Pruner),
		),
	)

	return &feegrant.MsgPruneDIDAllowancesResponse{}, nil
}
