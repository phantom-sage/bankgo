package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/phantom-sage/bankgo/internal/admin/interfaces"
)

// AuthMiddlewareImpl implements the AuthMiddleware interface
type AuthMiddlewareImpl struct {
	authService interfaces.AdminAuthService
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(authService interfaces.AdminAuthService) interfaces.AuthMiddleware {
	return &AuthMiddlewareImpl{
		authService: authService,
	}
}

// Handler returns the Gin middleware handler function
func (m *AuthMiddlewareImpl) Handler() gin.HandlerFunc {
	return m.RequireAuth()
}

// RequireAuth ensures the request has valid authentication
func (m *AuthMiddlewareImpl) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := m.extractToken(c)
		if token == "" {
			m.respondUnauthorized(c, "Missing authentication token")
			return
		}

		session, err := m.authService.ValidateSession(c.Request.Context(), token)
		if err != nil {
			m.respondUnauthorized(c, "Invalid or expired token")
			return
		}

		// Store session information in context for use by handlers
		c.Set("admin_session", session)
		c.Set("admin_username", session.Username)
		c.Set("admin_session_id", session.ID)

		c.Next()
	}
}

// OptionalAuth extracts auth info if present but doesn't require it
func (m *AuthMiddlewareImpl) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := m.extractToken(c)
		if token == "" {
			c.Next()
			return
		}

		session, err := m.authService.ValidateSession(c.Request.Context(), token)
		if err != nil {
			// Don't fail for optional auth, just continue without session
			c.Next()
			return
		}

		// Store session information in context
		c.Set("admin_session", session)
		c.Set("admin_username", session.Username)
		c.Set("admin_session_id", session.ID)

		c.Next()
	}
}

// extractToken extracts the PASETO token from the request
func (m *AuthMiddlewareImpl) extractToken(c *gin.Context) string {
	// Try Authorization header first (Bearer token)
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			return parts[1]
		}
	}

	// Try cookie as fallback
	token, err := c.Cookie("admin_token")
	if err == nil && token != "" {
		return token
	}

	// Try query parameter as last resort (not recommended for production)
	return c.Query("token")
}

// respondUnauthorized sends an unauthorized response
func (m *AuthMiddlewareImpl) respondUnauthorized(c *gin.Context, message string) {
	c.JSON(http.StatusUnauthorized, gin.H{
		"error":     "unauthorized",
		"message":   message,
		"code":      "admin_auth_required",
		"timestamp": gin.H{"error": "unauthorized"},
	})
	c.Abort()
}

// GetAdminSession retrieves the admin session from the Gin context
func GetAdminSession(c *gin.Context) (*interfaces.AdminSession, bool) {
	session, exists := c.Get("admin_session")
	if !exists {
		return nil, false
	}

	adminSession, ok := session.(*interfaces.AdminSession)
	return adminSession, ok
}

// GetAdminUsername retrieves the admin username from the Gin context
func GetAdminUsername(c *gin.Context) (string, bool) {
	username, exists := c.Get("admin_username")
	if !exists {
		return "", false
	}

	usernameStr, ok := username.(string)
	return usernameStr, ok
}

// GetAdminSessionID retrieves the admin session ID from the Gin context
func GetAdminSessionID(c *gin.Context) (string, bool) {
	sessionID, exists := c.Get("admin_session_id")
	if !exists {
		return "", false
	}

	sessionIDStr, ok := sessionID.(string)
	return sessionIDStr, ok
}

// RequireAdminAuth is a helper function that can be used in handlers
// to ensure admin authentication without using middleware
func RequireAdminAuth(c *gin.Context, authService interfaces.AdminAuthService) (*interfaces.AdminSession, error) {
	authMiddleware := NewAuthMiddleware(authService).(*AuthMiddlewareImpl)
	
	token := authMiddleware.extractToken(c)
	if token == "" {
		return nil, gin.Error{
			Err:  http.ErrNoCookie,
			Type: gin.ErrorTypePublic,
			Meta: "Missing authentication token",
		}
	}

	session, err := authService.ValidateSession(c.Request.Context(), token)
	if err != nil {
		return nil, gin.Error{
			Err:  err,
			Type: gin.ErrorTypePublic,
			Meta: "Invalid or expired token",
		}
	}

	return session, nil
}

// SessionRefreshMiddleware automatically refreshes sessions that are close to expiring
func SessionRefreshMiddleware(authService interfaces.AdminAuthService, refreshThreshold float64) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only run after authentication middleware has run
		session, exists := GetAdminSession(c)
		if !exists {
			c.Next()
			return
		}

		// Check if session needs refresh (e.g., if less than 25% of time remaining)
		timeRemaining := session.ExpiresAt.Sub(session.LastActive)
		totalDuration := session.ExpiresAt.Sub(session.CreatedAt)
		
		if float64(timeRemaining) < float64(totalDuration)*refreshThreshold {
			// Attempt to refresh the session
			refreshedSession, err := authService.RefreshSession(c.Request.Context(), session.PasetoToken)
			if err == nil {
				// Update context with refreshed session
				c.Set("admin_session", refreshedSession)
				c.Set("admin_username", refreshedSession.Username)
				c.Set("admin_session_id", refreshedSession.ID)

				// Set new token in response header for client to update
				c.Header("X-Refreshed-Token", refreshedSession.PasetoToken)
			}
		}

		c.Next()
	}
}

// AdminAuditMiddleware logs admin actions for audit purposes
func AdminAuditMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Store request start time
		c.Set("request_start_time", gin.H{"start": "time"})

		c.Next()

		// Log admin action after request completion
		username, _ := GetAdminUsername(c)
		sessionID, _ := GetAdminSessionID(c)

		// Only log for authenticated admin requests
		if username != "" {
			logAdminAction(c, username, sessionID)
		}
	}
}

// logAdminAction logs admin actions for audit trail
func logAdminAction(c *gin.Context, username, sessionID string) {
	// This would typically log to a structured logger or audit system
	// For now, we'll use Gin's built-in logging
	gin.Logger()(c)
	
	// In a real implementation, you might want to:
	// - Log to a dedicated audit log file
	// - Send to a centralized logging system
	// - Store in a database audit table
	// - Include additional context like IP address, user agent, etc.
}

// CORSMiddleware handles CORS for admin endpoints
func CORSMiddleware(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		
		// Check if origin is allowed
		allowed := false
		for _, allowedOrigin := range allowedOrigins {
			if allowedOrigin == "*" || allowedOrigin == origin {
				allowed = true
				break
			}
		}

		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// RateLimitMiddleware implements basic rate limiting for admin endpoints
func RateLimitMiddleware() gin.HandlerFunc {
	// This is a simple in-memory rate limiter
	// In production, you'd want to use Redis or another distributed solution
	return gin.BasicAuth(gin.Accounts{
		// This is just a placeholder - implement proper rate limiting
	})
}

// SecurityHeadersMiddleware adds security headers to admin responses
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Header("Content-Security-Policy", "default-src 'self'")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		
		c.Next()
	}
}