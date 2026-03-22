package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	appparams "github.com/sourcenetwork/sourcehub/app/params"
	"github.com/sourcenetwork/sourcehub/x/hub/types"
)

// InvalidateJWS handles the MsgInvalidateJWS message.
// It allows a user to invalidate a JWS token if:
// - The transaction has a JWS extension option with matching DID, or
// - The message creator matches the token's authorized account.
func (k *Keeper) InvalidateJWS(goCtx context.Context, req *types.MsgInvalidateJWS) (*types.MsgInvalidateJWSResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Get the JWS token record
	record, found, err := k.GetJWSToken(goCtx, req.TokenHash)
	if err != nil {
		return nil, errorsmod.Wrap(err, "decoding JWS token")
	}
	if !found {
		return nil, errorsmod.Wrapf(types.ErrJWSTokenNotFound, "token hash: %s", req.TokenHash)
	}

	// Check if token is already invalid
	if record.Status == types.JWSTokenStatus_STATUS_INVALID {
		return nil, errorsmod.Wrapf(types.ErrJWSTokenAlreadyInvalid, "token hash: %s", req.TokenHash)
	}

	// Get the DID from the context
	extractedDID := ""
	if did, ok := ctx.Value(appparams.ExtractedDIDContextKey).(string); ok {
		extractedDID = did
	}

	// Check authorization
	authorized := false

	if extractedDID != "" && extractedDID == record.IssuerDid {
		// Authorization via matching DID
		authorized = true
	} else if req.Creator == record.AuthorizedAccount {
		// Authorization via matching creator account
		authorized = true
	}

	if !authorized {
		return nil, errorsmod.Wrapf(
			types.ErrUnauthorizedInvalidation,
			"creator %s is not authorized to invalidate token for DID %s and account %s",
			req.Creator,
			record.IssuerDid,
			record.AuthorizedAccount,
		)
	}

	// Update token status to invalid
	if err := k.UpdateJWSTokenStatus(goCtx, req.TokenHash, types.JWSTokenStatus_STATUS_INVALID, req.Creator); err != nil {
		return nil, errorsmod.Wrap(types.ErrFailedToInvalidateToken, err.Error())
	}

	if err := ctx.EventManager().EmitTypedEvent(&types.EventJWSTokenInvalidated{
		TokenHash:         req.TokenHash,
		IssuerDid:         record.IssuerDid,
		AuthorizedAccount: record.AuthorizedAccount,
		InvalidatedBy:     req.Creator,
	}); err != nil {
		return nil, err
	}

	return &types.MsgInvalidateJWSResponse{Success: true}, nil
}
