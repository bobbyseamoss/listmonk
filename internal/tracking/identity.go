package tracking

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"time"
)

// ErrTokenExpired indicates the identity token has expired.
var ErrTokenExpired = errors.New("identity token has expired")

// ErrInvalidToken indicates the identity token is invalid or corrupted.
var ErrInvalidToken = errors.New("invalid identity token")

// IdentityToken contains the encrypted payload for subscriber identification.
type IdentityToken struct {
	SubscriberUUID string    `json:"s"` // Subscriber UUID
	CampaignUUID   string    `json:"c"` // Campaign UUID (optional)
	CreatedAt      time.Time `json:"t"` // Token creation time
}

// GenerateToken creates an encrypted identity token containing subscriber and campaign UUIDs.
// The token is URL-safe base64 encoded and can be appended to email links.
func GenerateToken(subscriberUUID, campaignUUID string, key []byte) (string, error) {
	token := IdentityToken{
		SubscriberUUID: subscriberUUID,
		CampaignUUID:   campaignUUID,
		CreatedAt:      time.Now().UTC(),
	}

	// Serialize to JSON
	plaintext, err := json.Marshal(token)
	if err != nil {
		return "", err
	}

	// Encrypt using AES-256-GCM
	ciphertext, err := encrypt(plaintext, key)
	if err != nil {
		return "", err
	}

	// Return URL-safe base64 encoded string
	return base64.RawURLEncoding.EncodeToString(ciphertext), nil
}

// ValidateToken decrypts and validates an identity token.
// Returns the token contents if valid and not expired.
// maxAge is the maximum age of the token in hours (e.g., 24 for 24 hours).
func ValidateToken(tokenStr string, key []byte, maxAgeHours int) (*IdentityToken, error) {
	// Decode from URL-safe base64
	ciphertext, err := base64.RawURLEncoding.DecodeString(tokenStr)
	if err != nil {
		return nil, ErrInvalidToken
	}

	// Decrypt using AES-256-GCM
	plaintext, err := decrypt(ciphertext, key)
	if err != nil {
		return nil, ErrInvalidToken
	}

	// Deserialize from JSON
	var token IdentityToken
	if err := json.Unmarshal(plaintext, &token); err != nil {
		return nil, ErrInvalidToken
	}

	// Check expiry
	if maxAgeHours > 0 {
		expiryTime := token.CreatedAt.Add(time.Duration(maxAgeHours) * time.Hour)
		if time.Now().UTC().After(expiryTime) {
			return nil, ErrTokenExpired
		}
	}

	return &token, nil
}

// encrypt encrypts plaintext using AES-256-GCM.
func encrypt(plaintext, key []byte) ([]byte, error) {
	// Ensure key is 32 bytes for AES-256
	if len(key) < 32 {
		// Pad key if too short
		paddedKey := make([]byte, 32)
		copy(paddedKey, key)
		key = paddedKey
	} else if len(key) > 32 {
		key = key[:32]
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Generate random nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// Encrypt and prepend nonce to ciphertext
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// decrypt decrypts ciphertext using AES-256-GCM.
func decrypt(ciphertext, key []byte) ([]byte, error) {
	// Ensure key is 32 bytes for AES-256
	if len(key) < 32 {
		paddedKey := make([]byte, 32)
		copy(paddedKey, key)
		key = paddedKey
	} else if len(key) > 32 {
		key = key[:32]
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Extract nonce from beginning of ciphertext
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, ErrInvalidToken
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrInvalidToken
	}

	return plaintext, nil
}

// GenerateBrowserID creates a random browser ID for tracking.
func GenerateBrowserID() string {
	b := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		// Fallback to timestamp-based ID
		return base64.RawURLEncoding.EncodeToString([]byte(time.Now().UTC().String()))
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
