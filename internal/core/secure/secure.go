package secure

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/base64"
	"fmt"
	"strings"
)

const (
	markerSecret = "your-secret-key-from-config" // TODO: Load from config
)

func CreateNewAccessAndSecretKey() (string, string, error) {
	accessKey, err := generateAccessKey()
	if err != nil {
		return "", "", err
	}

	secretKey, err := generateSecretKey()
	if err != nil {
		return "", "", err
	}

	return accessKey, secretKey, nil
}

// TODO: am I using proper algorithem
func generateAccessKey() (string, error) {
	// 10 random bytes ~ 16 base32 chars
	b := make([]byte, 10)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	ak := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b)
	return "AFA" + strings.ToUpper(ak), nil // AFA prefix = "A for Apple"
}

// TODO: am I using proper algorithem
func generateSecretKey() (string, error) {
	b := make([]byte, 32) // 256-bit
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

func EncodeSignedMarker(value string) string {
	if value == "" {
		return ""
	}

	// Create signature
	h := hmac.New(sha256.New, []byte(markerSecret))
	h.Write([]byte(value))
	signature := base64.URLEncoding.EncodeToString(h.Sum(nil))

	// Combine value and signature
	encoded := base64.URLEncoding.EncodeToString([]byte(value))
	return fmt.Sprintf("%s.%s", encoded, signature)
}

func DecodeSignedMarker(marker string) (string, error) {
	if marker == "" {
		return "", nil
	}

	parts := strings.Split(marker, ".")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid marker format")
	}

	encoded, signature := parts[0], parts[1]

	// Decode value
	decoded, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	value := string(decoded)

	// Verify signature
	h := hmac.New(sha256.New, []byte(markerSecret))
	h.Write([]byte(value))
	expectedSig := base64.URLEncoding.EncodeToString(h.Sum(nil))

	if !hmac.Equal([]byte(signature), []byte(expectedSig)) {
		return "", fmt.Errorf("invalid marker signature")
	}

	return value, nil
}
