package keeper

import (
	"context"
	"fmt"
	"time"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/sourcenetwork/sourcehub/x/hub/types"
)

// jwsTokenStore returns the prefix store for JWS tokens.
func (k *Keeper) jwsTokenStore(ctx context.Context) prefix.Store {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	return prefix.NewStore(storeAdapter, []byte(types.JWSTokenKeyPrefix))
}

// jwsTokenByDIDStore returns the prefix store for JWS tokens indexed by DID.
func (k *Keeper) jwsTokenByDIDStore(ctx context.Context) prefix.Store {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	return prefix.NewStore(storeAdapter, []byte(types.JWSTokenByDIDKeyPrefix))
}

// jwsTokenByAccountStore returns the prefix store for JWS tokens indexed by account.
func (k *Keeper) jwsTokenByAccountStore(ctx context.Context) prefix.Store {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	return prefix.NewStore(storeAdapter, []byte(types.JWSTokenByAccountKeyPrefix))
}

// SetJWSToken stores a JWS token record and updates secondary indices.
func (k *Keeper) SetJWSToken(ctx context.Context, record *types.JWSTokenRecord) error {
	if record == nil {
		return fmt.Errorf("JWS token record cannot be nil")
	}

	// Marshal the record
	bz, err := k.cdc.Marshal(record)
	if err != nil {
		return fmt.Errorf("failed to marshal JWS token record: %w", err)
	}

	// Store in primary index (by token hash)
	store := k.jwsTokenStore(ctx)
	store.Set([]byte(record.TokenHash), bz)

	// Store in DID index
	didStore := k.jwsTokenByDIDStore(ctx)
	didKey := types.JWSTokenByDIDKey(record.IssuerDid, record.TokenHash)
	// Remove the prefix since we're already in the DID prefix store
	didKeyWithoutPrefix := didKey[len(types.JWSTokenByDIDKeyPrefix):]
	didStore.Set(didKeyWithoutPrefix, []byte{0x01}) // Just a marker, actual data is in primary store

	// Store in account index
	accountStore := k.jwsTokenByAccountStore(ctx)
	accountKey := types.JWSTokenByAccountKey(record.AuthorizedAccount, record.TokenHash)
	// Remove the prefix since we're already in the account prefix store
	accountKeyWithoutPrefix := accountKey[len(types.JWSTokenByAccountKeyPrefix):]
	accountStore.Set(accountKeyWithoutPrefix, []byte{0x01}) // Just a marker

	return nil
}

// GetJWSToken retrieves a JWS token record by its hash.
func (k *Keeper) GetJWSToken(ctx context.Context, tokenHash string) (*types.JWSTokenRecord, bool) {
	store := k.jwsTokenStore(ctx)
	bz := store.Get([]byte(tokenHash))
	if bz == nil {
		return nil, false
	}

	var record types.JWSTokenRecord
	if err := k.cdc.Unmarshal(bz, &record); err != nil {
		k.Logger().Error("failed to unmarshal JWS token record", "hash", tokenHash, "error", err)
		return nil, false
	}

	return &record, true
}

// DeleteJWSToken removes a JWS token record and its indices.
func (k *Keeper) DeleteJWSToken(ctx context.Context, tokenHash string) error {
	// First, get the record to access DID and account for index cleanup
	record, found := k.GetJWSToken(ctx, tokenHash)
	if !found {
		return fmt.Errorf("JWS token not found: %s", tokenHash)
	}

	// Delete from primary store
	store := k.jwsTokenStore(ctx)
	store.Delete([]byte(tokenHash))

	// Delete from DID index
	didStore := k.jwsTokenByDIDStore(ctx)
	didKey := types.JWSTokenByDIDKey(record.IssuerDid, tokenHash)
	didKeyWithoutPrefix := didKey[len(types.JWSTokenByDIDKeyPrefix):]
	didStore.Delete(didKeyWithoutPrefix)

	// Delete from account index
	accountStore := k.jwsTokenByAccountStore(ctx)
	accountKey := types.JWSTokenByAccountKey(record.AuthorizedAccount, tokenHash)
	accountKeyWithoutPrefix := accountKey[len(types.JWSTokenByAccountKeyPrefix):]
	accountStore.Delete(accountKeyWithoutPrefix)

	return nil
}

// IterateJWSTokens iterates over all JWS token records.
func (k *Keeper) IterateJWSTokens(ctx context.Context, cb func(record *types.JWSTokenRecord) (stop bool)) error {
	store := k.jwsTokenStore(ctx)
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var record types.JWSTokenRecord
		if err := k.cdc.Unmarshal(iterator.Value(), &record); err != nil {
			return fmt.Errorf("failed to unmarshal JWS token record: %w", err)
		}

		if cb(&record) {
			break
		}
	}

	return nil
}

// GetJWSTokensByDID retrieves all JWS tokens for a specific DID.
func (k *Keeper) GetJWSTokensByDID(ctx context.Context, did string) ([]*types.JWSTokenRecord, error) {
	var records []*types.JWSTokenRecord

	didStore := k.jwsTokenByDIDStore(ctx)
	prefix := types.JWSTokenDIDPrefix(did)
	// Remove the main prefix since we're already in the DID prefix store
	prefixWithoutMain := prefix[len(types.JWSTokenByDIDKeyPrefix):]

	iterator := storetypes.KVStorePrefixIterator(didStore, prefixWithoutMain)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		// Extract token hash from the key
		// Key format: "did/tokenHash"
		key := string(iterator.Key())
		// Find the last '/' to extract token hash
		lastSlash := -1
		for i := len(key) - 1; i >= 0; i-- {
			if key[i] == '/' {
				lastSlash = i
				break
			}
		}
		if lastSlash == -1 {
			continue
		}
		tokenHash := key[lastSlash+1:]

		// Get the actual record from primary store
		record, found := k.GetJWSToken(ctx, tokenHash)
		if found {
			records = append(records, record)
		}
	}

	return records, nil
}

