package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/o1egl/paseto/v2"
)

// TokenClaims represents the claims in a PASETO token
type TokenClaims struct {
	UserID    int       `json:"user_id"`
	Email     string    `json:"email"`
	IssuedAt  time.Time `json:"iat"`
	ExpiresAt time.Time `json:"exp"`
}

// PASETOManager handles PASETO token operations
type PASETOManager struct {
	secretKey  []byte
	expiration time.Duration
}

// NewPASETOManager creates a new PASETO manager
func NewPASETOManager(secretKey string, expiration time.Duration) (*PASETOManager, error) {
	if len(secretKey) < 32 {
		return nil, errors.New("secret key must be at least 32 characters long")
	}

	// PASETO v2 requires exactly 32 bytes for the key
	key := []byte(secretKey)
	if len(key) > 32 {
		key = key[:32] // Truncate to 32 bytes
	} else if len(key) < 32 {
		// Pad with zeros if less than 32 bytes (though we check above)
		padded := make([]byte, 32)
		copy(padded, key)
		key = padded
	}

	return &PASETOManager{
		secretKey:  key,
		expiration: expiration,
	}, nil
}

// GenerateToken generates a new PASETO token for the given user
func (pm *PASETOManager) GenerateToken(userID int, email string) (string, error) {
	if userID <= 0 {
		return "", errors.New("invalid user ID")
	}

	if email == "" {
		return "", errors.New("email cannot be empty")
	}

	now := time.Now()
	claims := TokenClaims{
		UserID:    userID,
		Email:     email,
		IssuedAt:  now,
		ExpiresAt: now.Add(pm.expiration),
	}

	token, err := paseto.NewV2().Encrypt(pm.secretKey, claims, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	return token, nil
}

// ValidateToken validates a PASETO token and returns the claims
func (pm *PASETOManager) ValidateToken(token string) (*TokenClaims, error) {
	if token == "" {
		return nil, errors.New("token cannot be empty")
	}

	var claims TokenClaims
	err := paseto.NewV2().Decrypt(token, pm.secretKey, &claims, nil)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	// Check if token is expired
	if time.Now().After(claims.ExpiresAt) {
		return nil, errors.New("token has expired")
	}

	// Validate claims
	if claims.UserID <= 0 {
		return nil, errors.New("invalid user ID in token")
	}

	if claims.Email == "" {
		return nil, errors.New("invalid email in token")
	}

	return &claims, nil
}

// RefreshToken generates a new token with extended expiration
func (pm *PASETOManager) RefreshToken(token string) (string, error) {
	claims, err := pm.ValidateToken(token)
	if err != nil {
		return "", fmt.Errorf("cannot refresh invalid token: %w", err)
	}

	// Generate new token with same user info but new expiration
	return pm.GenerateToken(claims.UserID, claims.Email)
}

// GetTokenExpiration returns the token expiration duration
func (pm *PASETOManager) GetTokenExpiration() time.Duration {
	return pm.expiration
}

// IsTokenExpired checks if a token is expired without full validation
func (pm *PASETOManager) IsTokenExpired(token string) bool {
	claims, err := pm.ValidateToken(token)
	if err != nil {
		return true // Consider invalid tokens as expired
	}

	return time.Now().After(claims.ExpiresAt)
}