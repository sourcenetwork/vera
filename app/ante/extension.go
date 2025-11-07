package ante

import (
	"time"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"

	antetypes "github.com/sourcenetwork/sourcehub/app/ante/types"
	appparams "github.com/sourcenetwork/sourcehub/app/params"
)

// ExtensionOptionsDecorator validates extension options in transactions.
// It allows JWSExtensionOption and rejects all other extension options.
// It also extracts DID from JWS extension options and stores it in context.
// Additionally, it stores used JWS tokens for tracking and invalidation.
type ExtensionOptionsDecorator struct {
	hubKeeper HubKeeper
}

// NewExtensionOptionsDecorator creates a new ExtensionOptionsDecorator.
func NewExtensionOptionsDecorator(hubKeeper HubKeeper) ExtensionOptionsDecorator {
	return ExtensionOptionsDecorator{
		hubKeeper: hubKeeper,
	}
}

// AnteHandle validates extension options, allowing only JWSExtensionOption.
// It extracts DID from JWS extension options and stores it in context for later use.
func (eod ExtensionOptionsDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	// Check if the transaction has extension options
	hasExtOptsTx, ok := tx.(ante.HasExtensionOptionsTx)
	if !ok {
		// Transaction doesn't support extension options, continue
		return next(ctx, tx, simulate)
	}

	extensionOptions := hasExtOptsTx.GetExtensionOptions()

	// If no extension options, continue normally
	if len(extensionOptions) == 0 {
		return next(ctx, tx, simulate)
	}

	// Reject transactions with more than one extension option
	if len(extensionOptions) > 1 {
		return ctx, errorsmod.Wrapf(
			sdkerrors.ErrInvalidRequest,
			"only one extension option is supported, got %d",
			len(extensionOptions),
		)
	}

	// Process the single extension option
	extOpt := extensionOptions[0]

	// Check if it's a JWSExtensionOption
	jwsOpt, ok := extOpt.GetCachedValue().(*antetypes.JWSExtensionOption)
	if !ok {
		// Unknown extension option, reject
		return ctx, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "unsupported extension option type: %T", extOpt.GetCachedValue())
	}

	currentTime := ctx.BlockTime()

	// Check if bearer auth should be ignored for validation
	skipAuthValidation := eod.hubKeeper != nil && eod.hubKeeper.GetChainConfig(ctx).IgnoreBearerAuth

	// Validate JWS format, signature, required claims, and timing
	did, authorizedAccount, err := validateJWSExtension(ctx, jwsOpt.BearerToken, currentTime, skipAuthValidation)
	if err != nil {
		return ctx, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "bearer token validation failed: %v", err)
	}

	// Parse the bearer token to extract timestamps for storage
	bearerToken, err := parseValidateJWS(ctx, nil, jwsOpt.BearerToken, skipAuthValidation)
	if err != nil {
		return ctx, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "failed to parse bearer token: %v", err)
	}

	// Store or update the JWS token record
	if eod.hubKeeper != nil && !simulate {
		issuedAt := time.Unix(bearerToken.IssuedTime, 0)
		expiresAt := time.Unix(bearerToken.ExpirationTime, 0)

		err = eod.hubKeeper.StoreOrUpdateJWSToken(
			ctx,
			jwsOpt.BearerToken,
			did,
			authorizedAccount,
			issuedAt,
			expiresAt,
		)
		if err != nil {
			// Log error but don't fail the transaction
			ctx.Logger().Error("failed to store JWS token", "error", err)
		}
	}

	// Ensure that transaction has at least one message
	msgs := tx.GetMsgs()
	if len(msgs) == 0 {
		return ctx, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "transaction has no messages")
	}

	// Check that all messages are signed by the same authorized account
	if !skipAuthValidation {
		for i, msg := range msgs {
			var msgSigner string
			if msgWithCreator, ok := msg.(interface{ GetCreator() string }); ok {
				msgSigner = msgWithCreator.GetCreator()
			}
			if msgSigner == "" {
				return ctx, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "cannot determine signer for message %d", i)
			}
			if msgSigner != authorizedAccount {
				return ctx, errorsmod.Wrapf(
					sdkerrors.ErrUnauthorized,
					"message %d signer mismatch: bearer token authorizes %s but message is signed by %s",
					i,
					authorizedAccount,
					msgSigner,
				)
			}
		}
	}

	// Store extracted DID in context
	ctx = ctx.WithValue(appparams.ExtractedDIDContextKey, did)

	return next(ctx, tx, simulate)
}
