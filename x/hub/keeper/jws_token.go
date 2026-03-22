package keeper

import (
	"context"
	"fmt"
	"time"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/sourcenetwork/sourcehub/x/hub/types"
)

// jwsTokenStore returns the prefix store for JWS tokens.
func (k *Keeper) jwsTokenStore(ctx context.Context) prefix.Store {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	return prefix.NewStore(storeAdapter, types.JWSTokenKeyPrefix)
}

// jwsTokenByDIDStore returns the prefix store for JWS tokens indexed by DID.
func (k *Keeper) jwsTokenByDIDStore(ctx context.Context) prefix.Store {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	return prefix.NewStore(storeAdapter, types.JWSTokenByDIDKeyPrefix)
}

// jwsTokenByAccountStore returns the prefix store for JWS tokens indexed by account.
func (k *Keeper) jwsTokenByAccountStore(ctx context.Context) prefix.Store {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	return prefix.NewStore(storeAdapter, types.JWSTokenByAccountKeyPrefix)
}

// SetJWSToken stores a JWS token record and updates secondary indices.
func (k *Keeper) SetJWSToken(ctx context.Context, record *types.JWSTokenRecord) error {
	if record == nil {
		return errorsmod.Wrap(types.ErrInvalidInput, "JWS token record cannot be nil")
	}

	if record.TokenHash == "" {
		return errorsmod.Wrap(types.ErrInvalidInput, "token hash cannot be empty")
	}

	if record.IssuerDid == "" {
		return errorsmod.Wrap(types.ErrInvalidInput, "issuer DID cannot be empty")
	}

	// Validate authorized account if provided
	if record.AuthorizedAccount != "" {
		if _, err := sdk.AccAddressFromBech32(record.AuthorizedAccount); err != nil {
			return errorsmod.Wrap(types.ErrInvalidInput, "invalid authorized account address: "+err.Error())
		}
	} else if !k.GetChainConfig(ctx).IgnoreBearerAuth {
		return errorsmod.Wrap(types.ErrInvalidInput, "authorized account is required when bearer auth is enabled")
	}

	bz, err := k.cdc.Marshal(record)
	if err != nil {
		return fmt.Errorf("failed to marshal JWS token record: %w", err)
	}

	store := k.jwsTokenStore(ctx)
	store.Set([]byte(record.TokenHash), bz)

	// Store in DID index
	didStore := k.jwsTokenByDIDStore(ctx)
	didKey := types.JWSTokenByDIDKey(record.IssuerDid, record.TokenHash)
	// Remove the prefix since we're already in the DID prefix store
	didKeyWithoutPrefix := didKey[len(types.JWSTokenByDIDKeyPrefix):]
	didStore.Set(didKeyWithoutPrefix, []byte{0x01}) // Just a marker, actual data is in primary store

	// Store in account index only if authorized account is provided
	if record.AuthorizedAccount != "" {
		accountStore := k.jwsTokenByAccountStore(ctx)
		accountKey := types.JWSTokenByAccountKey(record.AuthorizedAccount, record.TokenHash)
		// Remove the prefix since we're already in the account prefix store
		accountKeyWithoutPrefix := accountKey[len(types.JWSTokenByAccountKeyPrefix):]
		accountStore.Set(accountKeyWithoutPrefix, []byte{0x01}) // Just a marker
	}

	return nil
}

// GetJWSToken retrieves a JWS token record by its hash.
// Returns the record and true if found, or (nil, false, nil) if not found.
// Returns a non-nil error if the record exists but cannot be decoded.
func (k *Keeper) GetJWSToken(ctx context.Context, tokenHash string) (*types.JWSTokenRecord, bool, error) {
	store := k.jwsTokenStore(ctx)
	bz := store.Get([]byte(tokenHash))
	if bz == nil {
		return nil, false, nil
	}

	var record types.JWSTokenRecord
	if err := k.cdc.Unmarshal(bz, &record); err != nil {
		return nil, false, fmt.Errorf("failed to unmarshal JWS token record %s: %w", tokenHash, err)
	}

	return &record, true, nil
}

