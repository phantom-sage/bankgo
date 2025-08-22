package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/phantom-sage/bankgo/pkg/auth"
)

// AuthMiddleware creates a middleware for PASETO token authentication
func AuthMiddleware(pasetoManager *auth.PASETOManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "authorization header is required",
				"code":    http.StatusUnauthorized,
			})
			c.Abort()
			return
		}

		// Check if header starts with "Bearer "
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "authorization header must start with 'Bearer '",
				"code":    http.StatusUnauthorized,
			})
			c.Abort()
			return
		}

		// Extract token
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "token is required",
				"code":    http.StatusUnauthorized,
			})
			c.Abort()
			return
		}

		// Validate token
		claims, err := pasetoManager.ValidateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "invalid or expired token",
				"code":    http.StatusUnauthorized,
			})
			c.Abort()
			return
		}

		// Set user information in context
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("token_claims", claims)

		c.Next()
	}
}

// GetUserIDFromContext extracts user ID from gin context
func GetUserIDFromContext(c *gin.Context) (int, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}

	id, ok := userID.(int)
	return id, ok
}

// GetUserEmailFromContext extracts user email from gin context
func GetUserEmailFromContext(c *gin.Context) (string, bool) {
	email, exists := c.Get("user_email")
	if !exists {
		return "", false
	}

	emailStr, ok := email.(string)
	return emailStr, ok
}

// GetTokenClaimsFromContext extracts token claims from gin context
func GetTokenClaimsFromContext(c *gin.Context) (*auth.TokenClaims, bool) {
	claims, exists := c.Get("token_claims")
	if !exists {
		return nil, false
	}

	tokenClaims, ok := claims.(*auth.TokenClaims)
	return tokenClaims, ok
}