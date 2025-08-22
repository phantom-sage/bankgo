package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/phantom-sage/bankgo/pkg/auth"
	"github.com/stretchr/testify/assert"
)

func setupTestRouter(pasetoManager *auth.PASETOManager) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	// Protected route
	protected := router.Group("/api")
	protected.Use(AuthMiddleware(pasetoManager))
	protected.GET("/protected", func(c *gin.Context) {
		userID, exists := GetUserIDFromContext(c)
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "user ID not found"})
			return
		}
		
		email, exists := GetUserEmailFromContext(c)
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "email not found"})
			return
		}
		
		c.JSON(http.StatusOK, gin.H{
			"user_id": userID,
			"email":   email,
			"message": "access granted",
		})
	})
	
	return router
}

func TestAuthMiddleware(t *testing.T) {
	secretKey := "this-is-a-very-long-secret-key-for-testing-purposes"
	expiration := 24 * time.Hour
	pasetoManager, _ := auth.NewPASETOManager(secretKey, expiration)
	router := setupTestRouter(pasetoManager)

	t.Run("successful authentication", func(t *testing.T) {
		userID := 123
		email := "test@example.com"
		
		// Generate valid token
		token, err := pasetoManager.GenerateToken(userID, email)
		assert.NoError(t, err)

		// Create request with valid token
		req, _ := http.NewRequest("GET", "/api/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "access granted")
		assert.Contains(t, w.Body.String(), email)
		assert.Contains(t, w.Body.String(), "123")
	})

	t.Run("missing authorization header", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/protected", nil)
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "authorization header is required")
	})

	t.Run("invalid authorization header format", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/protected", nil)
		req.Header.Set("Authorization", "InvalidFormat token")
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "authorization header must start with 'Bearer '")
	})

	t.Run("empty token", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/protected", nil)
		req.Header.Set("Authorization", "Bearer ")
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "token is required")
	})

	t.Run("invalid token", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/protected", nil)
		req.Header.Set("Authorization", "Bearer invalid.token.format")
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "invalid or expired token")
	})

	t.Run("expired token", func(t *testing.T) {
		// Create manager with very short expiration
		shortExpiration := 1 * time.Millisecond
		shortManager, _ := auth.NewPASETOManager(secretKey, shortExpiration)
		
		// Generate token
		token, err := shortManager.GenerateToken(123, "test@example.com")
		assert.NoError(t, err)
		
		// Wait for token to expire
		time.Sleep(10 * time.Millisecond)
		
		req, _ := http.NewRequest("GET", "/api/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "invalid or expired token")
	})

	t.Run("token with wrong secret", func(t *testing.T) {
		// Create token with different secret
		wrongSecretKey := "this-is-a-different-secret-key-for-testing-purposes"
		wrongManager, _ := auth.NewPASETOManager(wrongSecretKey, expiration)
		token, _ := wrongManager.GenerateToken(123, "test@example.com")
		
		req, _ := http.NewRequest("GET", "/api/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "invalid or expired token")
	})
}

func TestGetUserIDFromContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	t.Run("user ID exists in context", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("user_id", 123)
		
		userID, exists := GetUserIDFromContext(c)
		
		assert.True(t, exists)
		assert.Equal(t, 123, userID)
	})

	t.Run("user ID does not exist in context", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		
		userID, exists := GetUserIDFromContext(c)
		
		assert.False(t, exists)
		assert.Equal(t, 0, userID)
	})

	t.Run("user ID has wrong type in context", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("user_id", "not_an_int")
		
		userID, exists := GetUserIDFromContext(c)
		
		assert.False(t, exists)
		assert.Equal(t, 0, userID)
	})
}

func TestGetUserEmailFromContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	t.Run("email exists in context", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("user_email", "test@example.com")
		
		email, exists := GetUserEmailFromContext(c)
		
		assert.True(t, exists)
		assert.Equal(t, "test@example.com", email)
	})

	t.Run("email does not exist in context", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		
		email, exists := GetUserEmailFromContext(c)
		
		assert.False(t, exists)
		assert.Equal(t, "", email)
	})

	t.Run("email has wrong type in context", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("user_email", 123)
		
		email, exists := GetUserEmailFromContext(c)
		
		assert.False(t, exists)
		assert.Equal(t, "", email)
	})
}

func TestGetTokenClaimsFromContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	t.Run("token claims exist in context", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		expectedClaims := &auth.TokenClaims{
			UserID: 123,
			Email:  "test@example.com",
		}
		c.Set("token_claims", expectedClaims)
		
		claims, exists := GetTokenClaimsFromContext(c)
		
		assert.True(t, exists)
		assert.Equal(t, expectedClaims, claims)
	})

	t.Run("token claims do not exist in context", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		
		claims, exists := GetTokenClaimsFromContext(c)
		
		assert.False(t, exists)
		assert.Nil(t, claims)
	})

	t.Run("token claims have wrong type in context", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("token_claims", "not_token_claims")
		
		claims, exists := GetTokenClaimsFromContext(c)
		
		assert.False(t, exists)
		assert.Nil(t, claims)
	})
}

func TestAuthMiddleware_Integration(t *testing.T) {
	secretKey := "this-is-a-very-long-secret-key-for-testing-purposes"
	expiration := 24 * time.Hour
	pasetoManager, _ := auth.NewPASETOManager(secretKey, expiration)
	
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	// Route that uses all context helper functions
	router.Use(AuthMiddleware(pasetoManager))
	router.GET("/test", func(c *gin.Context) {
		userID, userIDExists := GetUserIDFromContext(c)
		email, emailExists := GetUserEmailFromContext(c)
		claims, claimsExist := GetTokenClaimsFromContext(c)
		
		c.JSON(http.StatusOK, gin.H{
			"user_id_exists":    userIDExists,
			"user_id":           userID,
			"email_exists":      emailExists,
			"email":             email,
			"claims_exist":      claimsExist,
			"claims_user_id":    claims.UserID,
			"claims_email":      claims.Email,
		})
	})

	t.Run("all context helpers work together", func(t *testing.T) {
		userID := 456
		email := "integration@example.com"
		
		// Generate valid token
		token, err := pasetoManager.GenerateToken(userID, email)
		assert.NoError(t, err)

		// Create request with valid token
		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"user_id_exists":true`)
		assert.Contains(t, w.Body.String(), `"user_id":456`)
		assert.Contains(t, w.Body.String(), `"email_exists":true`)
		assert.Contains(t, w.Body.String(), `"email":"integration@example.com"`)
		assert.Contains(t, w.Body.String(), `"claims_exist":true`)
		assert.Contains(t, w.Body.String(), `"claims_user_id":456`)
		assert.Contains(t, w.Body.String(), `"claims_email":"integration@example.com"`)
	})
}