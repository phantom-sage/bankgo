package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/phantom-sage/bankgo/internal/admin/interfaces"
)

// CORSMiddlewareImpl implements the CORSMiddleware interface
type CORSMiddlewareImpl struct {
	allowedOrigins []string
}

// NewCORSMiddleware creates a new CORS middleware
func NewCORSMiddleware(allowedOrigins []string) interfaces.CORSMiddleware {
	return &CORSMiddlewareImpl{
		allowedOrigins: allowedOrigins,
	}
}

// Handler returns the Gin middleware handler function
func (m *CORSMiddlewareImpl) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		
		// Check if origin is allowed
		allowed := false
		for _, allowedOrigin := range m.allowedOrigins {
			if allowedOrigin == "*" || allowedOrigin == origin {
				allowed = true
				break
			}
		}

		// Set CORS headers
		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400")

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// IsOriginAllowed checks if an origin is allowed
func (m *CORSMiddlewareImpl) IsOriginAllowed(origin string) bool {
	for _, allowedOrigin := range m.allowedOrigins {
		if allowedOrigin == "*" || allowedOrigin == origin {
			return true
		}
		
		// Support wildcard subdomains (e.g., *.example.com)
		if strings.HasPrefix(allowedOrigin, "*.") {
			domain := allowedOrigin[2:]
			if strings.HasSuffix(origin, "."+domain) || origin == domain {
				return true
			}
		}
	}
	return false
}

// GetAllowedOrigins returns the list of allowed origins
func (m *CORSMiddlewareImpl) GetAllowedOrigins() []string {
	return m.allowedOrigins
}

// SetAllowedOrigins updates the list of allowed origins
func (m *CORSMiddlewareImpl) SetAllowedOrigins(origins []string) {
	m.allowedOrigins = origins
}

// AddAllowedOrigin adds a new allowed origin
func (m *CORSMiddlewareImpl) AddAllowedOrigin(origin string) {
	m.allowedOrigins = append(m.allowedOrigins, origin)
}

// RemoveAllowedOrigin removes an allowed origin
func (m *CORSMiddlewareImpl) RemoveAllowedOrigin(origin string) {
	for i, allowedOrigin := range m.allowedOrigins {
		if allowedOrigin == origin {
			m.allowedOrigins = append(m.allowedOrigins[:i], m.allowedOrigins[i+1:]...)
			break
		}
	}
}