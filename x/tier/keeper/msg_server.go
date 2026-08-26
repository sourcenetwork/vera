package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/sourcenetwork/vera/x/tier/types"
)

type msgServer struct {
	*Keeper
}

var _ types.MsgServer = &msgServer{}

func NewMsgServerImpl(keeper *Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

func (m *msgServer) UpdateParams(ctx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	authority := m.Keeper.GetAuthority()
	if msg.Authority != authority {
		return nil, types.ErrUnauthorized.Wrapf("invalid authority: %s", msg.Authority)
	}

	err := msg.Params.Validate()
	if err != nil {
		return nil, types.ErrInvalidRequest.Wrapf("invalid params: %s", err)
	}

	err = m.Keeper.SetParams(ctx, msg.Params)
	if err != nil {
		return nil, types.ErrInvalidRequest.Wrapf("update params: %s", err)
	}

	return &types.MsgUpdateParamsResponse{}, nil
}

func (m *msgServer) Lock(ctx context.Context, msg *types.MsgLock) (*types.MsgLockResponse, error) {
	// Input validation has been done by ValidateBasic.
	delAddr := sdk.MustAccAddressFromBech32(msg.DelegatorAddress)
	valAddr := types.MustValAddressFromBech32(msg.ValidatorAddress)

	err := m.Keeper.Lock(ctx, delAddr, valAddr, msg.Stake.Amount)
	if err != nil {
		return nil, errorsmod.Wrap(err, "lock")
	}

	return &types.MsgLockResponse{}, nil
}

func (m *msgServer) LockAuto(ctx context.Context, msg *types.MsgLockAuto) (*types.MsgLockAutoResponse, error) {
	// Input validation has been done by ValidateBasic.
	delAddr := sdk.MustAccAddressFromBech32(msg.DelegatorAddress)

	valAddr, err := m.Keeper.LockAuto(ctx, delAddr, msg.Stake.Amount)
	if err != nil {
		return nil, errorsmod.Wrap(err, "lock auto")
	}

	return &types.MsgLockAutoResponse{ValidatorAddress: valAddr.String()}, nil
}

func (m *msgServer) Unlock(ctx context.Context, msg *types.MsgUnlock) (*types.MsgUnlockResponse, error) {
	// Input validation has been done by ValidateBasic.
	delAddr := sdk.MustAccAddressFromBech32(msg.DelegatorAddress)
	valAddr := types.MustValAddressFromBech32(msg.ValidatorAddress)

	creationHeight, completionTime, unlockTime, err := m.Keeper.Unlock(ctx, delAddr, valAddr, msg.Stake.Amount)
	if err != nil {
		return nil, errorsmod.Wrap(err, "undelegate")
	}

	return &types.MsgUnlockResponse{CreationHeight: creationHeight, CompletionTime: completionTime, UnlockTime: unlockTime}, nil
}

func (m *msgServer) CancelUnlocking(ctx context.Context, msg *types.MsgCancelUnlocking) (*types.MsgCancelUnlockingResponse, error) {
	// Input validation has been done by ValidateBasic.
	delAddr := sdk.MustAccAddressFromBech32(msg.DelegatorAddress)
	valAddr := types.MustValAddressFromBech32(msg.ValidatorAddress)

	err := m.Keeper.CancelUnlocking(ctx, delAddr, valAddr, msg.CreationHeight, msg.Stake.Amount)
	if err != nil {
		return nil, errorsmod.Wrap(err, "cancel unlocking")
	}

	return &types.MsgCancelUnlockingResponse{}, nil
}

func (m *msgServer) Redelegate(ctx context.Context, msg *types.MsgRedelegate) (*types.MsgRedelegateResponse, error) {
	// Input validation has been done by ValidateBasic.
	delAddr := sdk.MustAccAddressFromBech32(msg.DelegatorAddress)
	srcValAddr := types.MustValAddressFromBech32(msg.SrcValidatorAddress)
	dstValAddr := types.MustValAddressFromBech32(msg.DstValidatorAddress)

	completionTime, err := m.Keeper.Redelegate(ctx, delAddr, srcValAddr, dstValAddr, msg.Stake.Amount)
	if err != nil {
		return nil, errorsmod.Wrap(err, "redelegate")
	}

	return &types.MsgRedelegateResponse{CompletionTime: completionTime}, nil
}

func (m *msgServer) CreateDeveloper(ctx context.Context, msg *types.MsgCreateDeveloper) (*types.MsgCreateDeveloperResponse, error) {
	// Input validation has been done by ValidateBasic.
	developerAddr := sdk.MustAccAddressFromBech32(msg.Developer)

	err := m.Keeper.CreateDeveloper(ctx, developerAddr, msg.AutoLockEnabled)
	if err != nil {
		return nil, errorsmod.Wrap(err, "create developer")
	}

	return &types.MsgCreateDeveloperResponse{}, nil
}

func (m *msgServer) UpdateDeveloper(ctx context.Context, msg *types.MsgUpdateDeveloper) (*types.MsgUpdateDeveloperResponse, error) {
	// Input validation has been done by ValidateBasic.
	developerAddr := sdk.MustAccAddressFromBech32(msg.Developer)

	err := m.Keeper.UpdateDeveloper(ctx, developerAddr, msg.AutoLockEnabled)
	if err != nil {
		return nil, errorsmod.Wrap(err, "update developer")
	}

	return &types.MsgUpdateDeveloperResponse{}, nil
}

func (m *msgServer) RemoveDeveloper(ctx context.Context, msg *types.MsgRemoveDeveloper) (*types.MsgRemoveDeveloperResponse, error) {
	// Input validation has been done by ValidateBasic.
	developerAddr := sdk.MustAccAddressFromBech32(msg.Developer)

	err := m.Keeper.RemoveDeveloper(ctx, developerAddr)
	if err != nil {
		return nil, errorsmod.Wrap(err, "remove developer")
	}

	return &types.MsgRemoveDeveloperResponse{}, nil
}

func (m *msgServer) AddUserSubscription(
	ctx context.Context,
	msg *types.MsgAddUserSubscription,
) (*types.MsgAddUserSubscriptionResponse, error) {
	// Input validation has been done by ValidateBasic.
	developerAddr := sdk.MustAccAddressFromBech32(msg.Developer)

	err := m.Keeper.AddUserSubscription(ctx, developerAddr, msg.UserDid, msg.Amount, msg.Period)
	if err != nil {
		return nil, errorsmod.Wrap(err, "add subscription")
	}

	return &types.MsgAddUserSubscriptionResponse{}, nil
}

func (m *msgServer) UpdateUserSubscription(
	ctx context.Context,
	msg *types.MsgUpdateUserSubscription,
) (*types.MsgUpdateUserSubscriptionResponse, error) {
	// Input validation has been done by ValidateBasic.
	developerAddr := sdk.MustAccAddressFromBech32(msg.Developer)

	err := m.Keeper.UpdateUserSubscription(ctx, developerAddr, msg.UserDid, msg.Amount, msg.Period)
	if err != nil {
		return nil, errorsmod.Wrap(err, "update user subscription")
	}

	return &types.MsgUpdateUserSubscriptionResponse{}, nil
}

func (m *msgServer) RemoveUserSubscription(
	ctx context.Context,
	msg *types.MsgRemoveUserSubscription,
) (*types.MsgRemoveUserSubscriptionResponse, error) {
	// Input validation has been done by ValidateBasic.
	developerAddr := sdk.MustAccAddressFromBech32(msg.Developer)

	err := m.Keeper.RemoveUserSubscription(ctx, developerAddr, msg.UserDid)
	if err != nil {
		return nil, errorsmod.Wrap(err, "remove user subscription")
	}

	return &types.MsgRemoveUserSubscriptionResponse{}, nil
}