// DeleteJWSToken removes a JWS token record and its indices.
func (k *Keeper) DeleteJWSToken(ctx context.Context, tokenHash string) error {
	// First, get the record to access DID and account for index cleanup
	record, found, err := k.GetJWSToken(ctx, tokenHash)
	if err != nil {
		return errorsmod.Wrap(err, "decoding JWS token")
	}
	if !found {
		return errorsmod.Wrap(types.ErrJWSTokenNotFound, tokenHash)
	}

	// Delete from primary store
	store := k.jwsTokenStore(ctx)
	store.Delete([]byte(tokenHash))

	// Delete from DID index
	didStore := k.jwsTokenByDIDStore(ctx)
	didKey := types.JWSTokenByDIDKey(record.IssuerDid, tokenHash)
	didKeyWithoutPrefix := didKey[len(types.JWSTokenByDIDKeyPrefix):]
	didStore.Delete(didKeyWithoutPrefix)

	// Delete from account index only if authorized account was provided
	if record.AuthorizedAccount != "" {
		accountStore := k.jwsTokenByAccountStore(ctx)
		accountKey := types.JWSTokenByAccountKey(record.AuthorizedAccount, tokenHash)
		accountKeyWithoutPrefix := accountKey[len(types.JWSTokenByAccountKeyPrefix):]
		accountStore.Delete(accountKeyWithoutPrefix)
	}

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
	prefixWithoutMain := prefix[len(types.JWSTokenByDIDKeyPrefix):]

	iterator := storetypes.KVStorePrefixIterator(didStore, prefixWithoutMain)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		_, tokenHash, err := types.ParseJWSTokenByDIDKey(iterator.Key())
		if err != nil {
			k.Logger().Error("failed to parse JWS token by DID key", "error", err)
			continue
		}

		record, found, err := k.GetJWSToken(ctx, tokenHash)
		if err != nil {
			return nil, err
		}
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
	prefixWithoutMain := prefix[len(types.JWSTokenByAccountKeyPrefix):]

	iterator := storetypes.KVStorePrefixIterator(accountStore, prefixWithoutMain)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		_, tokenHash, err := types.ParseJWSTokenByAccountKey(iterator.Key())
		if err != nil {
			k.Logger().Error("failed to parse JWS token by account key", "error", err)
			continue
		}

		record, found, err := k.GetJWSToken(ctx, tokenHash)
		if err != nil {
			return nil, err
		}
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
	if tokenHash == "" {
		return errorsmod.Wrap(types.ErrInvalidInput, "token hash cannot be empty")
	}

	record, found, err := k.GetJWSToken(ctx, tokenHash)
	if err != nil {
		return errorsmod.Wrap(err, "decoding JWS token")
	}
	if !found {
		return errorsmod.Wrap(types.ErrJWSTokenNotFound, tokenHash)
	}

	record.Status = status

	if status == types.JWSTokenStatus_STATUS_INVALID {
		blockTime := sdk.UnwrapSDKContext(ctx).BlockTime()
		record.InvalidatedAt = &blockTime
		if invalidatedBy != "" {
			record.InvalidatedBy = invalidatedBy
		}
	}

	return k.SetJWSToken(ctx, record)
}

// RecordJWSTokenUsage updates the last used timestamp for a JWS token.
func (k *Keeper) RecordJWSTokenUsage(ctx context.Context, tokenHash string) error {
	if tokenHash == "" {
		return errorsmod.Wrap(types.ErrInvalidInput, "token hash cannot be empty")
	}

	record, found, err := k.GetJWSToken(ctx, tokenHash)
	if err != nil {
		return errorsmod.Wrap(err, "decoding JWS token")
	}
	if !found {
		return errorsmod.Wrap(types.ErrJWSTokenNotFound, tokenHash)
	}

	blockTime := sdk.UnwrapSDKContext(ctx).BlockTime()
	if record.FirstUsedAt == nil {
		record.FirstUsedAt = &blockTime
	}
	record.LastUsedAt = &blockTime

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
		if !record.ExpiresAt.After(currentTime) {
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
	_, found, err := k.GetJWSToken(ctx, tokenHash)
	if err != nil {
		return errorsmod.Wrap(err, "decoding JWS token")
	}
	if found {
		// Token exists, update usage timestamp
		return k.RecordJWSTokenUsage(ctx, tokenHash)
	}

	// Validate that new tokens aren't already expired
	if !expiresAt.IsZero() && !expiresAt.After(sdk.UnwrapSDKContext(ctx).BlockTime()) {
		return errorsmod.Wrap(types.ErrJWSTokenExpired, "cannot create token with expiration time in the past")
	}

	// Create new token record
	blockTime := sdk.UnwrapSDKContext(ctx).BlockTime()
	record := &types.JWSTokenRecord{
		TokenHash:         tokenHash,
		BearerToken:       bearerToken,
		IssuerDid:         issuerDid,
		AuthorizedAccount: authorizedAccount,
		IssuedAt:          issuedAt,
		ExpiresAt:         expiresAt,
		Status:            types.JWSTokenStatus_STATUS_VALID,
		FirstUsedAt:       &blockTime,
		LastUsedAt:        &blockTime,
	}

	return k.SetJWSToken(ctx, record)
}
