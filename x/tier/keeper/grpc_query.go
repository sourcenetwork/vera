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

type Querier struct {
	Keeper
}

var _ types.QueryServer = Querier{}

func NewQuerier(keeper Keeper) Querier {
	return Querier{Keeper: keeper}
}

func (q Querier) Params(ctx context.Context, req *types.QueryParamsRequest) (
	*types.QueryParamsResponse, error) {

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	return &types.QueryParamsResponse{Params: q.GetParams(ctx)}, nil
}

func (q Querier) Lockup(ctx context.Context, req *types.LockupRequest) (
	*types.LockupResponse, error) {

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

	lockup := q.GetLockup(ctx, delAddr, valAddr)
	if lockup == nil {
		return nil, status.Error(codes.NotFound, "unlocking lockup does not exist")
	}

	return &types.LockupResponse{Lockup: *lockup}, nil
}

func (q Querier) Lockups(ctx context.Context, req *types.LockupsRequest) (
	*types.LockupsResponse, error) {

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	delAddr, err := sdk.AccAddressFromBech32(req.DelegatorAddress)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid delegator address")
	}

	lockups, pageRes, err := q.getLockupsPaginated(ctx, delAddr, req.Pagination)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.LockupsResponse{Lockups: lockups, Pagination: pageRes}, nil
}

func (q Querier) UnlockingLockup(ctx context.Context, req *types.UnlockingLockupRequest) (
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

	unlockingLockup := q.GetUnlockingLockup(ctx, delAddr, valAddr, req.CreationHeight)
	if unlockingLockup == nil {
		return nil, status.Error(codes.NotFound, "unlocking lockup does not exist")
	}

	return &types.UnlockingLockupResponse{UnlockingLockup: *unlockingLockup}, nil
}

func (q Querier) UnlockingLockups(ctx context.Context, req *types.UnlockingLockupsRequest) (
	*types.UnlockingLockupsResponse, error) {

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	delAddr, err := sdk.AccAddressFromBech32(req.DelegatorAddress)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid delegator address")
	}

	lockups, pageRes, err := q.getUnlockingLockupsPaginated(ctx, delAddr, req.Pagination)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.UnlockingLockupsResponse{UnlockingLockups: lockups, Pagination: pageRes}, nil
}

func (q Querier) getLockupsPaginated(ctx context.Context, delAddr sdk.AccAddress, page *query.PageRequest) (
	[]types.Lockup, *query.PageResponse, error) {

	var lockups []types.Lockup
	store := q.lockupStore(ctx, false)
	onResult := func(key []byte, value []byte) error {
		if !bytes.HasPrefix(key, delAddr.Bytes()) {
			return nil
		}
		var lockup types.Lockup
		q.cdc.MustUnmarshal(value, &lockup)
		lockups = append(lockups, lockup)
		return nil
	}

	pageRes, err := query.Paginate(store, page, onResult)
	if err != nil {
		return nil, nil, status.Error(codes.Internal, err.Error())
	}

	return lockups, pageRes, nil
}

func (q Querier) getUnlockingLockupsPaginated(ctx context.Context, delAddr sdk.AccAddress, page *query.PageRequest) (
	[]types.UnlockingLockup, *query.PageResponse, error) {

	var unlockingLockups []types.UnlockingLockup
	store := q.lockupStore(ctx, true)
	onResult := func(key []byte, value []byte) error {
		if !bytes.HasPrefix(key, delAddr.Bytes()) {
			return nil
		}
		var unlockingLockup types.UnlockingLockup
		q.cdc.MustUnmarshal(value, &unlockingLockup)
		unlockingLockups = append(unlockingLockups, unlockingLockup)
		return nil
	}

	pageRes, err := query.Paginate(store, page, onResult)
	if err != nil {
		return nil, nil, status.Error(codes.Internal, err.Error())
	}

	return unlockingLockups, pageRes, nil
}