// GetJWSTokensByAccount retrieves all JWS tokens for a specific authorized account.
func (k *Keeper) GetJWSTokensByAccount(ctx context.Context, account string) ([]*types.JWSTokenRecord, error) {
	var records []*types.JWSTokenRecord

	accountStore := k.jwsTokenByAccountStore(ctx)
	prefix := types.JWSTokenAccountPrefix(account)
	// Remove the main prefix since we're already in the account prefix store
	prefixWithoutMain := prefix[len(types.JWSTokenByAccountKeyPrefix):]

	iterator := storetypes.KVStorePrefixIterator(accountStore, prefixWithoutMain)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		// Extract token hash from the key
		// Key format: "account/tokenHash"
		key := string(iterator.Key())
		// Find the last '/' to extract token hash
		lastSlash := -1
		for i := len(key) - 1; i >= 0; i-- {
			if key[i] == '/' {
				lastSlash = i
				break
			}
		}
		if lastSlash == -1 {
			continue
		}
		tokenHash := key[lastSlash+1:]

		// Get the actual record from primary store
		record, found := k.GetJWSToken(ctx, tokenHash)
		if found {
			records = append(records, record)
		}
	}

	return records, nil
}

// GetAllJWSTokens retrieves all JWS token records.
func (k *Keeper) GetAllJWSTokens(ctx context.Context) ([]*types.JWSTokenRecord, error) {
	var records []*types.JWSTokenRecord

	err := k.IterateJWSTokens(ctx, func(record *types.JWSTokenRecord) bool {
		records = append(records, record)
		return false
	})

	return records, err
}

// UpdateJWSTokenStatus updates the status of a JWS token.
func (k *Keeper) UpdateJWSTokenStatus(ctx context.Context, tokenHash string, status types.JWSTokenStatus, invalidatedBy string) error {
	record, found := k.GetJWSToken(ctx, tokenHash)
	if !found {
		return fmt.Errorf("JWS token not found: %s", tokenHash)
	}

	record.Status = status

	if status == types.JWSTokenStatus_STATUS_INVALID {
		now := time.Now()
		record.InvalidatedAt = &now
		if invalidatedBy != "" {
			record.InvalidatedBy = invalidatedBy
		}
	}

	return k.SetJWSToken(ctx, record)
}

// RecordJWSTokenUsage updates the last used timestamp for a JWS token.
func (k *Keeper) RecordJWSTokenUsage(ctx context.Context, tokenHash string) error {
	record, found := k.GetJWSToken(ctx, tokenHash)
	if !found {
		return fmt.Errorf("JWS token not found: %s", tokenHash)
	}

	now := time.Now()
	if record.FirstUsedAt == nil {
		record.FirstUsedAt = &now
	}
	record.LastUsedAt = &now

	return k.SetJWSToken(ctx, record)
}

// CheckAndUpdateExpiredTokens iterates through all tokens and marks expired ones as invalid.
func (k *Keeper) CheckAndUpdateExpiredTokens(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentTime := sdkCtx.BlockTime()

	return k.IterateJWSTokens(ctx, func(record *types.JWSTokenRecord) bool {
		// Skip already invalid tokens
		if record.Status == types.JWSTokenStatus_STATUS_INVALID {
			return false
		}

		// Check if token is expired
		if record.ExpiresAt.Before(currentTime) {
			// Mark as invalid
			if err := k.UpdateJWSTokenStatus(ctx, record.TokenHash, types.JWSTokenStatus_STATUS_INVALID, ""); err != nil {
				k.Logger().Error("failed to update expired token status", "hash", record.TokenHash, "error", err)
			}
		}

		return false
	})
}

// StoreOrUpdateJWSToken stores a new JWS token or updates an existing one.
// This is the main method called from the ante handler.
func (k *Keeper) StoreOrUpdateJWSToken(
	ctx context.Context,
	bearerToken string,
	issuerDid string,
	authorizedAccount string,
	issuedAt time.Time,
	expiresAt time.Time,
) error {
	tokenHash := types.HashJWSToken(bearerToken)

	// Check if token already exists
	_, found := k.GetJWSToken(ctx, tokenHash)
	if found {
		// Token exists, update usage timestamp
		return k.RecordJWSTokenUsage(ctx, tokenHash)
	}

	// Create new token record
	now := time.Now()
	record := &types.JWSTokenRecord{
		TokenHash:         tokenHash,
		BearerToken:       bearerToken,
		IssuerDid:         issuerDid,
		AuthorizedAccount: authorizedAccount,
		IssuedAt:          issuedAt,
		ExpiresAt:         expiresAt,
		Status:            types.JWSTokenStatus_STATUS_VALID,
		FirstUsedAt:       &now,
		LastUsedAt:        &now,
	}

	return k.SetJWSToken(ctx, record)
}
