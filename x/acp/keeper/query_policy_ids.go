package keeper

import (
	"context"
	"encoding/binary"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/sourcenetwork/vera/utils"
	"github.com/sourcenetwork/vera/x/acp/types"
)

func (k *Keeper) PolicyIds(goCtx context.Context, req *types.QueryPolicyIdsRequest) (*types.QueryPolicyIdsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	engine := k.getACPEngine(ctx)

	resp, err := engine.ListPolicies(ctx, &coretypes.ListPoliciesRequest{})
	if err != nil {
		return nil, err
	}

	// Use MapNullableSlice to filter out 'nil' policies and get policy IDs
	allPolicyIds := utils.MapNullableSlice(resp.Records, func(p *coretypes.PolicyRecord) string { return p.Policy.Id })

	// Apply pagination
	policyIds, pageRes := paginateSlice(allPolicyIds, req.Pagination)

	return &types.QueryPolicyIdsResponse{
		Ids:        policyIds,
		Pagination: pageRes,
	}, nil
}

// paginateSlice applies pagination to a slice of strings following Cosmos SDK patterns.
func paginateSlice(items []string, pageReq *query.PageRequest) ([]string, *query.PageResponse) {
	total := uint64(len(items))

	if pageReq == nil {
		return items, &query.PageResponse{Total: total}
	}

	// Handle key-based pagination
	offset := pageReq.Offset
	if len(pageReq.Key) > 0 {
		// Decode offset from key (uint64 = 8 bytes)
		if len(pageReq.Key) != 8 {
			return []string{}, &query.PageResponse{
				Total: total,
			}
		}
		offset = binary.BigEndian.Uint64(pageReq.Key)
	}

	// Determine limit
	limit := pageReq.Limit
	if limit == 0 {
		if offset > 0 {
			limit = query.DefaultLimit
		} else {
			return items, &query.PageResponse{Total: total}
		}
	}
	if limit > query.PaginationMaxLimit {
		limit = query.PaginationMaxLimit
	}

	// Validate offset
	if offset >= total {
		return []string{}, &query.PageResponse{Total: total}
	}

	// Calculate end index
	end := offset + limit
	if end > total {
		end = total
	}

	result := items[offset:end]

	// Encode next key using proper uint64 encoding
	var nextKey []byte
	if end < total {
		nextKey = make([]byte, 8)
		binary.BigEndian.PutUint64(nextKey, end)
	}

	return result, &query.PageResponse{
		NextKey: nextKey,
		Total:   total,
	}
}
