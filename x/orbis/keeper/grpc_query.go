package keeper

import (
	"context"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/sourcenetwork/sourcehub/x/orbis/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ types.QueryServer = &Keeper{}

// Params query returns orbis module params.
func (k *Keeper) Params(ctx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	return &types.QueryParamsResponse{Params: k.GetParams(ctx)}, nil
}

func (k *Keeper) Ring(ctx context.Context, req *types.QueryRingRequest) (*types.QueryRingResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, types.ErrInvalidRingId.Error())
	}

	ring := k.GetRing(ctx, req.Id)
	if ring == nil {
		return nil, status.Error(codes.NotFound, types.ErrRingNotFound.Error())
	}

	return &types.QueryRingResponse{Ring: ring}, nil
}

func (k *Keeper) Rings(ctx context.Context, req *types.QueryRingsRequest) (*types.QueryRingsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.RingKeyPrefix))
	var rings []types.Ring

	pageRes, err := query.Paginate(store, req.Pagination, func(key []byte, value []byte) error {
		var ring types.Ring
		k.cdc.MustUnmarshal(value, &ring)
		rings = append(rings, ring)
		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryRingsResponse{Rings: rings, Pagination: pageRes}, nil
}

func (k *Keeper) Document(ctx context.Context, req *types.QueryDocumentRequest) (*types.QueryDocumentResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, types.ErrInvalidDocumentId.Error())
	}

	document := k.GetDocument(ctx, req.Id)
	if document == nil {
		return nil, status.Error(codes.NotFound, types.ErrDocumentNotFound.Error())
	}

	return &types.QueryDocumentResponse{Document: document}, nil
}

func (k *Keeper) Documents(ctx context.Context, req *types.QueryDocumentsRequest) (*types.QueryDocumentsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.DocumentKeyPrefix))
	var documents []types.Document

	pageRes, err := query.Paginate(store, req.Pagination, func(key []byte, value []byte) error {
		var document types.Document
		k.cdc.MustUnmarshal(value, &document)
		documents = append(documents, document)
		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryDocumentsResponse{Documents: documents, Pagination: pageRes}, nil
}

func (k *Keeper) KeyDerivation(ctx context.Context, req *types.QueryKeyDerivationRequest) (*types.QueryKeyDerivationResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, types.ErrInvalidKeyDerivationId.Error())
	}

	keyDerivation := k.GetKeyDerivation(ctx, req.Id)
	if keyDerivation == nil {
		return nil, status.Error(codes.NotFound, types.ErrKeyDerivationNotFound.Error())
	}

	return &types.QueryKeyDerivationResponse{KeyDerivation: keyDerivation}, nil
}

func (k *Keeper) NodeInfo(ctx context.Context, req *types.QueryNodeInfoRequest) (*types.QueryNodeInfoResponse, error) {
	if req == nil || req.NodeKey == "" {
		return nil, status.Error(codes.InvalidArgument, "node_key is required")
	}

	nodeInfo := k.GetNodeInfo(ctx, req.NodeKey)
	if nodeInfo == nil {
		return nil, status.Error(codes.NotFound, types.ErrNodeInfoNotFound.Error())
	}

	return &types.QueryNodeInfoResponse{NodeInfo: nodeInfo}, nil
}

func (k *Keeper) NodeDemerits(ctx context.Context, req *types.QueryNodeDemeritsRequest) (*types.QueryNodeDemeritsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	if req.RingId == "" {
		return nil, status.Error(codes.InvalidArgument, types.ErrInvalidRingId.Error())
	}
	if req.NodeKey == "" {
		return nil, status.Error(codes.InvalidArgument, "node_key is required")
	}
	ring := k.GetRing(ctx, req.RingId)
	if ring == nil {
		return nil, status.Error(codes.NotFound, types.ErrRingNotFound.Error())
	}
	now, err := reportBlockUnixTime(sdk.UnwrapSDKContext(ctx))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryNodeDemeritsResponse{
		Points: k.GetEffectiveNodeDemerits(ctx, req.RingId, req.NodeKey, now, ring.Reporting.DemeritConfig.ResetIntervalSeconds),
	}, nil
}

func (k *Keeper) KeyDerivations(ctx context.Context, req *types.QueryKeyDerivationsRequest) (*types.QueryKeyDerivationsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.KeyDerivationKeyPrefix))
	var keyDerivations []types.KeyDerivation

	pageRes, err := query.Paginate(store, req.Pagination, func(key []byte, value []byte) error {
		var keyDerivation types.KeyDerivation
		k.cdc.MustUnmarshal(value, &keyDerivation)
		keyDerivations = append(keyDerivations, keyDerivation)
		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryKeyDerivationsResponse{KeyDerivations: keyDerivations, Pagination: pageRes}, nil
}
