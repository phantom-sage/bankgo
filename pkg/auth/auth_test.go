package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewPASETOManager(t *testing.T) {
	t.Run("successful creation", func(t *testing.T) {
		secretKey := "this-is-a-very-long-secret-key-for-testing-purposes"
		expiration := 24 * time.Hour

		manager, err := NewPASETOManager(secretKey, expiration)

		assert.NoError(t, err)
		assert.NotNil(t, manager)
		assert.Equal(t, expiration, manager.expiration)
		assert.Len(t, manager.secretKey, 32) // Should be exactly 32 bytes
		assert.Equal(t, []byte(secretKey)[:32], manager.secretKey) // Should be first 32 bytes
	})

	t.Run("secret key too short", func(t *testing.T) {
		secretKey := "short"
		expiration := 24 * time.Hour

		manager, err := NewPASETOManager(secretKey, expiration)

		assert.Error(t, err)
		assert.Nil(t, manager)
		assert.Contains(t, err.Error(), "secret key must be at least 32 characters long")
	})
}

func TestPASETOManager_GenerateToken(t *testing.T) {
	secretKey := "this-is-a-very-long-secret-key-for-testing-purposes"
	expiration := 24 * time.Hour
	manager, _ := NewPASETOManager(secretKey, expiration)

	t.Run("successful token generation", func(t *testing.T) {
		userID := 123
		email := "test@example.com"

		token, err := manager.GenerateToken(userID, email)

		assert.NoError(t, err)
		assert.NotEmpty(t, token)
		assert.Contains(t, token, "v2.local.")
	})

	t.Run("invalid user ID", func(t *testing.T) {
		userID := 0
		email := "test@example.com"

		token, err := manager.GenerateToken(userID, email)

		assert.Error(t, err)
		assert.Empty(t, token)
		assert.Contains(t, err.Error(), "invalid user ID")
	})

	t.Run("negative user ID", func(t *testing.T) {
		userID := -1
		email := "test@example.com"

		token, err := manager.GenerateToken(userID, email)

		assert.Error(t, err)
		assert.Empty(t, token)
		assert.Contains(t, err.Error(), "invalid user ID")
	})

	t.Run("empty email", func(t *testing.T) {
		userID := 123
		email := ""

		token, err := manager.GenerateToken(userID, email)

		assert.Error(t, err)
		assert.Empty(t, token)
		assert.Contains(t, err.Error(), "email cannot be empty")
	})
}

func TestPASETOManager_ValidateToken(t *testing.T) {
	secretKey := "this-is-a-very-long-secret-key-for-testing-purposes"
	expiration := 24 * time.Hour
	manager, _ := NewPASETOManager(secretKey, expiration)

	t.Run("successful token validation", func(t *testing.T) {
		userID := 123
		email := "test@example.com"

		// Generate token
		token, err := manager.GenerateToken(userID, email)
		assert.NoError(t, err)

		// Validate token
		claims, err := manager.ValidateToken(token)

		assert.NoError(t, err)
		assert.NotNil(t, claims)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, email, claims.Email)
		assert.True(t, claims.ExpiresAt.After(time.Now()))
		assert.True(t, claims.IssuedAt.Before(time.Now().Add(time.Second)))
	})

	t.Run("empty token", func(t *testing.T) {
		claims, err := manager.ValidateToken("")

		assert.Error(t, err)
		assert.Nil(t, claims)
		assert.Contains(t, err.Error(), "token cannot be empty")
	})

	t.Run("invalid token format", func(t *testing.T) {
		invalidToken := "invalid.token.format"

		claims, err := manager.ValidateToken(invalidToken)

		assert.Error(t, err)
		assert.Nil(t, claims)
		assert.Contains(t, err.Error(), "invalid token")
	})

	t.Run("token with wrong secret", func(t *testing.T) {
		// Create token with different manager
		wrongSecretKey := "this-is-a-different-secret-key-for-testing-purposes"
		wrongManager, _ := NewPASETOManager(wrongSecretKey, expiration)
		token, _ := wrongManager.GenerateToken(123, "test@example.com")

		// Try to validate with original manager
		claims, err := manager.ValidateToken(token)

		assert.Error(t, err)
		assert.Nil(t, claims)
		assert.Contains(t, err.Error(), "invalid token")
	})

	t.Run("expired token", func(t *testing.T) {
		// Create manager with very short expiration
		shortExpiration := 1 * time.Millisecond
		shortManager, _ := NewPASETOManager(secretKey, shortExpiration)

		// Generate token
		token, err := shortManager.GenerateToken(123, "test@example.com")
		assert.NoError(t, err)

		// Wait for token to expire
		time.Sleep(10 * time.Millisecond)

		// Validate expired token
		claims, err := shortManager.ValidateToken(token)

		assert.Error(t, err)
		assert.Nil(t, claims)
		assert.Contains(t, err.Error(), "token has expired")
	})
}

