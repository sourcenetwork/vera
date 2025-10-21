package keeper

import (
	"context"

	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/sourcenetwork/sourcehub/x/hub/types"
	hubtypes "github.com/sourcenetwork/sourcehub/x/hub/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ types.QueryServer = &Keeper{}

// recordToInfo converts a JWSTokenRecord to JWSTokenInfo.
// This removes the sensitive bearer_token field before returning to queries.
func recordToInfo(record *hubtypes.JWSTokenRecord) *hubtypes.JWSTokenInfo {
	if record == nil {
		return nil
	}
	return &hubtypes.JWSTokenInfo{
		TokenHash:         record.TokenHash,
		IssuerDid:         record.IssuerDid,
		AuthorizedAccount: record.AuthorizedAccount,
		IssuedAt:          record.IssuedAt,
		ExpiresAt:         record.ExpiresAt,
		Status:            record.Status,
		FirstUsedAt:       record.FirstUsedAt,
		LastUsedAt:        record.LastUsedAt,
		InvalidatedAt:     record.InvalidatedAt,
		InvalidatedBy:     record.InvalidatedBy,
	}
}

// JWSToken returns a specific JWS token by hash.
func (k *Keeper) JWSToken(goCtx context.Context, req *hubtypes.QueryJWSTokenRequest) (*hubtypes.QueryJWSTokenResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.TokenHash == "" {
		return nil, status.Error(codes.InvalidArgument, "token hash cannot be empty")
	}

	record, found := k.GetJWSToken(goCtx, req.TokenHash)
	if !found {
		return nil, status.Errorf(codes.NotFound, "JWS token not found: %s", req.TokenHash)
	}

	// Convert to JWSTokenInfo (removes bearer_token)
	tokenInfo := recordToInfo(record)

	return &hubtypes.QueryJWSTokenResponse{Token: tokenInfo}, nil
}

// JWSTokensByDID returns all JWS tokens for a specific DID.
func (k *Keeper) JWSTokensByDID(goCtx context.Context, req *hubtypes.QueryJWSTokensByDIDRequest) (*hubtypes.QueryJWSTokensByDIDResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.Did == "" {
		return nil, status.Error(codes.InvalidArgument, "DID cannot be empty")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	store := k.jwsTokenByDIDStore(ctx)
	didPrefix := hubtypes.JWSTokenDIDPrefix(req.Did)
	prefixWithoutMain := didPrefix[len(hubtypes.JWSTokenByDIDKeyPrefix):]
	didStore := prefix.NewStore(store, prefixWithoutMain)

	var tokenInfos []*hubtypes.JWSTokenInfo
	pageRes, err := query.Paginate(didStore, req.Pagination, func(key []byte, value []byte) error {
		// Extract token hash from the key
		keyStr := string(key)
		lastSlash := -1
		for i := len(keyStr) - 1; i >= 0; i-- {
			if keyStr[i] == '/' {
				lastSlash = i
				break
			}
		}
		if lastSlash == -1 {
			return nil
		}
		tokenHash := keyStr[lastSlash+1:]

		// Get the actual record from primary store
		record, found := k.GetJWSToken(goCtx, tokenHash)
		if found {
			// Convert to JWSTokenInfo (removes bearer_token)
			tokenInfos = append(tokenInfos, recordToInfo(record))
		}
		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &hubtypes.QueryJWSTokensByDIDResponse{
		Tokens:     tokenInfos,
		Pagination: pageRes,
	}, nil
}

// JWSTokensByAccount returns all JWS tokens for a specific authorized account.
func (k *Keeper) JWSTokensByAccount(goCtx context.Context, req *hubtypes.QueryJWSTokensByAccountRequest) (*hubtypes.QueryJWSTokensByAccountResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.Account == "" {
		return nil, status.Error(codes.InvalidArgument, "account cannot be empty")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	store := k.jwsTokenByAccountStore(ctx)
	accountPrefix := hubtypes.JWSTokenAccountPrefix(req.Account)
	prefixWithoutMain := accountPrefix[len(hubtypes.JWSTokenByAccountKeyPrefix):]
	accountStore := prefix.NewStore(store, prefixWithoutMain)

	var tokenInfos []*hubtypes.JWSTokenInfo
	pageRes, err := query.Paginate(accountStore, req.Pagination, func(key []byte, value []byte) error {
		// Extract token hash from the key
		keyStr := string(key)
		lastSlash := -1
		for i := len(keyStr) - 1; i >= 0; i-- {
			if keyStr[i] == '/' {
				lastSlash = i
				break
			}
		}
		if lastSlash == -1 {
			return nil
		}
		tokenHash := keyStr[lastSlash+1:]

		// Get the actual record from primary store
		record, found := k.GetJWSToken(goCtx, tokenHash)
		if found {
			// Convert to JWSTokenInfo (removes bearer_token)
			tokenInfos = append(tokenInfos, recordToInfo(record))
		}
		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &hubtypes.QueryJWSTokensByAccountResponse{
		Tokens:     tokenInfos,
		Pagination: pageRes,
	}, nil
}

// AllJWSTokens returns all JWS tokens with pagination.
func (k *Keeper) AllJWSTokens(goCtx context.Context, req *hubtypes.QueryAllJWSTokensRequest) (*hubtypes.QueryAllJWSTokensResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	store := k.jwsTokenStore(ctx)

	var tokenInfos []*hubtypes.JWSTokenInfo
	pageRes, err := query.Paginate(store, req.Pagination, func(key []byte, value []byte) error {
		var record hubtypes.JWSTokenRecord
		if err := k.cdc.Unmarshal(value, &record); err != nil {
			return err
		}
		// Convert to JWSTokenInfo (removes bearer_token)
		tokenInfos = append(tokenInfos, recordToInfo(&record))
		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &hubtypes.QueryAllJWSTokensResponse{
		Tokens:     tokenInfos,
		Pagination: pageRes,
	}, nil
}
