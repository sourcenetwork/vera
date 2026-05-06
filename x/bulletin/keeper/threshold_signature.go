package keeper

import (
	"encoding/hex"
	"fmt"
	"strings"

	errorsmod "cosmossdk.io/errors"
	decaf377 "github.com/mizufinance/decaf377-go"
	"github.com/mizufinance/decaf377-go/orbisfrost"
	blst "github.com/supranational/blst/bindings/go"

	"github.com/sourcenetwork/sourcehub/x/bulletin/types"
)

const (
	ThresholdSignatureSchemeBLS12381G1PKG2SigNUL = "bls12_381_g1_pk_g2_sig_nul"
	ThresholdSignatureSchemeDecaf377FROST        = "decaf377_frost"

	bls12381PublicKeySize  = 48
	bls12381SignatureSize  = 96
	bls12381G2SignatureDST = "BLS_SIG_BLS12381G2_XMD:SHA-256_SSWU_RO_NUL_"
	decaf377PublicKeySize  = decaf377.ElementSize
	decaf377SignatureSize  = decaf377.ElementSize + decaf377.ScalarSize
)

func verifyThresholdSignatureForRingPayloadUpdate(
	currentRingPayload *ringPayloadJSON,
	message []byte,
	scheme string,
	signature []byte,
) error {
	return verifyThresholdSignature(scheme, *currentRingPayload.RingPK, message, signature)
}

func verifyThresholdSignature(scheme string, ringPK string, message []byte, signature []byte) error {
	switch normalizeThresholdSignatureScheme(scheme) {
	case ThresholdSignatureSchemeBLS12381G1PKG2SigNUL:
		return verifyBLS12381ThresholdSignature(ringPK, message, signature)
	case ThresholdSignatureSchemeDecaf377FROST:
		return verifyDecaf377FROSTThresholdSignature(ringPK, message, signature)
	default:
		return errorsmod.Wrapf(types.ErrInvalidThresholdSignature, "unsupported threshold signature scheme %q", scheme)
	}
}

func normalizeThresholdSignatureScheme(scheme string) string {
	return strings.ToLower(strings.TrimSpace(scheme))
}

func verifyBLS12381ThresholdSignature(ringPK string, message []byte, signature []byte) error {
	publicKey, err := decodeHexBytes(ringPK)
	if err != nil {
		return errorsmod.Wrapf(types.ErrInvalidThresholdSignature, "invalid bls12_381 public key encoding: %s", err)
	}
	if len(publicKey) != bls12381PublicKeySize {
		return errorsmod.Wrapf(types.ErrInvalidThresholdSignature, "invalid bls12_381 public key length %d", len(publicKey))
	}
	if len(signature) != bls12381SignatureSize {
		return errorsmod.Wrapf(types.ErrInvalidThresholdSignature, "invalid bls12_381 signature length %d", len(signature))
	}

	if !new(blst.P2Affine).VerifyCompressed(
		signature,
		true,
		publicKey,
		true,
		message,
		[]byte(bls12381G2SignatureDST),
	) {
		return types.ErrInvalidThresholdSignature
	}

	return nil
}

func verifyDecaf377FROSTThresholdSignature(ringPK string, message []byte, signature []byte) error {
	publicKey, err := decodeHexBytes(ringPK)
	if err != nil {
		return errorsmod.Wrapf(types.ErrInvalidThresholdSignature, "invalid decaf377 public key encoding: %s", err)
	}
	if len(publicKey) != decaf377PublicKeySize {
		return errorsmod.Wrapf(types.ErrInvalidThresholdSignature, "invalid decaf377 public key length %d", len(publicKey))
	}
	if len(signature) != decaf377SignatureSize {
		return errorsmod.Wrapf(types.ErrInvalidThresholdSignature, "invalid decaf377 signature length %d", len(signature))
	}

	ok, err := orbisfrost.Verify(publicKey, message, signature)
	if err != nil {
		return errorsmod.Wrapf(types.ErrInvalidThresholdSignature, "invalid decaf377 signature: %s", err)
	}
	if !ok {
		return types.ErrInvalidThresholdSignature
	}

	return nil
}

func decodeHexBytes(value string) ([]byte, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, fmt.Errorf("empty value")
	}

	decoded, err := hex.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("expected plain hex: %w", err)
	}

	return decoded, nil
}