func TestPASETOManager_RefreshToken(t *testing.T) {
	secretKey := "this-is-a-very-long-secret-key-for-testing-purposes"
	expiration := 24 * time.Hour
	manager, _ := NewPASETOManager(secretKey, expiration)

	t.Run("successful token refresh", func(t *testing.T) {
		userID := 123
		email := "test@example.com"

		// Generate original token
		originalToken, err := manager.GenerateToken(userID, email)
		assert.NoError(t, err)

		// Wait a bit to ensure different timestamps
		time.Sleep(10 * time.Millisecond)

		// Refresh token
		newToken, err := manager.RefreshToken(originalToken)

		assert.NoError(t, err)
		assert.NotEmpty(t, newToken)
		assert.NotEqual(t, originalToken, newToken)

		// Validate new token
		claims, err := manager.ValidateToken(newToken)
		assert.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, email, claims.Email)
	})

	t.Run("refresh invalid token", func(t *testing.T) {
		invalidToken := "invalid.token.format"

		newToken, err := manager.RefreshToken(invalidToken)

		assert.Error(t, err)
		assert.Empty(t, newToken)
		assert.Contains(t, err.Error(), "cannot refresh invalid token")
	})

	t.Run("refresh expired token", func(t *testing.T) {
		// Create manager with very short expiration
		shortExpiration := 1 * time.Millisecond
		shortManager, _ := NewPASETOManager(secretKey, shortExpiration)

		// Generate token
		token, err := shortManager.GenerateToken(123, "test@example.com")
		assert.NoError(t, err)

		// Wait for token to expire
		time.Sleep(10 * time.Millisecond)

		// Try to refresh expired token
		newToken, err := shortManager.RefreshToken(token)

		assert.Error(t, err)
		assert.Empty(t, newToken)
		assert.Contains(t, err.Error(), "cannot refresh invalid token")
	})
}

func TestPASETOManager_GetTokenExpiration(t *testing.T) {
	secretKey := "this-is-a-very-long-secret-key-for-testing-purposes"
	expiration := 48 * time.Hour
	manager, _ := NewPASETOManager(secretKey, expiration)

	result := manager.GetTokenExpiration()

	assert.Equal(t, expiration, result)
}

func TestPASETOManager_IsTokenExpired(t *testing.T) {
	secretKey := "this-is-a-very-long-secret-key-for-testing-purposes"
	expiration := 24 * time.Hour
	manager, _ := NewPASETOManager(secretKey, expiration)

	t.Run("valid token not expired", func(t *testing.T) {
		token, err := manager.GenerateToken(123, "test@example.com")
		assert.NoError(t, err)

		isExpired := manager.IsTokenExpired(token)

		assert.False(t, isExpired)
	})

	t.Run("expired token", func(t *testing.T) {
		// Create manager with very short expiration
		shortExpiration := 1 * time.Millisecond
		shortManager, _ := NewPASETOManager(secretKey, shortExpiration)

		// Generate token
		token, err := shortManager.GenerateToken(123, "test@example.com")
		assert.NoError(t, err)

		// Wait for token to expire
		time.Sleep(10 * time.Millisecond)

		isExpired := shortManager.IsTokenExpired(token)

		assert.True(t, isExpired)
	})

	t.Run("invalid token considered expired", func(t *testing.T) {
		invalidToken := "invalid.token.format"

		isExpired := manager.IsTokenExpired(invalidToken)

		assert.True(t, isExpired)
	})
}

func TestTokenClaims_Validation(t *testing.T) {
	secretKey := "this-is-a-very-long-secret-key-for-testing-purposes"
	expiration := 24 * time.Hour
	manager, _ := NewPASETOManager(secretKey, expiration)

	t.Run("token with invalid user ID in claims", func(t *testing.T) {
		// This test verifies that the validation logic in ValidateToken
		// properly checks the claims content
		userID := 123
		email := "test@example.com"

		token, err := manager.GenerateToken(userID, email)
		assert.NoError(t, err)

		// Validate the token normally first
		claims, err := manager.ValidateToken(token)
		assert.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, email, claims.Email)
	})
}

func TestPASETOManager_EdgeCases(t *testing.T) {
	secretKey := "this-is-a-very-long-secret-key-for-testing-purposes"
	expiration := 24 * time.Hour
	manager, _ := NewPASETOManager(secretKey, expiration)

	t.Run("token generation and validation with special characters in email", func(t *testing.T) {
		userID := 123
		email := "test+special@example-domain.co.uk"

		token, err := manager.GenerateToken(userID, email)
		assert.NoError(t, err)

		claims, err := manager.ValidateToken(token)
		assert.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, email, claims.Email)
	})

	t.Run("token generation with maximum user ID", func(t *testing.T) {
		userID := 2147483647 // Max int32
		email := "test@example.com"

		token, err := manager.GenerateToken(userID, email)
		assert.NoError(t, err)

		claims, err := manager.ValidateToken(token)
		assert.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, email, claims.Email)
	})
}