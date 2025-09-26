package ante

import (
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"

	antetypes "github.com/sourcenetwork/sourcehub/app/ante/types"
)

// Context key for storing extracted DID from JWS extension options.
type contextKey string

const (
	// ExtractedDIDContextKey is the key used to store extracted DID in context
	ExtractedDIDContextKey contextKey = "extracted_did"
)

// ExtensionOptionsDecorator validates extension options in transactions.
// It allows JWSExtensionOption and rejects all other extension options.
// It also extracts DID from JWS extension options and stores it in context.
type ExtensionOptionsDecorator struct{}

// NewExtensionOptionsDecorator creates a new ExtensionOptionsDecorator.
func NewExtensionOptionsDecorator() ExtensionOptionsDecorator {
	return ExtensionOptionsDecorator{}
}

// AnteHandle validates extension options, allowing only JWSExtensionOption.
// It extracts DID from JWS extension options and stores it in context for later use.
func (eod ExtensionOptionsDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	// Check if the transaction has extension options
	if hasExtOptsTx, ok := tx.(ante.HasExtensionOptionsTx); ok {
		extensionOptions := hasExtOptsTx.GetExtensionOptions()
		if len(extensionOptions) > 0 {
			var extractedDID string

			// Validate each extension option
			for _, extOpt := range extensionOptions {
				// Check if it's a JWSExtensionOption
				if jwsOpt, ok := extOpt.GetCachedValue().(*antetypes.JWSExtensionOption); ok {
					currentTime := ctx.BlockTime()

					// Validate JWS format, signature, required claims, and timing
					did, authorizedAccount, err := validateJWSExtension(ctx, jwsOpt.BearerToken, currentTime)
					if err != nil {
						return ctx, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "bearer token validation failed: %v", err)
					}

					// Verify authorized account matches the first message creator/signer
					msgs := tx.GetMsgs()
					if len(msgs) == 0 {
						return ctx, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "transaction has no messages")
					}

					firstMsg := msgs[0]
					var msgSigner string

					// Try GetCreator for modern messages
					if msgWithCreator, ok := firstMsg.(interface{ GetCreator() string }); ok {
						msgSigner = msgWithCreator.GetCreator()
					} else if legacyMsg, ok := firstMsg.(sdk.LegacyMsg); ok {
						// Fallback to GetSigners for legacy messages
						signers := legacyMsg.GetSigners()
						if len(signers) > 0 {
							msgSigner = signers[0].String()
						}
					}

					if msgSigner == "" {
						return ctx, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "cannot determine message signer")
					}

					if msgSigner != authorizedAccount {
						return ctx, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "authorized account mismatch: bearer token authorizes %s but transaction is signed by %s", authorizedAccount, msgSigner)
					}

					extractedDID = did
				} else {
					// Unknown extension option, reject
					return ctx, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "unsupported extension option type: %T", extOpt.GetCachedValue())
				}
			}

			// Store extracted DID in context if found
			if extractedDID != "" {
				ctx = ctx.WithValue(ExtractedDIDContextKey, extractedDID)
			}
		}
	}

	return next(ctx, tx, simulate)
}
