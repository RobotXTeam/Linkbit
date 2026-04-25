package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
)

const apiKeyBytes = 32

var ErrInvalidSecret = errors.New("api key pepper must not be empty")

// NewAPIKey returns a high-entropy bearer secret. The caller must show this
// value only once and persist HashAPIKey output instead of the plaintext key.
func NewAPIKey() (string, error) {
	raw := make([]byte, apiKeyBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}

	return "lb_" + base64.RawURLEncoding.EncodeToString(raw), nil
}

// HashAPIKey derives a stable HMAC digest. HMAC keeps database disclosure from
// becoming immediate API access as long as the server-side pepper remains safe.
func HashAPIKey(apiKey string, pepper []byte) (string, error) {
	if len(pepper) == 0 {
		return "", ErrInvalidSecret
	}
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return "", errors.New("api key must not be empty")
	}

	mac := hmac.New(sha256.New, pepper)
	_, _ = mac.Write([]byte(apiKey))
	return hex.EncodeToString(mac.Sum(nil)), nil
}

// VerifyAPIKey compares a presented key with a stored digest without leaking
// timing information about the expected value.
func VerifyAPIKey(presented string, expectedDigest string, pepper []byte) bool {
	actualDigest, err := HashAPIKey(presented, pepper)
	if err != nil {
		return false
	}

	return hmac.Equal([]byte(actualDigest), []byte(expectedDigest))
}
