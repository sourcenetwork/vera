package keeper

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

	errorsmod "cosmossdk.io/errors"
	blst "github.com/supranational/blst/bindings/go"

	"github.com/sourcenetwork/sourcehub/x/bulletin/types"
)

const (
	ThresholdSignatureSchemeBLS12381 = "bls12_381"

	bls12381PublicKeySize  = 48
	bls12381SignatureSize  = 96
	bls12381G2SignatureDST = "BLS_SIG_BLS12381G2_XMD:SHA-256_SSWU_RO_POP_"
)

func verifyThresholdSignatureForRingPayloadUpdate(
	currentPayload []byte,
	nextPayload []byte,
	scheme string,
	signature []byte,
) error {
	currentRingPayload, err := parseRingPayloadJSON(currentPayload)
	if err != nil {
		return err
	}
	if _, err := parseRingPayloadJSON(nextPayload); err != nil {
		return err
	}

	return verifyThresholdSignature(scheme, *currentRingPayload.RingPK, nextPayload, signature)
}

func verifyThresholdSignature(scheme string, ringPK string, message []byte, signature []byte) error {
	switch normalizeThresholdSignatureScheme(scheme) {
	case ThresholdSignatureSchemeBLS12381:
		return verifyBLS12381ThresholdSignature(ringPK, message, signature)
	default:
		return errorsmod.Wrapf(types.ErrInvalidThresholdSignature, "unsupported threshold signature scheme %q", scheme)
	}
}

func normalizeThresholdSignatureScheme(scheme string) string {
	switch strings.ToLower(strings.TrimSpace(scheme)) {
	case "", "bls12_381", "bls12381", "bls12-381":
		return ThresholdSignatureSchemeBLS12381
	default:
		return strings.ToLower(strings.TrimSpace(scheme))
	}
}

func verifyBLS12381ThresholdSignature(ringPK string, message []byte, signature []byte) error {
	publicKey, err := decodeEncodedBytes(ringPK)
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

func decodeEncodedBytes(value string) ([]byte, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, fmt.Errorf("empty value")
	}

	if strings.HasPrefix(value, "0x") || strings.HasPrefix(value, "0X") {
		return hex.DecodeString(value[2:])
	}

	if decoded, err := hex.DecodeString(value); err == nil {
		return decoded, nil
	}
	if decoded, err := base64.StdEncoding.DecodeString(value); err == nil {
		return decoded, nil
	}
	if decoded, err := base64.RawStdEncoding.DecodeString(value); err == nil {
		return decoded, nil
	}
	if decoded, err := base64.URLEncoding.DecodeString(value); err == nil {
		return decoded, nil
	}
	if decoded, err := base64.RawURLEncoding.DecodeString(value); err == nil {
		return decoded, nil
	}

	return nil, fmt.Errorf("expected hex or base64")
}
