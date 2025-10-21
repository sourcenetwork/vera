package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	appparams "github.com/sourcenetwork/sourcehub/app/params"
)

// GetActorDID returns the actor DID for the current transaction.
// It prioritizes the DID extracted from the extension option bearer token if present,
// otherwise falls back to issuing a DID from the provided account address.
func (k *Keeper) GetActorDID(ctx sdk.Context, accountAddr string) (string, error) {
	// Check if a DID was extracted from the extension option bearer token
	if extractedDID, ok := ctx.Value(appparams.ExtractedDIDContextKey).(string); ok && extractedDID != "" {
		return extractedDID, nil
	}

	// Fall back to issuing DID from account address
	return k.IssueDIDFromAccountAddr(ctx, accountAddr)
}
