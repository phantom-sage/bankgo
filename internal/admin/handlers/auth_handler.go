package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/phantom-sage/bankgo/internal/admin/interfaces"
	"github.com/phantom-sage/bankgo/internal/admin/middleware"
)

// AuthHandlerImpl implements the AuthHandler interface
type AuthHandlerImpl struct {
	authService interfaces.AdminAuthService
}

// NewAuthHandler creates a new authentication handler
func NewAuthHandler(authService interfaces.AdminAuthService) interfaces.AuthHandler {
	return &AuthHandlerImpl{
		authService: authService,
	}
}

// RegisterRoutes registers authentication routes
func (h *AuthHandlerImpl) RegisterRoutes(router gin.IRouter) {
	auth := router.Group("/auth")
	{
		auth.POST("/login", h.Login)
		auth.POST("/logout", h.Logout)
		auth.GET("/session", h.ValidateSession)
		auth.PUT("/credentials", h.UpdateCredentials)
	}
}

// LoginRequest represents the login request payload
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse represents the login response
type LoginResponse struct {
	Success     bool                     `json:"success"`
	Message     string                   `json:"message"`
	Session     *interfaces.AdminSession `json:"session,omitempty"`
	ExpiresIn   int64                    `json:"expires_in,omitempty"` // seconds until expiration
}

// UpdateCredentialsRequest represents the credential update request
type UpdateCredentialsRequest struct {
	Username    string `json:"username" binding:"required"`
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

// Login handles admin login requests
func (h *AuthHandlerImpl) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "validation_error",
			"message": "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Authenticate user
	session, err := h.authService.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "authentication_failed",
			"message": "Invalid credentials",
		})
		return
	}

	// Calculate expires in seconds
	expiresIn := time.Until(session.ExpiresAt).Seconds()

	// Set token in cookie for browser clients
	c.SetCookie(
		"admin_token",
		session.PasetoToken,
		int(expiresIn),
		"/",
		"",
		false, // secure - set to true in production with HTTPS
		true,  // httpOnly
	)

	// Return success response
	c.JSON(http.StatusOK, LoginResponse{
		Success:   true,
		Message:   "Login successful",
		Session:   session,
		ExpiresIn: int64(expiresIn),
	})
}

// Logout handles admin logout requests
func (h *AuthHandlerImpl) Logout(c *gin.Context) {
	// Extract token from request
	token := h.extractToken(c)
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "validation_error",
			"message": "No token provided",
		})
		return
	}

	// Logout (invalidate session)
	err := h.authService.Logout(c.Request.Context(), token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to logout",
		})
		return
	}

	// Clear cookie
	c.SetCookie(
		"admin_token",
		"",
		-1,
		"/",
		"",
		false,
		true,
	)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Logout successful",
	})
}

// ValidateSession validates the current session and returns session info
func (h *AuthHandlerImpl) ValidateSession(c *gin.Context) {
	// Try to get session from middleware first
	if session, exists := middleware.GetAdminSession(c); exists {
		// Session already validated by middleware
		expiresIn := time.Until(session.ExpiresAt).Seconds()
		c.JSON(http.StatusOK, gin.H{
			"valid":      true,
			"session":    session,
			"expires_in": int64(expiresIn),
		})
		return
	}

	// If no middleware session, try to validate token directly
	token := h.extractToken(c)
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"valid":   false,
			"error":   "no_token",
			"message": "No authentication token provided",
		})
		return
	}

	session, err := h.authService.ValidateSession(c.Request.Context(), token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"valid":   false,
			"error":   "invalid_token",
			"message": "Invalid or expired token",
		})
		return
	}

	expiresIn := time.Until(session.ExpiresAt).Seconds()
	c.JSON(http.StatusOK, gin.H{
		"valid":      true,
		"session":    session,
		"expires_in": int64(expiresIn),
	})
}

// UpdateCredentials handles admin credential update requests
func (h *AuthHandlerImpl) UpdateCredentials(c *gin.Context) {
	var req UpdateCredentialsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "validation_error",
			"message": "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Update credentials
	err := h.authService.UpdateCredentials(
		c.Request.Context(),
		req.Username,
		req.OldPassword,
		req.NewPassword,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "credential_update_failed",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Credentials updated successfully. Please log in again with your new credentials.",
	})
}

// extractToken extracts the PASETO token from the request
func (h *AuthHandlerImpl) extractToken(c *gin.Context) string {
	// Try Authorization header first (Bearer token)
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			return authHeader[7:]
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