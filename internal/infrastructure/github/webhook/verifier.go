package webhook

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"go.uber.org/zap"
)

// SignatureVerifier verifies GitHub webhook signatures using HMAC-SHA256.
type SignatureVerifier struct {
	secret []byte
	logger *zap.Logger
}

// NewSignatureVerifier creates a new signature verifier.
func NewSignatureVerifier(secret string, logger *zap.Logger) *SignatureVerifier {
	return &SignatureVerifier{
		secret: []byte(secret),
		logger: logger.Named("webhook.verifier"),
	}
}

// Verify verifies the X-Hub-Signature-256 header against the payload.
// Returns true if signature is valid, false otherwise.
//
// IMPORTANT: This is security-sensitive. The implementation must:
// 1. Use HMAC-SHA256 (not SHA1)
// 2. Compare signatures using hmac.Equal (constant-time comparison)
// 3. Handle edge cases (missing header, malformed signature)
func (v *SignatureVerifier) Verify(payload []byte, signatureHeader string) bool {
	// Check if header is present
	if signatureHeader == "" {
		v.logger.Warn("missing signature header")
		return false
	}

	// Parse signature header format: "sha256=<hex>"
	const expectedPrefix = "sha256="
	if !strings.HasPrefix(signatureHeader, expectedPrefix) {
		v.logger.Warn("invalid signature format",
			zap.String("header", signatureHeader),
		)
		return false
	}

	// Decode hex signature
	expectedSigHex := strings.TrimPrefix(signatureHeader, "sha256=")
	expectedSig, err := hex.DecodeString(expectedSigHex)
	if err != nil {
		v.logger.Warn("failed to decode signature hex",
			zap.Error(err),
		)
		return false
	}

	// Compute HMAC-SHA256 of payload
	mac := hmac.New(sha256.New, v.secret)
	mac.Write(payload)
	computedSig := mac.Sum(nil)

	// Use constant-time comparison to prevent timing attacks
	if !hmac.Equal(expectedSig, computedSig) {
		v.logger.Warn("signature mismatch")
		return false
	}

	v.logger.Debug("signature verified successfully")
	return true
}

// VerifySHA1 verifies the X-Hub-Signature header (SHA1, legacy).
// Deprecated: Use Verify for SHA256 signatures instead.
func (v *SignatureVerifier) VerifySHA1(payload []byte, signatureHeader string) bool {
	if signatureHeader == "" {
		return false
	}

	if !strings.HasPrefix(signatureHeader, "sha1=") {
		return false
	}

	expectedSigHex := strings.TrimPrefix(signatureHeader, "sha1=")
	expectedSig, err := hex.DecodeString(expectedSigHex)
	if err != nil {
		return false
	}

	mac := hmac.New(sha1.New, v.secret)
	mac.Write(payload)
	computedSig := mac.Sum(nil)

	return hmac.Equal(expectedSig, computedSig)
}