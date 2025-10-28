package ante

import (
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
)

// RejectLegacyTxDecorator rejects transactions signed with SIGN_MODE_LEGACY_AMINO_JSON.
// Legacy transactions do not support extension options, which means the extension options
// part of the transaction is not signed. This creates a potential security vulnerability
// where extension options could be modified without invalidating the signature.
type RejectLegacyTxDecorator struct{}

// NewRejectLegacyTxDecorator creates a new RejectLegacyTxDecorator.
func NewRejectLegacyTxDecorator() RejectLegacyTxDecorator {
	return RejectLegacyTxDecorator{}
}

// AnteHandle rejects transactions that use SIGN_MODE_LEGACY_AMINO_JSON.
func (rltd RejectLegacyTxDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	sigTx, ok := tx.(authsigning.SigVerifiableTx)
	if !ok {
		return ctx, errorsmod.Wrap(sdkerrors.ErrTxDecode, "invalid transaction type")
	}

	sigs, err := sigTx.GetSignaturesV2()
	if err != nil {
		return ctx, err
	}

	// Check all signatures for legacy amino sign mode
	for i, sig := range sigs {
		if hasLegacyAminoSignMode(sig.Data) {
			return ctx, errorsmod.Wrapf(
				sdkerrors.ErrInvalidRequest,
				"signature %d uses SIGN_MODE_LEGACY_AMINO_JSON which is not supported due to security concerns (extension options are not signed)",
				i,
			)
		}
	}

	return next(ctx, tx, simulate)
}

// hasLegacyAminoSignMode recursively checks if any signature uses SIGN_MODE_LEGACY_AMINO_JSON.
func hasLegacyAminoSignMode(sigData signing.SignatureData) bool {
	switch data := sigData.(type) {
	case *signing.SingleSignatureData:
		return data.SignMode == signing.SignMode_SIGN_MODE_LEGACY_AMINO_JSON
	case *signing.MultiSignatureData:
		for _, sig := range data.Signatures {
			if hasLegacyAminoSignMode(sig) {
				return true
			}
		}
		return false
	default:
		return false
	}
}
