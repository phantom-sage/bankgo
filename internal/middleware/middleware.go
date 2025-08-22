package middleware

import (
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/phantom-sage/bankgo/internal/logging"
)

// SetupMiddleware configures all middleware for the Gin engine
func SetupMiddleware(r *gin.Engine, loggerManager *logging.LoggerManager) {
	// Request ID middleware (should be first to ensure all logs have request ID)
	r.Use(RequestID())
	
	// Request logging middleware (early to capture all requests)
	r.Use(RequestLogger(DefaultLoggerConfig(loggerManager)))
	
	// CORS middleware (configure based on environment)
	corsConfig := getCORSConfig()
	r.Use(CORS(corsConfig))
	
	// Rate limiting middleware (apply to all routes)
	r.Use(RateLimit(DefaultRateLimiterConfig()))
	
	// Error handling middleware (should be after other middleware to catch their errors)
	r.Use(ErrorHandler())
	
	// Recovery middleware to handle panics (should be last)
	r.Use(gin.Recovery())
}

// SetupAuthMiddleware configures middleware specifically for authentication routes
func SetupAuthMiddleware(r *gin.RouterGroup) {
	// Stricter rate limiting for auth endpoints
	r.Use(RateLimitByIP(AuthRateLimiterConfig()))
}

// SetupAPIMiddleware configures middleware for API routes
func SetupAPIMiddleware(r *gin.RouterGroup) {
	// Standard rate limiting for API endpoints
	r.Use(RateLimit(StrictRateLimiterConfig()))
}

// getCORSConfig returns CORS configuration based on environment
func getCORSConfig() CORSConfig {
	env := os.Getenv("GIN_MODE")
	allowedOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
	
	if env == "release" && allowedOrigins != "" {
		// Production: use restrictive CORS with specified origins
		origins := strings.Split(allowedOrigins, ",")
		for i, origin := range origins {
			origins[i] = strings.TrimSpace(origin)
		}
		return RestrictiveCORSConfig(origins)
	}
	
	// Development: use permissive CORS
	return DefaultCORSConfig()
}