package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/phantom-sage/bankgo/internal/admin/interfaces"
)

// UserHandler implements the UserHandler interface
type UserHandler struct {
	userService interfaces.UserManagementService
}

// NewUserHandler creates a new user handler
func NewUserHandler(userService interfaces.UserManagementService) interfaces.UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

// RegisterRoutes registers HTTP routes for user management
func (h *UserHandler) RegisterRoutes(router gin.IRouter) {
	userGroup := router.Group("/users")
	{
		userGroup.GET("", h.ListUsers)
		userGroup.POST("", h.CreateUser)
		userGroup.GET("/:id", h.GetUser)
		userGroup.PUT("/:id", h.UpdateUser)
		userGroup.DELETE("/:id", h.DeleteUser)
		userGroup.POST("/:id/disable", h.DisableUser)
		userGroup.POST("/:id/enable", h.EnableUser)
	}
}

// ListUsers handles GET /api/admin/users
func (h *UserHandler) ListUsers(c *gin.Context) {
	var params interfaces.ListUsersParams

	// Parse pagination parameters
	if page := c.Query("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil {
			params.Page = p
		}
	}
	if pageSize := c.Query("page_size"); pageSize != "" {
		if ps, err := strconv.Atoi(pageSize); err == nil {
			params.PageSize = ps
		}
	}

	// Parse search parameter
	params.Search = c.Query("search")

	// Parse is_active filter
	if isActive := c.Query("is_active"); isActive != "" {
		if active, err := strconv.ParseBool(isActive); err == nil {
			params.IsActive = &active
		}
	}

	// Parse sorting parameters
	params.SortBy = c.DefaultQuery("sort_by", "created_at")
	if sortDesc := c.Query("sort_desc"); sortDesc != "" {
		if desc, err := strconv.ParseBool(sortDesc); err == nil {
			params.SortDesc = desc
		}
	}

	// Call service
	result, err := h.userService.ListUsers(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed_to_list_users",
			"message": "Failed to retrieve users",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetUser handles GET /api/admin/users/:id
func (h *UserHandler) GetUser(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_user_id",
			"message": "User ID is required",
		})
		return
	}

	user, err := h.userService.GetUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "user_not_found",
			"message": "User not found",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, user)
}

// CreateUser handles POST /api/admin/users
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req interfaces.CreateUserRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	user, err := h.userService.CreateUser(c.Request.Context(), req)
	if err != nil {
		// Check for duplicate email error
		if contains(err.Error(), "duplicate key") || contains(err.Error(), "unique constraint") {
			c.JSON(http.StatusConflict, gin.H{
				"error":   "email_already_exists",
				"message": "A user with this email already exists",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed_to_create_user",
			"message": "Failed to create user",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, user)
}

// UpdateUser handles PUT /api/admin/users/:id
func (h *UserHandler) UpdateUser(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_user_id",
			"message": "User ID is required",
		})
		return
	}

	var req interfaces.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	user, err := h.userService.UpdateUser(c.Request.Context(), userID, req)
	if err != nil {
		if contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "user_not_found",
				"message": "User not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed_to_update_user",
			"message": "Failed to update user",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, user)
}

// DisableUser handles POST /api/admin/users/:id/disable
func (h *UserHandler) DisableUser(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_user_id",
			"message": "User ID is required",
		})
		return
	}

	err := h.userService.DisableUser(c.Request.Context(), userID)
	if err != nil {
		if contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "user_not_found",
				"message": "User not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed_to_disable_user",
			"message": "Failed to disable user",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User disabled successfully",
	})
}

// EnableUser handles POST /api/admin/users/:id/enable
func (h *UserHandler) EnableUser(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_user_id",
			"message": "User ID is required",
		})
		return
	}

	err := h.userService.EnableUser(c.Request.Context(), userID)
	if err != nil {
		if contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "user_not_found",
				"message": "User not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed_to_enable_user",
			"message": "Failed to enable user",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User enabled successfully",
	})
}

// DeleteUser handles DELETE /api/admin/users/:id
func (h *UserHandler) DeleteUser(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_user_id",
			"message": "User ID is required",
		})
		return
	}

	err := h.userService.DeleteUser(c.Request.Context(), userID)
	if err != nil {
		if contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "user_not_found",
				"message": "User not found",
			})
			return
		}

		if contains(err.Error(), "cannot delete user") {
			c.JSON(http.StatusConflict, gin.H{
				"error":   "user_has_dependencies",
				"message": err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed_to_delete_user",
			"message": "Failed to delete user",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User deleted successfully",
	})
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || 
		(len(s) > len(substr) && 
			(s[:len(substr)] == substr || 
			 s[len(s)-len(substr):] == substr || 
			 containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}