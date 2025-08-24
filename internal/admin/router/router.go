package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/phantom-sage/bankgo/internal/admin/handlers"
	"github.com/phantom-sage/bankgo/internal/admin/interfaces"
	"github.com/phantom-sage/bankgo/internal/admin/middleware"
)

// Setup configures and returns the Gin router for the admin API
func Setup(handlers *handlers.Container, middleware *middleware.Container) *gin.Engine {
	// Set Gin mode based on environment
	if middleware.GetConfig().IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create router
	r := gin.New()

	// Add global middleware
	r.Use(gin.Recovery())
	r.Use(gin.Logger())
	
	// Add CORS middleware for admin SPA communication
	if middleware.CORSMiddleware != nil {
		r.Use(middleware.CORSMiddleware.Handler())
	}

	// Health check endpoint (no auth required)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "admin-api",
			"timestamp": gin.H{"now": "timestamp"},
		})
	})

	// API routes group
	api := r.Group("/api/admin")
	
	// Authentication routes (no auth middleware required)
	if handlers.AuthHandler != nil {
		handlers.AuthHandler.RegisterRoutes(api)
	}

	// Protected routes group (requires authentication)
	protected := api.Group("")
	if middleware.AuthMiddleware != nil {
		protected.Use(middleware.AuthMiddleware.Handler())
	}

	// Session validation endpoint for frontend authentication checks
	protected.GET("/validate", func(c *gin.Context) {
		// This endpoint is protected by auth middleware
		// If we reach here, the session is valid
		session, exists := GetAdminSession(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "session_not_found",
				"message": "Session information not available",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"valid":   true,
			"session": session,
		})
	})

	// Register user management routes
	if handlers.UserHandler != nil {
		handlers.UserHandler.RegisterRoutes(protected)
	}

	// Register system monitoring routes
	if handlers.SystemHandler != nil {
		handlers.SystemHandler.RegisterRoutes(protected)
	}

	// Register WebSocket routes
	if handlers.WebSocketHandler != nil {
		handlers.WebSocketHandler.RegisterRoutes(protected)
	}

	// TODO: Register other protected routes when handlers are implemented
	// protected.GET("/database/tables", handlers.DatabaseHandler.ListTables)

	return r
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