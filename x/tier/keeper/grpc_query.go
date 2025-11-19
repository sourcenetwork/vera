package keeper

import (
	"bytes"
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/sourcenetwork/sourcehub/x/tier/types"
)

var _ types.QueryServer = &Keeper{}

// Params query returns tier module params.
func (k *Keeper) Params(ctx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	return &types.QueryParamsResponse{Params: k.GetParams(ctx)}, nil
}

// Lockup query returns a lockup based on delegator address and validator address.
func (k *Keeper) Lockup(ctx context.Context, req *types.LockupRequest) (*types.LockupResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	delAddr, err := sdk.AccAddressFromBech32(req.DelegatorAddress)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid delegator address")
	}

	valAddr, err := sdk.ValAddressFromBech32(req.ValidatorAddress)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid validator address")
	}

	lockup := k.GetLockup(ctx, delAddr, valAddr)
	if lockup == nil {
		return nil, status.Error(codes.NotFound, "unlocking lockup does not exist")
	}

	return &types.LockupResponse{Lockup: *lockup}, nil
}

// Lockups query returns all delegator lockups with pagination.
func (k *Keeper) Lockups(ctx context.Context, req *types.LockupsRequest) (*types.LockupsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	delAddr, err := sdk.AccAddressFromBech32(req.DelegatorAddress)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid delegator address")
	}

	lockups, pageRes, err := k.getLockupsPaginated(ctx, delAddr, req.Pagination)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.LockupsResponse{Lockups: lockups, Pagination: pageRes}, nil
}

// UnlockingLockup query returns an unlocking lockup based on delAddr, valAddr, and creationHeight.
func (k *Keeper) UnlockingLockup(ctx context.Context, req *types.UnlockingLockupRequest) (
	*types.UnlockingLockupResponse, error) {

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	delAddr, err := sdk.AccAddressFromBech32(req.DelegatorAddress)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid delegator address")
	}

	valAddr, err := sdk.ValAddressFromBech32(req.ValidatorAddress)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid validator address")
	}

	unlockingLockup := k.GetUnlockingLockup(ctx, delAddr, valAddr, req.CreationHeight)
	if unlockingLockup == nil {
		return nil, status.Error(codes.NotFound, "unlocking lockup does not exist")
	}

	return &types.UnlockingLockupResponse{UnlockingLockup: *unlockingLockup}, nil
}

// UnlockingLockup query returns all delegator unlocking lockups with pagination.
func (k *Keeper) UnlockingLockups(ctx context.Context, req *types.UnlockingLockupsRequest) (
	*types.UnlockingLockupsResponse, error) {

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	delAddr, err := sdk.AccAddressFromBech32(req.DelegatorAddress)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid delegator address")
	}

	lockups, pageRes, err := k.getUnlockingLockupsPaginated(ctx, delAddr, req.Pagination)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.UnlockingLockupsResponse{UnlockingLockups: lockups, Pagination: pageRes}, nil
}

// Developers query returns all registered developers.
func (k *Keeper) Developers(ctx context.Context, req *types.DevelopersRequest) (*types.DevelopersResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	developers, pageRes, err := k.getDevelopersPaginated(ctx, req.Pagination)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.DevelopersResponse{
		Developers: developers,
		Pagination: pageRes,
	}, nil
}

// UserSubscriptions query returns all user subscriptions for a specific developer.
func (k *Keeper) UserSubscriptions(ctx context.Context, req *types.UserSubscriptionsRequest) (*types.UserSubscriptionsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	developerAddr, err := sdk.AccAddressFromBech32(req.Developer)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid developer address")
	}

	userSubscriptions, pageRes, err := k.getUserSubscriptionsPaginated(ctx, developerAddr, req.Pagination)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.UserSubscriptionsResponse{
		UserSubscriptions: userSubscriptions,
		Pagination:        pageRes,
	}, nil
}

// getLockupsPaginated returns delegator lockups with pagination.
func (k *Keeper) getLockupsPaginated(ctx context.Context, delAddr sdk.AccAddress, page *query.PageRequest) (
	[]types.Lockup, *query.PageResponse, error) {

	var lockups []types.Lockup
	store := k.lockupStore(ctx, false)
	onResult := func(key []byte, value []byte) error {
		if !bytes.HasPrefix(key, delAddr.Bytes()) {
			return nil
		}
		var lockup types.Lockup
		k.cdc.MustUnmarshal(value, &lockup)
		lockups = append(lockups, lockup)
		return nil
	}

	pageRes, err := query.Paginate(store, page, onResult)
	if err != nil {
		return nil, nil, status.Error(codes.Internal, err.Error())
	}

	return lockups, pageRes, nil
}

// getUnlockingLockupsPaginated returns delegator unlocking lockups with pagination.
func (k *Keeper) getUnlockingLockupsPaginated(ctx context.Context, delAddr sdk.AccAddress, page *query.PageRequest) (
	[]types.UnlockingLockup, *query.PageResponse, error) {

	var unlockingLockups []types.UnlockingLockup
	store := k.lockupStore(ctx, true)
	onResult := func(key []byte, value []byte) error {
		if !bytes.HasPrefix(key, delAddr.Bytes()) {
			return nil
		}
		var unlockingLockup types.UnlockingLockup
		k.cdc.MustUnmarshal(value, &unlockingLockup)
		unlockingLockups = append(unlockingLockups, unlockingLockup)
		return nil
	}

	pageRes, err := query.Paginate(store, page, onResult)
	if err != nil {
		return nil, nil, status.Error(codes.Internal, err.Error())
	}

	return unlockingLockups, pageRes, nil
}

// getDevelopersPaginated returns all registered developers with pagination.
func (k *Keeper) getDevelopersPaginated(ctx context.Context, page *query.PageRequest) (
	[]types.Developer, *query.PageResponse, error) {

	var developers []types.Developer
	store := k.developerStore(ctx)

	onResult := func(key []byte, value []byte) error {
		var developer types.Developer
		k.cdc.MustUnmarshal(value, &developer)
		developers = append(developers, developer)
		return nil
	}

	pageRes, err := query.Paginate(store, page, onResult)
	if err != nil {
		return nil, nil, status.Error(codes.Internal, err.Error())
	}

	return developers, pageRes, nil
}

// getUserSubscriptionsPaginated returns user subscriptions for a developer with pagination.
func (k *Keeper) getUserSubscriptionsPaginated(ctx context.Context, developerAddr sdk.AccAddress, page *query.PageRequest) (
	[]types.UserSubscription, *query.PageResponse, error) {

	var userSubscriptions []types.UserSubscription
	store := k.userSubscriptionStore(ctx)

	onResult := func(key []byte, value []byte) error {
		if !bytes.HasPrefix(key, developerAddr.Bytes()) {
			return nil
		}
		var userSubscription types.UserSubscription
		k.cdc.MustUnmarshal(value, &userSubscription)
		userSubscriptions = append(userSubscriptions, userSubscription)
		return nil
	}

	pageRes, err := query.Paginate(store, page, onResult)
	if err != nil {
		return nil, nil, status.Error(codes.Internal, err.Error())
	}

	return userSubscriptions, pageRes, nil
}
